package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"MtgLeaderwebserver/internal/domain"
	"MtgLeaderwebserver/internal/service"
)

type stubMatchesStore struct {
	t *testing.T

	getMatchByClientRefFunc func(context.Context, string, string) (domain.Match, error)
}

func (s *stubMatchesStore) CreateMatch(ctx context.Context, createdBy string, startedAt, endedAt, playedAt *time.Time, winnerID string, participants []domain.MatchParticipantInput, format domain.GameFormat, totalDurationSeconds, turnCount int, clientRef string, updatedAt time.Time) (string, bool, error) {
	s.t.Fatalf("CreateMatch called unexpectedly")
	return "", false, context.Canceled
}

func (s *stubMatchesStore) GetMatchByClientRef(ctx context.Context, createdBy, clientRef string) (domain.Match, error) {
	if s.getMatchByClientRefFunc != nil {
		return s.getMatchByClientRefFunc(ctx, createdBy, clientRef)
	}
	s.t.Fatalf("GetMatchByClientRef called unexpectedly")
	return domain.Match{}, context.Canceled
}

func (s *stubMatchesStore) ListMatchesForUser(ctx context.Context, userID string, limit int) ([]domain.Match, error) {
	return nil, nil
}

func (s *stubMatchesStore) GetMatchForUser(ctx context.Context, userID, matchID string) (domain.Match, error) {
	s.t.Fatalf("GetMatchForUser called unexpectedly")
	return domain.Match{}, context.Canceled
}

func (s *stubMatchesStore) StatsSummary(ctx context.Context, userID string) (domain.StatsSummary, error) {
	return domain.StatsSummary{}, nil
}

func (s *stubMatchesStore) HeadToHead(ctx context.Context, userID, opponentID string) (domain.HeadToHeadStats, error) {
	return domain.HeadToHeadStats{}, nil
}

func TestMatchesCreateInvalidUpdatedAt(t *testing.T) {
	store := &stubMatchesStore{t: t}
	api := &api{
		matchSvc: &service.MatchService{Matches: store},
	}

	body := `{"updated_at":"bad","player_ids":["u2"],"winner_id":"u1"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/matches", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), authUserKey, domain.User{ID: "u1"}))

	rr := httptest.NewRecorder()
	api.handleMatchesCreate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rr.Code)
	}

	var resp errorEnvelope
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error.Code != "invalid_updated_at" {
		t.Fatalf("unexpected error code: %s", resp.Error.Code)
	}
}

func TestMatchesCreateConflictReturnsMatch(t *testing.T) {
	updatedAt := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	existing := domain.Match{
		ID:            "match-1",
		CreatedBy:     "u1",
		UpdatedAt:     updatedAt,
		ClientMatchID: "client-1",
		Format:        domain.FormatCommander,
		Players: []domain.MatchPlayer{
			{User: domain.UserSummary{ID: "u1", Username: "one"}},
			{User: domain.UserSummary{ID: "u2", Username: "two"}},
		},
	}

	store := &stubMatchesStore{
		t: t,
		getMatchByClientRefFunc: func(_ context.Context, createdBy, clientRef string) (domain.Match, error) {
			if createdBy != "u1" || clientRef != "client-1" {
				t.Fatalf("unexpected client_ref lookup: %s %s", createdBy, clientRef)
			}
			return existing, nil
		},
	}
	api := &api{
		matchSvc: &service.MatchService{Matches: store},
	}

	body := `{"updated_at":"2025-01-01T12:00:00.000Z","client_match_id":"client-1"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/matches", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), authUserKey, domain.User{ID: "u1"}))

	rr := httptest.NewRecorder()
	api.handleMatchesCreate(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}

	var resp createMatchResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.MatchID != "match-1" {
		t.Fatalf("expected match id to be returned, got %q", resp.MatchID)
	}
	if resp.Match == nil || resp.Match.ID != "match-1" {
		t.Fatalf("expected match to be returned, got %#v", resp.Match)
	}
}
