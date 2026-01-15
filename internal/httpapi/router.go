package httpapi

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"MtgLeaderwebserver/internal/auth"
	"MtgLeaderwebserver/internal/service"
)

type RouterOpts struct {
	Logger *slog.Logger
	IsProd bool

	DBPing func(context.Context) error

	Auth          *service.AuthService
	Friends       *service.FriendsService
	Matches       *service.MatchService
	Users         *service.UsersService
	Profile       *service.ProfileService
	Reset         *service.PasswordResetService
	Email         *service.EmailService
	Notifications *service.NotificationService
	CookieCodec   auth.CookieCodec
	CookieSecure  bool
	SessionTTL    time.Duration
	AvatarDir     string
	PublicURL     *url.URL
}

func NewRouter(opts RouterOpts) http.Handler {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	if opts.AvatarDir == "" {
		opts.AvatarDir = "data/avatars"
	}

	api := &api{
		logger:           logger,
		isProd:           opts.IsProd,
		dbPing:           opts.DBPing,
		authSvc:          opts.Auth,
		friendsSvc:       opts.Friends,
		matchSvc:         opts.Matches,
		usersSvc:         opts.Users,
		profileSvc:       opts.Profile,
		resetSvc:         opts.Reset,
		emailSvc:         opts.Email,
		notificationsSvc: opts.Notifications,
		avatarDir:        opts.AvatarDir,
		publicURL:        opts.PublicURL,
		cookieCodec:      opts.CookieCodec,
		cookieSecure:     opts.CookieSecure,
		sessionTTL:       opts.SessionTTL,
		loginLimiter:     newLoginLimiter(),
	}

	publicMux := http.NewServeMux()
	apiMux := http.NewServeMux()

	publicMux.HandleFunc("GET /", api.handleHome)
	publicMux.HandleFunc("GET /privacy", api.handlePrivacyWeb)
	publicMux.HandleFunc("GET /privacy/android", api.handlePrivacyAndroid)
	publicMux.HandleFunc("GET /privacy/apple", api.handlePrivacyApple)
	publicMux.HandleFunc("GET /wiki", api.handleWikiIndex)
	publicMux.HandleFunc("GET /wiki/", api.handleWikiIndex)
	publicMux.HandleFunc("GET /wiki/delete-account", api.handleWikiDeleteAccount)
	publicMux.HandleFunc("GET /wiki/delete-account/", api.handleWikiDeleteAccount)
	publicMux.HandleFunc("GET /wiki/mobile-app", api.handleWikiMobileApp)
	publicMux.HandleFunc("GET /wiki/mobile-app/", api.handleWikiMobileApp)
	publicMux.HandleFunc("GET /wiki/web-ui", api.handleWikiWebUI)
	publicMux.HandleFunc("GET /wiki/web-ui/", api.handleWikiWebUI)
	publicMux.HandleFunc("GET /healthz", api.handleHealthz)

	if api.authSvc == nil {
		apiMux.HandleFunc("POST /v1/auth/register", handleNotImplemented)
		apiMux.HandleFunc("POST /v1/auth/login", handleNotImplemented)
		apiMux.HandleFunc("POST /v1/auth/google", handleNotImplemented)
		apiMux.HandleFunc("POST /v1/auth/apple", handleNotImplemented)
		apiMux.HandleFunc("POST /v1/auth/logout", handleNotImplemented)
		apiMux.HandleFunc("GET /v1/users/me", handleNotImplemented)
	} else {
		apiMux.HandleFunc("POST /v1/auth/register", api.handleAuthRegister)
		apiMux.HandleFunc("POST /v1/auth/login", api.handleAuthLogin)
		apiMux.HandleFunc("POST /v1/auth/google", api.handleAuthLoginGoogle)
		apiMux.HandleFunc("POST /v1/auth/apple", api.handleAuthLoginApple)
		if api.resetSvc != nil {
			apiMux.HandleFunc("POST /v1/auth/reset", api.handleAuthReset)
		}
		if api.resetSvc != nil && api.emailSvc != nil {
			apiMux.HandleFunc("POST /v1/auth/forgot", api.handleAuthForgot)
		}
		apiMux.HandleFunc("POST /v1/auth/logout", api.requireAuth(api.handleAuthLogout))
		apiMux.HandleFunc("GET /v1/users/me", api.requireAuth(api.handleUsersMe))
		apiMux.HandleFunc("GET /v1/users/me/", api.requireAuth(api.handleUsersMe))
		apiMux.HandleFunc("PATCH /v1/users/me", api.requireAuth(api.handleUsersMeUpdate))
		apiMux.HandleFunc("PUT /v1/users/me", api.requireAuth(api.handleUsersMeUpdate))
		apiMux.HandleFunc("POST /v1/users/me", api.requireAuth(api.handleUsersMeUpdate))
		apiMux.HandleFunc("PATCH /v1/users/me/", api.requireAuth(api.handleUsersMeUpdate))
		apiMux.HandleFunc("PUT /v1/users/me/", api.requireAuth(api.handleUsersMeUpdate))
		apiMux.HandleFunc("POST /v1/users/me/", api.requireAuth(api.handleUsersMeUpdate))
		apiMux.HandleFunc("DELETE /v1/users/me", api.requireAuth(api.handleUsersMeDelete))
		apiMux.HandleFunc("DELETE /v1/users/me/", api.requireAuth(api.handleUsersMeDelete))
		apiMux.HandleFunc("POST /v1/users/me/delete", api.requireAuth(api.handleUsersMeDelete))
		apiMux.HandleFunc("POST /v1/users/me/delete/", api.requireAuth(api.handleUsersMeDelete))
		apiMux.HandleFunc("DELETE /v1/users/me/delete", api.requireAuth(api.handleUsersMeDelete))
		apiMux.HandleFunc("DELETE /v1/users/me/delete/", api.requireAuth(api.handleUsersMeDelete))
		apiMux.HandleFunc("POST /v1/users/me/avatar", api.requireAuth(api.handleUsersMeAvatar))
		apiMux.HandleFunc("PATCH /v1/users/me/avatar", api.requireAuth(api.handleUsersMeAvatar))
		apiMux.HandleFunc("PUT /v1/users/me/avatar", api.requireAuth(api.handleUsersMeAvatar))
		apiMux.HandleFunc("POST /v1/users/me/avatar/", api.requireAuth(api.handleUsersMeAvatar))
		apiMux.HandleFunc("PATCH /v1/users/me/avatar/", api.requireAuth(api.handleUsersMeAvatar))
		apiMux.HandleFunc("PUT /v1/users/me/avatar/", api.requireAuth(api.handleUsersMeAvatar))
		if api.usersSvc != nil {
			apiMux.HandleFunc("GET /v1/users/search", api.requireAuth(api.handleUsersSearch))
		}

		if api.friendsSvc != nil {
			apiMux.HandleFunc("GET /v1/friends", api.requireAuth(api.handleFriendsList))
			apiMux.HandleFunc("GET /v1/friends/connections", api.requireAuth(api.handleFriendsConnections))
			apiMux.HandleFunc("POST /v1/friends/requests", api.requireAuth(api.handleFriendsCreateRequest))
			apiMux.HandleFunc("POST /v1/friends/requests/{id}/accept", api.requireAuth(api.handleFriendsAccept))
			apiMux.HandleFunc("POST /v1/friends/requests/{id}/decline", api.requireAuth(api.handleFriendsDecline))
			apiMux.HandleFunc("POST /v1/friends/requests/{id}/cancel", api.requireAuth(api.handleFriendsCancel))
			apiMux.HandleFunc("DELETE /v1/friends/{id}", api.requireAuth(api.handleFriendsRemove))
			apiMux.HandleFunc("POST /v1/friends/{id}/remove", api.requireAuth(api.handleFriendsRemove))
		}

		if api.matchSvc != nil {
			apiMux.HandleFunc("POST /v1/matches", api.requireAuth(api.handleMatchesCreate))
			apiMux.HandleFunc("GET /v1/matches", api.requireAuth(api.handleMatchesList))
			apiMux.HandleFunc("GET /v1/matches/{id}", api.requireAuth(api.handleMatchesGet))
			apiMux.HandleFunc("GET /v1/stats/summary", api.requireAuth(api.handleStatsSummary))
			apiMux.HandleFunc("GET /v1/stats/head-to-head/{id}", api.requireAuth(api.handleStatsHeadToHead))
			if api.friendsSvc != nil {
				apiMux.HandleFunc("GET /v1/stats/friends", api.requireAuth(api.handleStatsFriends))
			}
		}
		if api.notificationsSvc != nil {
			apiMux.HandleFunc("POST /v1/notifications/token", api.requireAuth(api.handleNotificationsTokenUpsert))
			apiMux.HandleFunc("DELETE /v1/notifications/token", api.requireAuth(api.handleNotificationsTokenDelete))
		}
	}

	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// We need ServeMux.ServeHTTP to run so it populates r.PathValue(...) for patterns
		// like "/v1/friends/requests/{id}/accept". We also want JSON 404s for API routes.
		brw := newBufferedResponseWriter()
		apiMux.ServeHTTP(brw, r)
		if brw.status == http.StatusNotFound {
			handleV1NotFound(w, r)
			return
		}
		brw.flushTo(w)
	})

	root := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/") || r.URL.Path == "/v1" {
			apiHandler.ServeHTTP(w, r)
			return
		}
		publicMux.ServeHTTP(w, r)
	})

	var h http.Handler = root
	h = RequestLogger(logger)(h)
	h = RequestID()(h)
	h = Recoverer(logger, opts.IsProd)(h)
	return h
}

