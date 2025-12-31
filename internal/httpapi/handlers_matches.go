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
	StartedAt            string                    `json:"started_at"`
	EndedAt              string                    `json:"ended_at"`
	StartingSeatIndex    *int                      `json:"starting_seat_index,omitempty"`
	Format               string                    `json:"format"`
	TotalDurationSeconds int                       `json:"total_duration_seconds"`
	TurnCount            int                       `json:"turn_count"`
	ClientRef            string                    `json:"client_ref"`
	ClientMatchID        string                    `json:"client_match_id"`
	UpdatedAt            string                    `json:"updated_at,omitempty"`
	Results              []domain.MatchResultInput `json:"results"`
	Players              []matchPlayerRequest      `json:"players"`
}

type createMatchResponse struct {
	MatchID      string               `json:"match_id"`
	Match        *domain.Match        `json:"match,omitempty"`
	StatsSummary *domain.StatsSummary `json:"stats_summary,omitempty"`
}

type matchPlayerRequest struct {
	SeatIndex            int    `json:"seat_index"`
	UserID               string `json:"user_id,omitempty"`
	GuestName            string `json:"guest_name,omitempty"`
	DisplayName          string `json:"display_name,omitempty"`
	Place                int    `json:"place"`
	EliminatedTurnNumber *int   `json:"eliminated_turn_number,omitempty"`
	EliminatedDuringSeat *int   `json:"eliminated_during_seat_index,omitempty"`
	TotalTurnTimeMs      *int64 `json:"total_turn_time_ms,omitempty"`
	TurnsTaken           *int   `json:"turns_taken,omitempty"`
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

	var updatedAt time.Time
	if strings.TrimSpace(req.UpdatedAt) != "" {
		parsed, err := parseUpdatedAt(req.UpdatedAt)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_updated_at", "updated_at must be RFC3339 UTC with milliseconds")
			return
		}
		updatedAt = parsed
	}

	clientMatchID := strings.TrimSpace(req.ClientMatchID)
	clientRef := strings.TrimSpace(req.ClientRef)
	if clientMatchID == "" {
		clientMatchID = clientRef
	} else if clientRef != "" && clientRef != clientMatchID {
		WriteDomainError(w, domain.NewValidationError(map[string]string{"client_match_id": "client_match_id must match client_ref"}))
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

	var startedAt *time.Time
	if strings.TrimSpace(req.StartedAt) != "" {
		t, err := time.Parse(time.RFC3339, req.StartedAt)
		if err != nil {
			WriteDomainError(w, domain.NewValidationError(map[string]string{"started_at": "must be RFC3339 timestamp"}))
			return
		}
		startedAt = &t
	}

	var endedAt *time.Time
	if strings.TrimSpace(req.EndedAt) != "" {
		t, err := time.Parse(time.RFC3339, req.EndedAt)
		if err != nil {
			WriteDomainError(w, domain.NewValidationError(map[string]string{"ended_at": "must be RFC3339 timestamp"}))
			return
		}
		endedAt = &t
	}

	if updatedAt.IsZero() {
		if endedAt != nil {
			updatedAt = endedAt.UTC().Truncate(time.Millisecond)
		} else {
			updatedAt = time.Now().UTC().Truncate(time.Millisecond)
		}
	}

	totalDurationSeconds := req.TotalDurationSeconds
	if totalDurationSeconds == 0 && startedAt != nil && endedAt != nil {
		diff := int(endedAt.Sub(*startedAt).Seconds())
		if diff > 0 {
			totalDurationSeconds = diff
		}
	}

	participants := make([]domain.MatchParticipantInput, 0, len(req.Players))
	for _, p := range req.Players {
		participants = append(participants, domain.MatchParticipantInput{
			SeatIndex:        p.SeatIndex,
			UserID:           strings.TrimSpace(p.UserID),
			GuestName:        strings.TrimSpace(p.GuestName),
			DisplayName:      strings.TrimSpace(p.DisplayName),
			Place:            p.Place,
			EliminatedTurn:   p.EliminatedTurnNumber,
			EliminatedDuring: p.EliminatedDuringSeat,
			TotalTurnTimeMs:  p.TotalTurnTimeMs,
			TurnsTaken:       p.TurnsTaken,
		})
	}

	match, result, err := a.matchSvc.CreateMatch(r.Context(), u.ID, service.CreateMatchParams{
		StartedAt:            startedAt,
		EndedAt:              endedAt,
		PlayedAt:             playedAt,
		WinnerID:             strings.TrimSpace(req.WinnerID),
		PlayerIDs:            req.PlayerIDs,
		Format:               domain.GameFormat(strings.TrimSpace(req.Format)),
		TotalDurationSeconds: totalDurationSeconds,
		TurnCount:            req.TurnCount,
		ClientMatchID:        clientMatchID,
		UpdatedAt:            updatedAt,
		Players:              participants,
		Results:              req.Results,
	})
	if err != nil {
		WriteDomainError(w, err)
		return
	}

	if result == service.MatchCreateConflict {
		WriteJSON(w, http.StatusOK, createMatchResponse{MatchID: match.ID, Match: &match})
		return
	}

	summary, err := a.matchSvc.Summary(r.Context(), u.ID)
	if err != nil {
		WriteDomainError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, createMatchResponse{
		MatchID:      match.ID,
		Match:        &match,
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
