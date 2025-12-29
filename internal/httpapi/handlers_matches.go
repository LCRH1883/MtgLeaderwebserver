package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"MtgLeaderwebserver/internal/domain"
	"MtgLeaderwebserver/internal/service"
)

type createMatchRequest struct {
	PlayerIDs            []string                 `json:"player_ids"`
	WinnerID             string                   `json:"winner_id"`
	PlayedAt             string                   `json:"played_at"`
	Format               string                   `json:"format"`
	TotalDurationSeconds int                      `json:"total_duration_seconds"`
	TurnCount            int                      `json:"turn_count"`
	ClientRef            string                   `json:"client_ref"`
	Results              []domain.MatchResultInput `json:"results"`
}

type createMatchResponse struct {
	ID string `json:"id"`
}

func (a *api) handleMatchesCreate(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}

	var req createMatchRequest
	if err := decodeJSON(w, r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_json", "invalid json")
		return
	}

	var playedAt *time.Time
	if strings.TrimSpace(req.PlayedAt) != "" {
		t, err := time.Parse(time.RFC3339, req.PlayedAt)
		if err != nil {
			WriteDomainError(w, domain.NewValidationError(map[string]string{"played_at": "must be RFC3339 timestamp"}))
			return
		}
		playedAt = &t
	}

	id, err := a.matchSvc.CreateMatch(r.Context(), u.ID, service.CreateMatchParams{
		PlayedAt:             playedAt,
		WinnerID:             strings.TrimSpace(req.WinnerID),
		PlayerIDs:            req.PlayerIDs,
		Format:               domain.GameFormat(strings.TrimSpace(req.Format)),
		TotalDurationSeconds: req.TotalDurationSeconds,
		TurnCount:            req.TurnCount,
		ClientRef:            strings.TrimSpace(req.ClientRef),
		Results:              req.Results,
	})
	if err != nil {
		WriteDomainError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, createMatchResponse{ID: id})
}

func (a *api) handleMatchesList(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}

	limit := 25
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			limit = n
		}
	}

	matches, err := a.matchSvc.ListMatches(r.Context(), u.ID, limit)
	if err != nil {
		WriteDomainError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, matches)
}

func (a *api) handleMatchesGet(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}

	matchID := strings.TrimSpace(r.PathValue("id"))
	if matchID == "" {
		WriteDomainError(w, domain.NewValidationError(map[string]string{"id": "required"}))
		return
	}

	match, err := a.matchSvc.GetMatch(r.Context(), u.ID, matchID)
	if err != nil {
		WriteDomainError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, match)
}
