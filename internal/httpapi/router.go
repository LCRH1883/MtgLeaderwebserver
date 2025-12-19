package httpapi

import (
	"context"
	"log/slog"
	"net/http"
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
	CookieCodec  auth.CookieCodec
	CookieSecure bool
	SessionTTL   time.Duration
}

func NewRouter(opts RouterOpts) http.Handler {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	api := &api{
		logger:       logger,
		isProd:       opts.IsProd,
		dbPing:       opts.DBPing,
		authSvc:      opts.Auth,
		friendsSvc:   opts.Friends,
		matchSvc:     opts.Matches,
		cookieCodec:  opts.CookieCodec,
		cookieSecure: opts.CookieSecure,
		sessionTTL:   opts.SessionTTL,
		loginLimiter: newLoginLimiter(),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", api.handleHealthz)

	if api.authSvc == nil {
		mux.HandleFunc("POST /v1/auth/register", handleNotImplemented)
		mux.HandleFunc("POST /v1/auth/login", handleNotImplemented)
		mux.HandleFunc("POST /v1/auth/logout", handleNotImplemented)
		mux.HandleFunc("GET /v1/users/me", handleNotImplemented)
	} else {
		mux.HandleFunc("POST /v1/auth/register", api.handleAuthRegister)
		mux.HandleFunc("POST /v1/auth/login", api.handleAuthLogin)
		mux.HandleFunc("POST /v1/auth/logout", api.requireAuth(api.handleAuthLogout))
		mux.HandleFunc("GET /v1/users/me", api.requireAuth(api.handleUsersMe))

		if api.friendsSvc != nil {
			mux.HandleFunc("GET /v1/friends", api.requireAuth(api.handleFriendsList))
			mux.HandleFunc("POST /v1/friends/requests", api.requireAuth(api.handleFriendsCreateRequest))
			mux.HandleFunc("POST /v1/friends/requests/{id}/accept", api.requireAuth(api.handleFriendsAccept))
			mux.HandleFunc("POST /v1/friends/requests/{id}/decline", api.requireAuth(api.handleFriendsDecline))
		}

		if api.matchSvc != nil {
			mux.HandleFunc("POST /v1/matches", api.requireAuth(api.handleMatchesCreate))
			mux.HandleFunc("GET /v1/matches", api.requireAuth(api.handleMatchesList))
			mux.HandleFunc("GET /v1/stats/summary", api.requireAuth(api.handleStatsSummary))
			mux.HandleFunc("GET /v1/stats/head-to-head/{id}", api.requireAuth(api.handleStatsHeadToHead))
		}
	}

	mux.Handle("/v1/", http.HandlerFunc(handleV1NotFound))

	var h http.Handler = mux
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
