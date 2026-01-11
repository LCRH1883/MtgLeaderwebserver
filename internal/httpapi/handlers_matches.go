package httpapi

import (
	"encoding/json"
	"errors"
	"io"
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
	SeatIndex            int     `json:"seat_index"`
	Seat                 *int    `json:"seat,omitempty"` // client-only
	UserID               *string `json:"user_id,omitempty"`
	GuestName            *string `json:"guest_name,omitempty"`
	DisplayName          *string `json:"display_name,omitempty"`
	ProfileName          *string `json:"profile_name,omitempty"` // client-only
	Life                 *int    `json:"life,omitempty"`         // client-only
	Counters             any     `json:"counters,omitempty"`     // client-only
	Place                int     `json:"place"`
	EliminatedTurnNumber *int    `json:"eliminated_turn_number,omitempty"`
	EliminatedDuringSeat *int    `json:"eliminated_during_seat_index,omitempty"`
	TotalTurnTimeMs      *int64  `json:"total_turn_time_ms,omitempty"`
	TurnsTaken           *int    `json:"turns_taken,omitempty"`
}

func (a *api) handleMatchesCreate(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}

	var req createMatchRequest
	if err := decodeJSONAllowUnknownFields(w, r, &req); err != nil {
		if a.logger != nil {
			a.logger.Warn("matches: decode json failed", "err", err)
		}
		WriteError(w, http.StatusBadRequest, "bad_json", jsonDecodeErrorMessage(err))
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
		t, err := time.Parse(time.RFC3339Nano, req.PlayedAt)
		if err != nil {
			WriteDomainError(w, domain.NewValidationError(map[string]string{"played_at": "must be RFC3339 timestamp"}))
			return
		}
		playedAt = &t
	}

	var startedAt *time.Time
	if strings.TrimSpace(req.StartedAt) != "" {
		t, err := time.Parse(time.RFC3339Nano, req.StartedAt)
		if err != nil {
			WriteDomainError(w, domain.NewValidationError(map[string]string{"started_at": "must be RFC3339 timestamp"}))
			return
		}
		startedAt = &t
	}

	var endedAt *time.Time
	if strings.TrimSpace(req.EndedAt) != "" {
		t, err := time.Parse(time.RFC3339Nano, req.EndedAt)
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
		userID := normalizeOptionalString(derefString(p.UserID))
		guestName := normalizeOptionalString(derefString(p.GuestName))
		displayName := strings.TrimSpace(derefString(p.DisplayName))

		participants = append(participants, domain.MatchParticipantInput{
			SeatIndex:        p.SeatIndex,
			UserID:           userID,
			GuestName:        guestName,
			DisplayName:      displayName,
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
		var ve *domain.ValidationError
		if errors.As(err, &ve) {
			a.logger.Warn("matches: create validation failed", "fields", ve.Fields, "err", err)
		} else {
			a.logger.Error("matches: create failed", "err", err)
		}
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

func jsonDecodeErrorMessage(err error) string {
	if err == nil {
		return "invalid json"
	}
	if errors.Is(err, io.EOF) {
		return "empty request body"
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return "truncated json body"
	}

	// json.Decoder.DisallowUnknownFields() returns plain errors with this prefix.
	msg := err.Error()
	if strings.HasPrefix(msg, "json: unknown field ") {
		return msg
	}
	if strings.HasPrefix(msg, "multiple json values") {
		return msg
	}

	var se *json.SyntaxError
	if errors.As(err, &se) {
		return "invalid json syntax"
	}
	var ute *json.UnmarshalTypeError
	if errors.As(err, &ute) {
		if ute.Field != "" {
			return "invalid json type for field: " + ute.Field
		}
		return "invalid json type"
	}

	return "invalid json"
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func normalizeOptionalString(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	switch strings.ToLower(v) {
	case "null", "undefined":
		return ""
	default:
		return v
	}
}
