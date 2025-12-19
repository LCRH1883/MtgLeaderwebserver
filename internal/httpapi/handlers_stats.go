package httpapi

import (
	"net/http"
	"strings"

	"MtgLeaderwebserver/internal/domain"
)

func (a *api) handleStatsSummary(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}

	summary, err := a.matchSvc.Summary(r.Context(), u.ID)
	if err != nil {
		WriteDomainError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, summary)
}

func (a *api) handleStatsHeadToHead(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}

	opponentID := strings.TrimSpace(r.PathValue("id"))
	if opponentID == "" {
		WriteDomainError(w, domain.NewValidationError(map[string]string{"id": "required"}))
		return
	}

	stats, err := a.matchSvc.HeadToHead(r.Context(), u.ID, opponentID)
	if err != nil {
		WriteDomainError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, stats)
}