func handleNotImplemented(w http.ResponseWriter, _ *http.Request) {
	WriteError(w, http.StatusNotImplemented, "not_implemented", "not implemented")
}

func handleV1NotFound(w http.ResponseWriter, _ *http.Request) {
	WriteError(w, http.StatusNotFound, "not_found", "not found")
}

type bufferedResponseWriter struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newBufferedResponseWriter() *bufferedResponseWriter {
	return &bufferedResponseWriter{
		header: make(http.Header),
		status: http.StatusOK,
	}
}

func (w *bufferedResponseWriter) Header() http.Header {
	return w.header
}

func (w *bufferedResponseWriter) WriteHeader(status int) {
	w.status = status
}

func (w *bufferedResponseWriter) Write(p []byte) (int, error) {
	return w.body.Write(p)
}

func (w *bufferedResponseWriter) flushTo(dst http.ResponseWriter) {
	for k, vv := range w.header {
		for _, v := range vv {
			dst.Header().Add(k, v)
		}
	}
	dst.WriteHeader(w.status)
	_, _ = dst.Write(w.body.Bytes())
}

type api struct {
	logger *slog.Logger
	isProd bool

	dbPing func(context.Context) error

	authSvc          *service.AuthService
	friendsSvc       *service.FriendsService
	matchSvc         *service.MatchService
	usersSvc         *service.UsersService
	profileSvc       *service.ProfileService
	resetSvc         *service.PasswordResetService
	emailSvc         *service.EmailService
	notificationsSvc *service.NotificationService
	avatarDir        string
	publicURL        *url.URL
	cookieCodec      auth.CookieCodec
	cookieSecure     bool
	sessionTTL       time.Duration

	loginLimiter *loginLimiter
}

func (a *api) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if a.dbPing != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()
		if err := a.dbPing(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("db down"))
			return
		}
	}

	_, _ = w.Write([]byte("ok"))
}
