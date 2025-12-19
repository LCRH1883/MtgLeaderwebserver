package httpapi

import (
	"context"
	"net"
	"net/http"
	"strings"

	"MtgLeaderwebserver/internal/auth"
	"MtgLeaderwebserver/internal/domain"
)

type authCtxKey int

const (
	authUserKey authCtxKey = iota
	authSessionKey
)

func (a *api) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(auth.SessionCookieName)
		if err != nil || c.Value == "" {
			WriteDomainError(w, domain.ErrUnauthorized)
			return
		}

		sessID, ok := a.cookieCodec.DecodeSessionID(c.Value)
		if !ok {
			WriteDomainError(w, domain.ErrUnauthorized)
			return
		}

		u, err := a.authSvc.GetUserForSession(r.Context(), sessID)
		if err != nil {
			WriteDomainError(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), authUserKey, u)
		ctx = context.WithValue(ctx, authSessionKey, sessID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func CurrentUser(ctx context.Context) (domain.User, bool) {
	u, ok := ctx.Value(authUserKey).(domain.User)
	return u, ok
}

func CurrentSessionID(ctx context.Context) (string, bool) {
	s, ok := ctx.Value(authSessionKey).(string)
	return s, ok
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
