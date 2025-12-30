package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"MtgLeaderwebserver/internal/domain"
)

type userResponse struct {
	ID              string               `json:"id"`
	Email           string               `json:"email,omitempty"`
	Username        string               `json:"username"`
	DisplayName     string               `json:"display_name"`
	Avatar          string               `json:"avatar"`
	AvatarPath      string               `json:"avatar_path"`
	AvatarUpdatedAt *string              `json:"avatar_updated_at,omitempty"`
	AvatarURL       string               `json:"avatar_url"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       string               `json:"updated_at"`
	StatsSummary    *domain.StatsSummary `json:"stats_summary,omitempty"`
}

func writeUser(w http.ResponseWriter, status int, u domain.User, stats *domain.StatsSummary) {
	w.Header().Set("ETag", userETag(u))
	resp := userResponse{
		ID:              u.ID,
		Email:           u.Email,
		Username:        u.Username,
		DisplayName:     u.DisplayName,
		Avatar:          u.AvatarPath,
		AvatarPath:      u.AvatarPath,
		AvatarUpdatedAt: formatMillisPtr(u.AvatarUpdatedAt),
		AvatarURL:       avatarURL(u),
		CreatedAt:       u.CreatedAt,
		UpdatedAt:       formatMillis(u.UpdatedAt),
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
		w.Header().Set("Cache-Control", "no-store")
		if a.matchSvc != nil {
			summary, err := a.matchSvc.Summary(r.Context(), u.ID)
			if err != nil {
				WriteDomainError(w, err)
				return
			}
			stats = &summary
		}
	} else {
		w.Header().Set("Cache-Control", "private, max-age=0")
		etag := userETag(u)
		if match := strings.TrimSpace(r.Header.Get("If-None-Match")); match != "" && match == etag {
			w.Header().Set("ETag", etag)
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	writeUser(w, http.StatusOK, u, stats)
}

func userETag(u domain.User) string {
	return fmt.Sprintf("W/\"user:%s:%d\"", u.ID, u.UpdatedAt.UnixNano())
}

func formatMillis(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

func formatMillisPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	out := formatMillis(*t)
	return &out
}
