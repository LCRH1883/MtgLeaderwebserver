package userui

import (
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"MtgLeaderwebserver/internal/auth"
	"MtgLeaderwebserver/internal/domain"
	"MtgLeaderwebserver/internal/service"
)

type Opts struct {
	Logger *slog.Logger

	Auth         *service.AuthService
	Friends      *service.FriendsService
	Users        *service.UsersService
	Reset        *service.PasswordResetService
	CookieCodec  auth.CookieCodec
	CookieSecure bool
	SessionTTL   time.Duration
}

func New(opts Opts) http.Handler {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	if opts.Auth == nil || opts.Friends == nil || opts.Users == nil {
		logger.Warn("userui: missing services", "auth", opts.Auth != nil, "friends", opts.Friends != nil, "users", opts.Users != nil)
	}

	app := &app{
		logger:       logger,
		authSvc:      opts.Auth,
		friendsSvc:   opts.Friends,
		usersSvc:     opts.Users,
		resetSvc:     opts.Reset,
		cookieCodec:  opts.CookieCodec,
		cookieSecure: opts.CookieSecure,
		sessionTTL:   opts.SessionTTL,
	}

	t, err := parseTemplates()
	if err != nil {
		logger.Error("userui: parse templates failed", "err", err)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		})
	}
	app.templates = t

	mux := http.NewServeMux()
	mux.HandleFunc("GET /app", app.redirectApp)
	mux.HandleFunc("GET /app/", app.requireAuth(app.handleHome))
	mux.HandleFunc("GET /app/login", app.handleLoginGet)
	mux.HandleFunc("POST /app/login", app.handleLoginPost)
	mux.HandleFunc("GET /app/register", app.handleRegisterGet)
	mux.HandleFunc("POST /app/register", app.handleRegisterPost)
	mux.HandleFunc("GET /app/reset", app.handleResetGet)
	mux.HandleFunc("POST /app/reset", app.handleResetPost)
	mux.HandleFunc("POST /app/logout", app.handleLogoutPost)
	mux.HandleFunc("POST /app/friends/requests", app.requireAuth(app.handleFriendRequest))
	mux.HandleFunc("POST /app/friends/requests/accept", app.requireAuth(app.handleFriendAccept))
	mux.HandleFunc("POST /app/friends/requests/decline", app.requireAuth(app.handleFriendDecline))
	mux.HandleFunc("POST /app/friends/requests/cancel", app.requireAuth(app.handleFriendCancel))

	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		logger.Error("userui: static fs setup failed", "err", err)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		})
	}
	static := http.StripPrefix("/app/static/", http.FileServer(http.FS(staticFS)))
	mux.Handle("GET /app/static/", static)
	mux.Handle("HEAD /app/static/", static)

	return mux
}

type app struct {
	logger *slog.Logger

	authSvc    *service.AuthService
	friendsSvc *service.FriendsService
	usersSvc   *service.UsersService
	resetSvc   *service.PasswordResetService

	cookieCodec  auth.CookieCodec
	cookieSecure bool
	sessionTTL   time.Duration

	templates *templates
}

func (a *app) redirectApp(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/app/", http.StatusFound)
}

func (a *app) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _, ok := a.currentUser(r)
		if !ok {
			http.Redirect(w, r, "/app/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (a *app) currentUser(r *http.Request) (domain.User, string, bool) {
	if a.authSvc == nil {
		return domain.User{}, "", false
	}
	c, err := r.Cookie(auth.SessionCookieName)
	if err != nil || c.Value == "" {
		return domain.User{}, "", false
	}
	sessID, ok := a.cookieCodec.DecodeSessionID(c.Value)
	if !ok {
		return domain.User{}, "", false
	}
	u, err := a.authSvc.GetUserForSession(r.Context(), sessID)
	if err != nil {
		return domain.User{}, "", false
	}
	return u, sessID, true
}
