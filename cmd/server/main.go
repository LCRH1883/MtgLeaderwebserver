package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log/slog"

	"MtgLeaderwebserver/internal/adminui"
	"MtgLeaderwebserver/internal/auth"
	"MtgLeaderwebserver/internal/config"
	"MtgLeaderwebserver/internal/domain"
	"MtgLeaderwebserver/internal/httpapi"
	"MtgLeaderwebserver/internal/service"
	"MtgLeaderwebserver/internal/store/postgres"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	logger := newLogger(cfg)

	var (
		authSvc    *service.AuthService
		friendsSvc *service.FriendsService
		matchSvc   *service.MatchService
		usersSvc   *service.UsersService
		adminSvc   *service.AdminService
		dbPing     func(context.Context) error
	)

	if cfg.DBDSN != "" {
		pgPool, err := postgres.Open(context.Background(), cfg.DBDSN)
		if err != nil {
			logger.Error("db open failed", "err", err)
			os.Exit(1)
		}
		defer pgPool.Close()

		users := postgres.NewUsersStore(pgPool)
		sessions := postgres.NewSessionsStore(pgPool)
		friendships := postgres.NewFriendshipsStore(pgPool)
		matches := postgres.NewMatchesStore(pgPool)
		userSearch := postgres.NewUserSearchStore(pgPool)
		adminUsers := postgres.NewAdminUsersStore(pgPool)

		if err := bootstrapAdminUser(context.Background(), logger, users, cfg.AdminBootstrapEmail, cfg.AdminBootstrapUsername, cfg.AdminBootstrapPassword); err != nil {
			logger.Error("bootstrap admin failed", "err", err)
			os.Exit(1)
		}

		authSvc = &service.AuthService{
			Users:      users,
			Sessions:   sessions,
			SessionTTL: cfg.SessionTTL,
		}
		friendsSvc = &service.FriendsService{
			Users:       users,
			Friendships: friendships,
		}
		matchSvc = &service.MatchService{
			Matches: matches,
			Friends: friendsSvc,
		}
		usersSvc = &service.UsersService{Store: userSearch}
		adminSvc = &service.AdminService{Users: adminUsers}
		dbPing = pgPool.Ping
	}

	apiRouter := httpapi.NewRouter(httpapi.RouterOpts{
		Logger:       logger,
		IsProd:       cfg.IsProd(),
		DBPing:       dbPing,
		Auth:         authSvc,
		Friends:      friendsSvc,
		Matches:      matchSvc,
		Users:        usersSvc,
		CookieCodec:  auth.NewCookieCodec([]byte(cfg.CookieSecret)),
		CookieSecure: cfg.CookieSecure(),
		SessionTTL:   cfg.SessionTTL,
	})

	root := http.NewServeMux()
	root.Handle("/", apiRouter)

	if adminSvc != nil && authSvc != nil && len(cfg.AdminEmails) > 0 {
		logger.Info("admin ui enabled", "admin_emails", len(cfg.AdminEmails))
		adminRouter := adminui.New(adminui.Opts{
			Logger:       logger,
			Auth:         authSvc,
			Admin:        adminSvc,
			CookieCodec:  auth.NewCookieCodec([]byte(cfg.CookieSecret)),
			CookieSecure: cfg.CookieSecure(),
			SessionTTL:   cfg.SessionTTL,
			AdminEmails:  cfg.AdminEmails,
		})
		root.Handle("/admin", adminRouter)
		root.Handle("/admin/", adminRouter)
	} else {
		logger.Info("admin ui disabled", "admin_emails", len(cfg.AdminEmails), "db_enabled", cfg.DBDSN != "")
		root.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/admin/", http.StatusFound)
		})
		root.HandleFunc("/admin/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("admin ui disabled: set APP_DB_DSN and APP_ADMIN_EMAILS (and restart the server)\n"))
		})
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           root,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "env", cfg.Env, "addr", cfg.Addr)
		errCh <- srv.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-stop:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}
}

func bootstrapAdminUser(ctx context.Context, logger *slog.Logger, users *postgres.UsersStore, email, username, password string) error {
	if password == "" {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	if len(password) < 12 {
		return errors.New("APP_ADMIN_BOOTSTRAP_PASSWORD: must be at least 12 characters")
	}
	if email == "" || username == "" {
		return errors.New("admin bootstrap: email and username are required")
	}

	_, err := users.GetUserByEmail(ctx, email)
	if err == nil {
		logger.Info("admin bootstrap: user already exists", "email", email)
		return nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("admin bootstrap: lookup user: %w", err)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("admin bootstrap: hash password: %w", err)
	}

	_, err = users.CreateUser(ctx, email, username, hash)
	if err != nil {
		if errors.Is(err, domain.ErrEmailTaken) || errors.Is(err, domain.ErrUsernameTaken) {
			logger.Info("admin bootstrap: user already exists", "email", email)
			return nil
		}
		return fmt.Errorf("admin bootstrap: create user: %w", err)
	}

	logger.Info("admin bootstrap: created admin user", "email", email)
	return nil
}

func newLogger(cfg config.Config) *slog.Logger {
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "info", "":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}
	if cfg.IsProd() {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}
