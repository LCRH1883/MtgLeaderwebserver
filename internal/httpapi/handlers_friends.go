package httpapi

import (
	"net/http"
	"strings"

	"MtgLeaderwebserver/internal/domain"
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

	if err := a.friendsSvc.Accept(r.Context(), u.ID, id); err != nil {
		WriteDomainError(w, err)
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

	if err := a.friendsSvc.Decline(r.Context(), u.ID, id); err != nil {
		WriteDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
