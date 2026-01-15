package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"log/slog"

	"MtgLeaderwebserver/internal/adminui"
	"MtgLeaderwebserver/internal/auth"
	"MtgLeaderwebserver/internal/config"
	"MtgLeaderwebserver/internal/domain"
	"MtgLeaderwebserver/internal/httpapi"
	"MtgLeaderwebserver/internal/notifications"
	"MtgLeaderwebserver/internal/service"
	"MtgLeaderwebserver/internal/store/postgres"
	"MtgLeaderwebserver/internal/userui"
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
		resetSvc   *service.PasswordResetService
		emailSvc   *service.EmailService
		profileSvc *service.ProfileService
		notifySvc  *service.NotificationService
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
		adminSettings := postgres.NewAdminSettingsStore(pgPool)
		passwordResets := postgres.NewPasswordResetStore(pgPool)
		notificationTokens := postgres.NewNotificationTokensStore(pgPool)

		if err := bootstrapAdminUser(context.Background(), logger, users, cfg.AdminBootstrapEmail, cfg.AdminBootstrapUsername, cfg.AdminBootstrapPassword); err != nil {
			logger.Error("bootstrap admin failed", "err", err)
			os.Exit(1)
		}

		authSvc = &service.AuthService{
			Users:             users,
			Sessions:          sessions,
			SessionTTL:        cfg.SessionTTL,
			GoogleWebClientID: cfg.GoogleWebClientID,
			AppleServiceID:    cfg.AppleServiceID,
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
		resetSvc = &service.PasswordResetService{
			Store: passwordResets,
			Users: users,
		}
		emailSvc = &service.EmailService{Settings: adminSettings}
		profileSvc = &service.ProfileService{Store: users}
		notifySvc = &service.NotificationService{
			Tokens: notificationTokens,
			Users:  users,
			Logger: logger,
		}
		if cfg.FCMProjectID != "" || cfg.FCMCredentialsPath != "" {
			sender, err := notifications.NewFCMSender(context.Background(), cfg.FCMProjectID, cfg.FCMCredentialsPath)
			if err != nil {
				logger.Error("fcm sender init failed", "err", err)
			} else {
				notifySvc.Sender = sender
			}
		}
		if friendsSvc != nil && notifySvc != nil {
			friendsSvc.Notifier = notifySvc
		}
		dbPing = pgPool.Ping
	}

	apiRouter := httpapi.NewRouter(httpapi.RouterOpts{
		Logger:        logger,
		IsProd:        cfg.IsProd(),
		DBPing:        dbPing,
		Auth:          authSvc,
		Friends:       friendsSvc,
		Matches:       matchSvc,
		Users:         usersSvc,
		Profile:       profileSvc,
		Reset:         resetSvc,
		Email:         emailSvc,
		Notifications: notifySvc,
		CookieCodec:   auth.NewCookieCodec([]byte(cfg.CookieSecret)),
		CookieSecure:  cfg.CookieSecure(),
		SessionTTL:    cfg.SessionTTL,
		AvatarDir:     cfg.AvatarDir,
		PublicURL:     cfg.PublicURL,
	})

	root := http.NewServeMux()

	imgDir := "img"
	iconSource := path.Join(imgDir, "wizard_icon.png")
	iconSvc := &iconService{
		sourcePath: iconSource,
	}

	root.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		// Browsers still request /favicon.ico by default. Serve a PNG anyway.
		iconSvc.ServeNamedPNG(w, r, "favicon-32.png")
	})
	root.HandleFunc("GET /icon/{name}", func(w http.ResponseWriter, r *http.Request) {
		iconSvc.ServeNamedPNG(w, r, r.PathValue("name"))
	})

	root.HandleFunc("GET /img/index.json", func(w http.ResponseWriter, r *http.Request) {
		entries, err := os.ReadDir(imgDir)
		if err != nil {
			http.Error(w, "img directory not available", http.StatusNotFound)
			return
		}

		var files []string
		for _, ent := range entries {
			if ent.IsDir() {
				continue
			}
			name := ent.Name()
			ext := strings.ToLower(path.Ext(name))
			switch ext {
			case ".png", ".jpg", ".jpeg", ".webp":
				files = append(files, "/img/"+url.PathEscape(name))
			default:
				continue
			}
		}
		sort.Strings(files)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_ = json.NewEncoder(w).Encode(map[string]any{"images": files})
	})

	imgFS := http.StripPrefix("/img/", http.FileServer(http.Dir(imgDir)))
	root.Handle("/img/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/img" || r.URL.Path == "/img/" {
			http.NotFound(w, r)
			return
		}
		imgFS.ServeHTTP(w, r)
	}))

	root.Handle("/", apiRouter)

	if adminSvc != nil && authSvc != nil && len(cfg.AdminEmails) > 0 {
		logger.Info("admin ui enabled", "admin_emails", len(cfg.AdminEmails))
		adminRouter := adminui.New(adminui.Opts{
			Logger:       logger,
			Auth:         authSvc,
			Admin:        adminSvc,
			Reset:        resetSvc,
			Email:        emailSvc,
			CookieCodec:  auth.NewCookieCodec([]byte(cfg.CookieSecret)),
			CookieSecure: cfg.CookieSecure(),
			SessionTTL:   cfg.SessionTTL,
			AdminEmails:  cfg.AdminEmails,
			GlobalAdmin:  cfg.AdminBootstrapEmail,
			PublicURL:    cfg.PublicURL,
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

	userRouter := userui.New(userui.Opts{
		Logger:       logger,
		Auth:         authSvc,
		Friends:      friendsSvc,
		Users:        usersSvc,
		Matches:      matchSvc,
		Reset:        resetSvc,
		Profile:      profileSvc,
		AvatarDir:    cfg.AvatarDir,
		CookieCodec:  auth.NewCookieCodec([]byte(cfg.CookieSecret)),
		CookieSecure: cfg.CookieSecure(),
		SessionTTL:   cfg.SessionTTL,
	})
	root.Handle("/app", userRouter)
	root.Handle("/app/", userRouter)

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

type iconService struct {
	sourcePath string

	mu       sync.Mutex
	cachedAt time.Time
	srcMod   time.Time
	cache    map[string][]byte
}

func (s *iconService) ServeNamedPNG(w http.ResponseWriter, r *http.Request, name string) {
	size, ok := iconSizeForName(name)
	if !ok {
		http.NotFound(w, r)
		return
	}

	b, mod, err := s.pngBytes(name, size)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
	w.Header().Set("Last-Modified", mod.UTC().Format(http.TimeFormat))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func iconSizeForName(name string) (int, bool) {
	switch name {
	case "favicon-16.png":
		return 16, true
	case "favicon-32.png":
		return 32, true
	case "apple-touch-icon.png":
		return 180, true
	case "android-chrome-192.png":
		return 192, true
	case "android-chrome-512.png":
		return 512, true
	default:
		return 0, false
	}
}

func (s *iconService) pngBytes(name string, size int) ([]byte, time.Time, error) {
	fi, err := os.Stat(s.sourcePath)
	if err != nil {
		return nil, time.Time{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cache == nil {
		s.cache = make(map[string][]byte)
	}
	if !s.srcMod.Equal(fi.ModTime()) {
		s.cache = make(map[string][]byte)
		s.srcMod = fi.ModTime()
		s.cachedAt = time.Now()
	}
	if b, ok := s.cache[name]; ok {
		return b, s.srcMod, nil
	}

	srcBytes, err := os.ReadFile(s.sourcePath)
	if err != nil {
		return nil, time.Time{}, err
	}
	srcImg, err := png.Decode(bytes.NewReader(srcBytes))
	if err != nil {
		return nil, time.Time{}, err
	}

	dst := resizeBilinear(srcImg, size)
	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return nil, time.Time{}, err
	}
	b := out.Bytes()
	s.cache[name] = b
	return b, s.srcMod, nil
}

func resizeBilinear(src image.Image, size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	sb := src.Bounds()
	sw := sb.Dx()
	sh := sb.Dy()
	if sw <= 0 || sh <= 0 {
		return dst
	}

	if size == 1 {
		dst.Set(0, 0, src.At(sb.Min.X, sb.Min.Y))
		return dst
	}

	for y := 0; y < size; y++ {
		fy := float64(y) * float64(sh-1) / float64(size-1)
		y0 := int(math.Floor(fy))
		y1 := y0 + 1
		if y1 >= sh {
			y1 = sh - 1
		}
		wy := fy - float64(y0)

		for x := 0; x < size; x++ {
			fx := float64(x) * float64(sw-1) / float64(size-1)
			x0 := int(math.Floor(fx))
			x1 := x0 + 1
			if x1 >= sw {
				x1 = sw - 1
			}
			wx := fx - float64(x0)

			r00, g00, b00, a00 := src.At(sb.Min.X+x0, sb.Min.Y+y0).RGBA()
			r10, g10, b10, a10 := src.At(sb.Min.X+x1, sb.Min.Y+y0).RGBA()
			r01, g01, b01, a01 := src.At(sb.Min.X+x0, sb.Min.Y+y1).RGBA()
			r11, g11, b11, a11 := src.At(sb.Min.X+x1, sb.Min.Y+y1).RGBA()

			r0 := (1-wx)*float64(r00) + wx*float64(r10)
			r1 := (1-wx)*float64(r01) + wx*float64(r11)
			g0 := (1-wx)*float64(g00) + wx*float64(g10)
			g1 := (1-wx)*float64(g01) + wx*float64(g11)
			b0 := (1-wx)*float64(b00) + wx*float64(b10)
			b1 := (1-wx)*float64(b01) + wx*float64(b11)
			a0 := (1-wx)*float64(a00) + wx*float64(a10)
			a1 := (1-wx)*float64(a01) + wx*float64(a11)

			r := (1-wy)*r0 + wy*r1
			g := (1-wy)*g0 + wy*g1
			b := (1-wy)*b0 + wy*b1
			a := (1-wy)*a0 + wy*a1

			dst.SetRGBA(x, y, rgba64ToRGBA(r, g, b, a))
		}
	}
	return dst
}

func rgba64ToRGBA(r, g, b, a float64) color.RGBA {
	return color.RGBA{
		R: uint8(clampTo8(r)),
		G: uint8(clampTo8(g)),
		B: uint8(clampTo8(b)),
		A: uint8(clampTo8(a)),
	}
}

func clampTo8(v float64) uint32 {
	if v < 0 {
		return 0
	}
	if v > 65535 {
		return 255
	}
	return uint32(v+0.5) >> 8
}

func bootstrapAdminUser(ctx context.Context, logger *slog.Logger, users *postgres.UsersStore, email, username, password string) error {
	if password == "" {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	if len(password) < 12 {
		if password != "admin" {
			return errors.New("APP_ADMIN_BOOTSTRAP_PASSWORD: must be at least 12 characters")
		}
		logger.Warn("admin bootstrap: weak password in use", "email", email)
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
