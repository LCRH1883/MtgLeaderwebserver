package httpapi

import (
	"net/http"
	"time"

	"MtgLeaderwebserver/internal/domain"
)

type userResponse struct {
	ID              string               `json:"id"`
	Email           string               `json:"email,omitempty"`
	Username        string               `json:"username"`
	DisplayName     string               `json:"display_name"`
	AvatarPath      string               `json:"avatar_path"`
	AvatarUpdatedAt *time.Time           `json:"avatar_updated_at,omitempty"`
	AvatarURL       string               `json:"avatar_url"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	StatsSummary    *domain.StatsSummary `json:"stats_summary,omitempty"`
}

func writeUser(w http.ResponseWriter, status int, u domain.User, stats *domain.StatsSummary) {
	resp := userResponse{
		ID:              u.ID,
		Email:           u.Email,
		Username:        u.Username,
		DisplayName:     u.DisplayName,
		AvatarPath:      u.AvatarPath,
		AvatarUpdatedAt: u.AvatarUpdatedAt,
		AvatarURL:       avatarURL(u),
		CreatedAt:       u.CreatedAt,
		UpdatedAt:       u.UpdatedAt,
		StatsSummary:    stats,
	}
	WriteJSON(w, status, resp)
}

func (a *api) handleUsersMe(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}

	var stats *domain.StatsSummary
	if include := r.URL.Query().Get("include_stats"); include == "1" || include == "true" {
		if a.matchSvc != nil {
			summary, err := a.matchSvc.Summary(r.Context(), u.ID)
			if err != nil {
				WriteDomainError(w, err)
				return
			}
			stats = &summary
		}
	}

	writeUser(w, http.StatusOK, u, stats)
}
