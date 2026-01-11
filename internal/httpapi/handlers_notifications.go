package httpapi

import (
	"net/http"
	"strings"

	"MtgLeaderwebserver/internal/domain"
)

type notificationTokenRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

type notificationTokenResponse struct {
	Token     string `json:"token"`
	Platform  string `json:"platform"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (a *api) handleNotificationsTokenUpsert(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}

	var req notificationTokenRequest
	if err := decodeJSON(w, r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_json", "invalid json")
		return
	}

	token := strings.TrimSpace(req.Token)
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	if token == "" || platform == "" {
		WriteDomainError(w, domain.NewValidationError(map[string]string{"token": "required", "platform": "required"}))
		return
	}
	switch platform {
	case "android", "ios":
	default:
		WriteDomainError(w, domain.NewValidationError(map[string]string{"platform": "must be ios or android"}))
		return
	}

	out, err := a.notificationsSvc.RegisterToken(r.Context(), u.ID, token, platform)
	if err != nil {
		WriteDomainError(w, err)
		return
	}

	resp := notificationTokenResponse{
		Token:     out.Token,
		Platform:  out.Platform,
		CreatedAt: formatMillis(out.CreatedAt),
		UpdatedAt: formatMillis(out.UpdatedAt),
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (a *api) handleNotificationsTokenDelete(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}

	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		WriteDomainError(w, domain.NewValidationError(map[string]string{"token": "required"}))
		return
	}

	if err := a.notificationsSvc.DeleteToken(r.Context(), u.ID, token); err != nil {
		WriteDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
