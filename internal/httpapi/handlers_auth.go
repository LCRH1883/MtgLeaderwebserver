package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"MtgLeaderwebserver/internal/auth"
	"MtgLeaderwebserver/internal/domain"
)

type registerRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *api) handleAuthRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(w, r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_json", "invalid json")
		return
	}

	fields := map[string]string{}
	req.Username = normalizeUsername(req.Username)
	req.Email = normalizeEmail(req.Email)
	if !validEmail(req.Email) {
		fields["email"] = "must be a valid email"
	}
	if req.Username == "" || !validUsername(req.Username) {
		fields["username"] = "must be 3-24 chars [A-Za-z0-9_]"
	}
	if len(req.Password) < 12 {
		fields["password"] = "must be at least 12 characters"
	}
	if len(fields) > 0 {
		WriteDomainError(w, domain.NewValidationError(fields))
		return
	}

	userAgent := r.UserAgent()
	ip := clientIP(r)

	u, sessID, err := a.authSvc.Register(r.Context(), req.Email, req.Username, req.Password, ip, userAgent)
	if err != nil {
		WriteDomainError(w, err)
		return
	}

	cookieValue := a.cookieCodec.EncodeSessionID(sessID)
	auth.SetSessionCookie(w, cookieValue, a.sessionTTL, a.cookieSecure)

	writeUser(w, http.StatusCreated, u)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type idTokenRequest struct {
	IDToken string `json:"id_token"`
}

func (a *api) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_json", "invalid json")
		return
	}

	req.Email = normalizeEmail(req.Email)
	if !validEmail(req.Email) || req.Password == "" {
		WriteDomainError(w, domain.NewValidationError(map[string]string{"email": "required", "password": "required"}))
		return
	}

	now := time.Now()
	ip := clientIP(r)
	if !a.loginLimiter.Allow("ip:"+ip, now) || !a.loginLimiter.Allow("email:"+req.Email, now) {
		WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many attempts")
		return
	}

	userAgent := r.UserAgent()
	u, sessID, err := a.authSvc.Login(r.Context(), req.Email, req.Password, ip, userAgent)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			WriteDomainError(w, err)
			return
		}
		WriteDomainError(w, err)
		return
	}

	cookieValue := a.cookieCodec.EncodeSessionID(sessID)
	auth.SetSessionCookie(w, cookieValue, a.sessionTTL, a.cookieSecure)

	writeUser(w, http.StatusOK, u)
}

func (a *api) handleAuthLoginGoogle(w http.ResponseWriter, r *http.Request) {
	var req idTokenRequest
	if err := decodeJSON(w, r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_json", "invalid json")
		return
	}
	if strings.TrimSpace(req.IDToken) == "" {
		WriteDomainError(w, domain.NewValidationError(map[string]string{"id_token": "required"}))
		return
	}

	now := time.Now()
	ip := clientIP(r)
	if !a.loginLimiter.Allow("google:ip:"+ip, now) {
		WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many attempts")
		return
	}

	userAgent := r.UserAgent()
	u, sessID, err := a.authSvc.LoginWithGoogle(r.Context(), req.IDToken, ip, userAgent)
	if err != nil {
		WriteDomainError(w, err)
		return
	}

	cookieValue := a.cookieCodec.EncodeSessionID(sessID)
	auth.SetSessionCookie(w, cookieValue, a.sessionTTL, a.cookieSecure)

	writeUser(w, http.StatusOK, u)
}

func (a *api) handleAuthLoginApple(w http.ResponseWriter, r *http.Request) {
	var req idTokenRequest
	if err := decodeJSON(w, r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_json", "invalid json")
		return
	}
	if strings.TrimSpace(req.IDToken) == "" {
		WriteDomainError(w, domain.NewValidationError(map[string]string{"id_token": "required"}))
		return
	}

	now := time.Now()
	ip := clientIP(r)
	if !a.loginLimiter.Allow("apple:ip:"+ip, now) {
		WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many attempts")
		return
	}

	userAgent := r.UserAgent()
	u, sessID, err := a.authSvc.LoginWithApple(r.Context(), req.IDToken, ip, userAgent)
	if err != nil {
		WriteDomainError(w, err)
		return
	}

	cookieValue := a.cookieCodec.EncodeSessionID(sessID)
	auth.SetSessionCookie(w, cookieValue, a.sessionTTL, a.cookieSecure)

	writeUser(w, http.StatusOK, u)
}

func (a *api) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	sessID, ok := CurrentSessionID(r.Context())
	if !ok || sessID == "" {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}

	_ = a.authSvc.Logout(r.Context(), sessID)
	auth.ClearSessionCookie(w, a.cookieSecure)
	w.WriteHeader(http.StatusNoContent)
}
