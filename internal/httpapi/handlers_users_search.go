package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"MtgLeaderwebserver/internal/domain"
)

func (a *api) handleUsersSearch(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			limit = n
		}
	}

	out, err := a.usersSvc.Search(r.Context(), q, limit, u.ID)
	if err != nil {
		WriteDomainError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, out)
}
