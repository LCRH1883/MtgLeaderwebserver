package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log/slog"

	"MtgLeaderwebserver/internal/auth"
	"MtgLeaderwebserver/internal/config"
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
		dbPing = pgPool.Ping
	}

	router := httpapi.NewRouter(httpapi.RouterOpts{
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

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
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
