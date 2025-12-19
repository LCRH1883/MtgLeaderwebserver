package adminui

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"MtgLeaderwebserver/internal/auth"
	"MtgLeaderwebserver/internal/domain"
	"MtgLeaderwebserver/internal/service"
)

type Opts struct {
	Logger *slog.Logger

	Auth         *service.AuthService
	Admin        *service.AdminService
	CookieCodec  auth.CookieCodec
	CookieSecure bool
	SessionTTL   time.Duration
	AdminEmails  []string
}

func New(opts Opts) http.Handler {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	adminSet := make(map[string]bool, len(opts.AdminEmails))
	for _, e := range opts.AdminEmails {
		e = strings.TrimSpace(strings.ToLower(e))
		if e != "" {
			adminSet[e] = true
		}
	}

	if len(adminSet) == 0 || opts.Auth == nil || opts.Admin == nil {
		return http.NotFoundHandler()
	}

	app := &app{
		logger:       logger,
		authSvc:      opts.Auth,
		adminSvc:     opts.Admin,
		cookieCodec:  opts.CookieCodec,
		cookieSecure: opts.CookieSecure,
		sessionTTL:   opts.SessionTTL,
		adminEmails:  adminSet,
	}

	t, err := parseTemplates()
	if err != nil {
		logger.Error("adminui: parse templates failed", "err", err)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		})
	}
	app.templates = t

	mux := http.NewServeMux()
	mux.HandleFunc("GET /admin", app.redirectAdmin)
	mux.HandleFunc("GET /admin/", app.requireAdmin(app.handleDashboard))
	mux.HandleFunc("GET /admin/login", app.handleLoginGet)
	mux.HandleFunc("POST /admin/login", app.handleLoginPost)
	mux.HandleFunc("POST /admin/logout", app.handleLogoutPost)
	mux.HandleFunc("GET /admin/users", app.requireAdmin(app.handleUsersList))
	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		logger.Error("adminui: static fs setup failed", "err", err)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		})
	}
	static := http.StripPrefix("/admin/static/", http.FileServer(http.FS(staticFS)))
	mux.Handle("GET /admin/static/", static)
	mux.Handle("HEAD /admin/static/", static)

	return mux
}

type app struct {
	logger *slog.Logger

	authSvc  *service.AuthService
	adminSvc *service.AdminService

	cookieCodec  auth.CookieCodec
	cookieSecure bool
	sessionTTL   time.Duration
	adminEmails  map[string]bool

	templates *templates
}

func (a *app) redirectAdmin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin/", http.StatusFound)
}

func (a *app) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, _, ok := a.currentUser(r)
		if !ok {
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}
		if !a.adminEmails[strings.ToLower(u.Email)] {
			a.templates.renderError(w, http.StatusForbidden, "Forbidden", "This account is not allowed to access admin.")
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
