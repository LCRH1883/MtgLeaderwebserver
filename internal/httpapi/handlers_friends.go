package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"MtgLeaderwebserver/internal/domain"
	"MtgLeaderwebserver/internal/service"
)

func (a *api) handleFriendsList(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}

	out, err := a.friendsSvc.ListOverview(r.Context(), u.ID)
	if err != nil {
		WriteDomainError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, out)
}

func (a *api) handleFriendsConnections(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}

	out, err := a.friendsSvc.ListConnections(r.Context(), u.ID)
	if err != nil {
		WriteDomainError(w, err)
		return
	}

	latest, err := a.friendsSvc.LatestFriendshipUpdate(r.Context(), u.ID)
	if err != nil {
		WriteDomainError(w, err)
		return
	}

	etag := friendsConnectionsETag(u.ID, out, latest)
	w.Header().Set("Cache-Control", "private, max-age=0")
	if match := strings.TrimSpace(r.Header.Get("If-None-Match")); match != "" && match == etag {
		w.Header().Set("ETag", etag)
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("ETag", etag)
	WriteJSON(w, http.StatusOK, out)
}

type createFriendRequestRequest struct {
	Username string `json:"username"`
}

func (a *api) handleFriendsCreateRequest(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}

	var req createFriendRequestRequest
	if err := decodeJSON(w, r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_json", "invalid json")
		return
	}

	fr, err := a.friendsSvc.CreateRequest(r.Context(), u.ID, req.Username)
	if err != nil {
		WriteDomainError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, fr)
}

func (a *api) handleFriendsAccept(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		WriteDomainError(w, domain.NewValidationError(map[string]string{"id": "required"}))
		return
	}

	var body friendRequestActionBody
	empty, err := decodeJSONAllowEmpty(w, r, &body)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad_json", "invalid json")
		return
	}

	var updatedAt *time.Time
	if !empty && body.UpdatedAt != nil {
		parsed, err := parseUpdatedAt(*body.UpdatedAt)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_updated_at", "updated_at must be RFC3339 UTC with milliseconds")
			return
		}
		updatedAt = &parsed
	}

	result, err := a.friendsSvc.Accept(r.Context(), u.ID, id, updatedAt)
	if err != nil {
		WriteDomainError(w, err)
		return
	}
	if result == service.FriendRequestActionConflict {
		connections, err := a.friendsSvc.ListConnections(r.Context(), u.ID)
		if err != nil {
			WriteDomainError(w, err)
			return
		}
		WriteJSON(w, http.StatusConflict, connections)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) handleFriendsDecline(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		WriteDomainError(w, domain.NewValidationError(map[string]string{"id": "required"}))
		return
	}

	var body friendRequestActionBody
	empty, err := decodeJSONAllowEmpty(w, r, &body)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad_json", "invalid json")
		return
	}

	var updatedAt *time.Time
	if !empty && body.UpdatedAt != nil {
		parsed, err := parseUpdatedAt(*body.UpdatedAt)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_updated_at", "updated_at must be RFC3339 UTC with milliseconds")
			return
		}
		updatedAt = &parsed
	}

	result, err := a.friendsSvc.Decline(r.Context(), u.ID, id, updatedAt)
	if err != nil {
		WriteDomainError(w, err)
		return
	}
	if result == service.FriendRequestActionConflict {
		connections, err := a.friendsSvc.ListConnections(r.Context(), u.ID)
		if err != nil {
			WriteDomainError(w, err)
			return
		}
		WriteJSON(w, http.StatusConflict, connections)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) handleFriendsCancel(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		WriteDomainError(w, domain.NewValidationError(map[string]string{"id": "required"}))
		return
	}

	var body friendRequestActionBody
	empty, err := decodeJSONAllowEmpty(w, r, &body)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad_json", "invalid json")
		return
	}

	var updatedAt *time.Time
	if !empty && body.UpdatedAt != nil {
		parsed, err := parseUpdatedAt(*body.UpdatedAt)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_updated_at", "updated_at must be RFC3339 UTC with milliseconds")
			return
		}
		updatedAt = &parsed
	}

	result, err := a.friendsSvc.Cancel(r.Context(), u.ID, id, updatedAt)
	if err != nil {
		WriteDomainError(w, err)
		return
	}
	if result == service.FriendRequestActionConflict {
		connections, err := a.friendsSvc.ListConnections(r.Context(), u.ID)
		if err != nil {
			WriteDomainError(w, err)
			return
		}
		WriteJSON(w, http.StatusConflict, connections)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) handleFriendsRemove(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		WriteDomainError(w, domain.NewValidationError(map[string]string{"id": "required"}))
		return
	}

	if err := a.friendsSvc.RemoveFriend(r.Context(), u.ID, id); err != nil {
		WriteDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type friendRequestActionBody struct {
	UpdatedAt *string `json:"updated_at,omitempty"`
}

func friendsConnectionsETag(userID string, connections []domain.FriendConnection, latestFriendshipUpdate time.Time) string {
	maxTime := latestFriendshipUpdate
	for _, conn := range connections {
		maxTime = maxTimeValue(maxTime, conn.CreatedAt)
		maxTime = maxTimeValue(maxTime, conn.UpdatedAt)
		if conn.User.AvatarUpdatedAt != nil {
			maxTime = maxTimeValue(maxTime, *conn.User.AvatarUpdatedAt)
		}
		if conn.User.UpdatedAt != nil {
			maxTime = maxTimeValue(maxTime, *conn.User.UpdatedAt)
		}
	}
	if maxTime.IsZero() {
		maxTime = time.Unix(0, 0).UTC()
	}
	return fmt.Sprintf("W/\"friends-connections-%s-%d\"", userID, maxTime.UnixNano())
}

func maxTimeValue(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}
