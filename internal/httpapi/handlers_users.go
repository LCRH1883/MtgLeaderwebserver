package httpapi

import (
	"net/http"
	"time"

	"MtgLeaderwebserver/internal/domain"
)

type userResponse struct {
	ID       string    `json:"id"`
	Email    string    `json:"email,omitempty"`
	Username string    `json:"username"`
	Created  time.Time `json:"created_at"`
}

func writeUser(w http.ResponseWriter, status int, u domain.User) {
	resp := userResponse{
		ID:       u.ID,
		Email:    u.Email,
		Username: u.Username,
		Created:  u.CreatedAt,
	}
	WriteJSON(w, status, resp)
}

func (a *api) handleUsersMe(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}
	writeUser(w, http.StatusOK, u)
}
