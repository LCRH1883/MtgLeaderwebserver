package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"MtgLeaderwebserver/internal/domain"
)

type stubMatchesStore struct {
	created struct {
		called               bool
		createdBy            string
		playedAt             *time.Time
		winnerID             string
		playerIDs            []string
		format               domain.GameFormat
		totalDurationSeconds int
		turnCount            int
		clientRef            string
		results              []domain.MatchResultInput
	}
	returnID string
	err      error
}

func (s *stubMatchesStore) CreateMatch(ctx context.Context, createdBy string, playedAt *time.Time, winnerID string, playerIDs []string, format domain.GameFormat, totalDurationSeconds, turnCount int, clientRef string, results []domain.MatchResultInput) (string, error) {
	s.created.called = true
	s.created.createdBy = createdBy
	s.created.playedAt = playedAt
	s.created.winnerID = winnerID
	s.created.playerIDs = append([]string(nil), playerIDs...)
	s.created.format = format
	s.created.totalDurationSeconds = totalDurationSeconds
	s.created.turnCount = turnCount
	s.created.clientRef = clientRef
	s.created.results = append([]domain.MatchResultInput(nil), results...)
	return s.returnID, s.err
}

func (s *stubMatchesStore) ListMatchesForUser(ctx context.Context, userID string, limit int) ([]domain.Match, error) {
	return nil, nil
}

func (s *stubMatchesStore) GetMatchForUser(ctx context.Context, userID, matchID string) (domain.Match, error) {
	return domain.Match{}, nil
}

func (s *stubMatchesStore) StatsSummary(ctx context.Context, userID string) (domain.StatsSummary, error) {
	return domain.StatsSummary{}, nil
}

func (s *stubMatchesStore) HeadToHead(ctx context.Context, userID, opponentID string) (domain.HeadToHeadStats, error) {
	return domain.HeadToHeadStats{}, nil
}

func TestCreateMatchRejectsSinglePlayer(t *testing.T) {
	store := &stubMatchesStore{}
	svc := &MatchService{Matches: store}
	_, err := svc.CreateMatch(context.Background(), "u1", CreateMatchParams{
		Results: []domain.MatchResultInput{{ID: "u1", Rank: 1}},
	})
	expectValidation(t, err)
	if store.created.called {
		t.Fatal("store should not be called on validation error")
	}
}

func TestCreateMatchRejectsMissingWinnerRank(t *testing.T) {
	store := &stubMatchesStore{}
	svc := &MatchService{Matches: store}
	_, err := svc.CreateMatch(context.Background(), "u1", CreateMatchParams{
		Results: []domain.MatchResultInput{
			{ID: "u1", Rank: 2},
			{ID: "u2", Rank: 3},
		},
	})
	expectValidation(t, err)
}

func TestCreateMatchRejectsMultipleWinners(t *testing.T) {
	store := &stubMatchesStore{}
	svc := &MatchService{Matches: store}
	_, err := svc.CreateMatch(context.Background(), "u1", CreateMatchParams{
		Results: []domain.MatchResultInput{
			{ID: "u1", Rank: 1},
			{ID: "u2", Rank: 1},
		},
	})
	expectValidation(t, err)
}

func TestCreateMatchAllowsTies(t *testing.T) {
	store := &stubMatchesStore{returnID: "match-1"}
	svc := &MatchService{Matches: store}
	id, err := svc.CreateMatch(context.Background(), "u1", CreateMatchParams{
		Format: domain.FormatModern,
		Results: []domain.MatchResultInput{
			{ID: "u1", Rank: 1},
			{ID: "u2", Rank: 2},
			{ID: "u3", Rank: 2},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "match-1" {
		t.Fatalf("expected match id to be returned, got %q", id)
	}
	if store.created.winnerID != "u1" {
		t.Fatalf("expected winner to be u1, got %q", store.created.winnerID)
	}
	if store.created.format != domain.FormatModern {
		t.Fatalf("expected format to be modern, got %q", store.created.format)
	}
}

func TestCreateMatchLegacyPayload(t *testing.T) {
	store := &stubMatchesStore{returnID: "match-2"}
	svc := &MatchService{Matches: store}
	playedAt := time.Date(2025, 12, 29, 20, 0, 0, 0, time.UTC)
	id, err := svc.CreateMatch(context.Background(), "u1", CreateMatchParams{
		PlayedAt:  &playedAt,
		WinnerID:  "u2",
		PlayerIDs: []string{"u2", "u3"},
		ClientRef: "  device-1 ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "match-2" {
		t.Fatalf("expected match id to be returned, got %q", id)
	}
	if store.created.format != domain.FormatCommander {
		t.Fatalf("expected default format commander, got %q", store.created.format)
	}
	if store.created.clientRef != "device-1" {
		t.Fatalf("expected trimmed client ref, got %q", store.created.clientRef)
	}
	if len(store.created.results) != 0 {
		t.Fatalf("expected no results for legacy payload, got %d", len(store.created.results))
	}
	if store.created.winnerID != "u2" {
		t.Fatalf("expected winner u2, got %q", store.created.winnerID)
	}
	wantPlayers := []string{"u2", "u3", "u1"}
	if !sameStringSet(store.created.playerIDs, wantPlayers) {
		t.Fatalf("expected players %v, got %v", wantPlayers, store.created.playerIDs)
	}
}

func TestCreateMatchInvalidFormat(t *testing.T) {
	store := &stubMatchesStore{}
	svc := &MatchService{Matches: store}
	_, err := svc.CreateMatch(context.Background(), "u1", CreateMatchParams{
		Format: domain.GameFormat("vintage"),
		Results: []domain.MatchResultInput{
			{ID: "u1", Rank: 1},
			{ID: "u2", Rank: 2},
		},
	})
	expectValidation(t, err)
}

func expectValidation(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	copyA := append([]string(nil), a...)
	copyB := append([]string(nil), b...)
	return reflect.DeepEqual(stringSet(copyA), stringSet(copyB))
}

func stringSet(in []string) map[string]bool {
	out := make(map[string]bool, len(in))
	for _, v := range in {
		out[v] = true
	}
	return out
}
