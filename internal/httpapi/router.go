package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"MtgLeaderwebserver/internal/auth"
	"MtgLeaderwebserver/internal/service"
)

type RouterOpts struct {
	Logger *slog.Logger
	IsProd bool

	DBPing func(context.Context) error

	Auth         *service.AuthService
	Friends      *service.FriendsService
	Matches      *service.MatchService
	Users        *service.UsersService
	Profile      *service.ProfileService
	CookieCodec  auth.CookieCodec
	CookieSecure bool
	SessionTTL   time.Duration
	AvatarDir    string
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
		logger:       logger,
		isProd:       opts.IsProd,
		dbPing:       opts.DBPing,
		authSvc:      opts.Auth,
		friendsSvc:   opts.Friends,
		matchSvc:     opts.Matches,
		usersSvc:     opts.Users,
		profileSvc:   opts.Profile,
		avatarDir:    opts.AvatarDir,
		cookieCodec:  opts.CookieCodec,
		cookieSecure: opts.CookieSecure,
		sessionTTL:   opts.SessionTTL,
		loginLimiter: newLoginLimiter(),
	}

	publicMux := http.NewServeMux()
	apiMux := http.NewServeMux()

	publicMux.HandleFunc("GET /", api.handleHome)
	publicMux.HandleFunc("GET /privacy", api.handlePrivacyWeb)
	publicMux.HandleFunc("GET /privacy/android", api.handlePrivacyAndroid)
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
		apiMux.HandleFunc("POST /v1/auth/logout", api.requireAuth(api.handleAuthLogout))
		apiMux.HandleFunc("GET /v1/users/me", api.requireAuth(api.handleUsersMe))
		if api.profileSvc != nil {
			apiMux.HandleFunc("PATCH /v1/users/me", api.requireAuth(api.handleUsersMeUpdate))
			apiMux.HandleFunc("POST /v1/users/me/avatar", api.requireAuth(api.handleUsersMeAvatar))
		}
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
	}

	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, pattern := apiMux.Handler(r)
		if pattern == "" {
			handleV1NotFound(w, r)
			return
		}
		h.ServeHTTP(w, r)
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

type api struct {
	logger *slog.Logger
	isProd bool

	dbPing func(context.Context) error

	authSvc      *service.AuthService
	friendsSvc   *service.FriendsService
	matchSvc     *service.MatchService
	usersSvc     *service.UsersService
	profileSvc   *service.ProfileService
	avatarDir    string
	cookieCodec  auth.CookieCodec
	cookieSecure bool
	sessionTTL   time.Duration

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
