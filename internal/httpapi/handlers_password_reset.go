package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"MtgLeaderwebserver/internal/domain"
)

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (a *api) handleAuthForgot(w http.ResponseWriter, r *http.Request) {
	if a.resetSvc == nil || a.emailSvc == nil {
		WriteError(w, http.StatusServiceUnavailable, "reset_unavailable", "password reset unavailable")
		return
	}

	var req forgotPasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_json", "invalid json")
		return
	}

	email := normalizeEmail(req.Email)
	if !validEmail(email) {
		WriteDomainError(w, domain.NewValidationError(map[string]string{"email": "must be a valid email"}))
		return
	}

	now := nowUTC()
	ip := clientIP(r)
	if !a.loginLimiter.Allow("forgot:ip:"+ip, now) || !a.loginLimiter.Allow("forgot:email:"+email, now) {
		WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many attempts")
		return
	}

	settings, ok, err := a.emailSvc.GetSMTPSettings(r.Context())
	if err != nil {
		a.logger.Error("smtp settings lookup failed", "err", err)
		WriteError(w, http.StatusServiceUnavailable, "smtp_unavailable", "smtp not configured")
		return
	}
	if !ok {
		WriteError(w, http.StatusServiceUnavailable, "smtp_unavailable", "smtp not configured")
		return
	}

	if a.authSvc == nil || a.authSvc.Users == nil {
		WriteError(w, http.StatusServiceUnavailable, "reset_unavailable", "password reset unavailable")
		return
	}

	user, err := a.authSvc.Users.GetUserByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		WriteDomainError(w, err)
		return
	}
	if user.Status == domain.UserStatusDisabled {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	token, err := a.resetSvc.CreateResetToken(r.Context(), user.ID, email, "")
	if err != nil {
		WriteDomainError(w, err)
		return
	}

	fromEmail := pickFromEmail(settings)
	if fromEmail == "" {
		WriteError(w, http.StatusServiceUnavailable, "smtp_unavailable", "smtp not configured")
		return
	}
	resetURL := a.resetLink(r, token)
	if err := a.emailSvc.SendPasswordReset(r.Context(), fromEmail, email, resetURL); err != nil {
		a.logger.Error("send reset email failed", "err", err)
		WriteError(w, http.StatusInternalServerError, "reset_failed", "failed to send reset email")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *api) handleAuthReset(w http.ResponseWriter, r *http.Request) {
	if a.resetSvc == nil {
		WriteError(w, http.StatusServiceUnavailable, "reset_unavailable", "password reset unavailable")
		return
	}

	var req resetPasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_json", "invalid json")
		return
	}

	token := strings.TrimSpace(req.Token)
	password := req.Password
	if token == "" || password == "" {
		WriteDomainError(w, domain.NewValidationError(map[string]string{"token": "required", "password": "required"}))
		return
	}
	if len(password) < 12 {
		WriteDomainError(w, domain.NewValidationError(map[string]string{"password": "must be at least 12 characters"}))
		return
	}

	if err := a.resetSvc.ResetPassword(r.Context(), token, password); err != nil {
		WriteDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *api) resetLink(r *http.Request, token string) string {
	if a.publicURL != nil {
		u := *a.publicURL
		u.Path = "/app/reset"
		u.RawQuery = "token=" + url.QueryEscape(token)
		return u.String()
	}
	scheme := "http"
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	}
	return fmt.Sprintf("%s://%s/app/reset?token=%s", scheme, r.Host, url.QueryEscape(token))
}

func pickFromEmail(settings domain.SMTPSettings) string {
	if strings.TrimSpace(settings.FromEmail) != "" {
		return strings.TrimSpace(settings.FromEmail)
	}
	for _, alias := range settings.AliasEmails {
		if strings.TrimSpace(alias) != "" {
			return strings.TrimSpace(alias)
		}
	}
	return ""
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
