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
	PlayerIDs            []string                  `json:"player_ids"`
	WinnerID             string                    `json:"winner_id"`
	PlayedAt             string                    `json:"played_at"`
	Format               string                    `json:"format"`
	TotalDurationSeconds int                       `json:"total_duration_seconds"`
	TurnCount            int                       `json:"turn_count"`
	ClientRef            string                    `json:"client_ref"`
	ClientMatchID        string                    `json:"client_match_id"`
	UpdatedAt            string                    `json:"updated_at"`
	Results              []domain.MatchResultInput `json:"results"`
	Match                *matchPayload             `json:"match,omitempty"`
}

type createMatchResponse struct {
	Match        domain.Match         `json:"match"`
	StatsSummary *domain.StatsSummary `json:"stats_summary,omitempty"`
}

type matchPayload struct {
	PlayerIDs            []string                  `json:"player_ids"`
	WinnerID             string                    `json:"winner_id"`
	PlayedAt             string                    `json:"played_at"`
	Format               string                    `json:"format"`
	TotalDurationSeconds int                       `json:"total_duration_seconds"`
	TurnCount            int                       `json:"turn_count"`
	Results              []domain.MatchResultInput `json:"results"`
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

	updatedAt, err := parseUpdatedAt(req.UpdatedAt)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_updated_at", "updated_at must be RFC3339 UTC with milliseconds")
		return
	}

	clientMatchID := strings.TrimSpace(req.ClientMatchID)
	clientRef := strings.TrimSpace(req.ClientRef)
	if clientMatchID == "" {
		clientMatchID = clientRef
	} else if clientRef != "" && clientRef != clientMatchID {
		WriteDomainError(w, domain.NewValidationError(map[string]string{"client_match_id": "client_match_id must match client_ref"}))
		return
	}

	payload := matchPayload{
		PlayerIDs:            req.PlayerIDs,
		WinnerID:             req.WinnerID,
		PlayedAt:             req.PlayedAt,
		Format:               req.Format,
		TotalDurationSeconds: req.TotalDurationSeconds,
		TurnCount:            req.TurnCount,
		Results:              req.Results,
	}
	if req.Match != nil {
		payload = *req.Match
	}

	var playedAt *time.Time
	if strings.TrimSpace(payload.PlayedAt) != "" {
		t, err := time.Parse(time.RFC3339, payload.PlayedAt)
		if err != nil {
			WriteDomainError(w, domain.NewValidationError(map[string]string{"played_at": "must be RFC3339 timestamp"}))
			return
		}
		playedAt = &t
	}

	match, result, err := a.matchSvc.CreateMatch(r.Context(), u.ID, service.CreateMatchParams{
		PlayedAt:             playedAt,
		WinnerID:             strings.TrimSpace(payload.WinnerID),
		PlayerIDs:            payload.PlayerIDs,
		Format:               domain.GameFormat(strings.TrimSpace(payload.Format)),
		TotalDurationSeconds: payload.TotalDurationSeconds,
		TurnCount:            payload.TurnCount,
		ClientMatchID:        clientMatchID,
		UpdatedAt:            updatedAt,
		Results:              payload.Results,
	})
	if err != nil {
		WriteDomainError(w, err)
		return
	}

	if result == service.MatchCreateConflict {
		WriteJSON(w, http.StatusConflict, createMatchResponse{Match: match})
		return
	}

	summary, err := a.matchSvc.Summary(r.Context(), u.ID)
	if err != nil {
		WriteDomainError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, createMatchResponse{
		Match:        match,
		StatsSummary: &summary,
	})
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
