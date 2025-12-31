package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"MtgLeaderwebserver/internal/domain"
)

type stubMatchesStore struct {
	t *testing.T

	created struct {
		called               bool
		createdBy            string
		startedAt            *time.Time
		endedAt              *time.Time
		playedAt             *time.Time
		winnerID             string
		participants         []domain.MatchParticipantInput
		format               domain.GameFormat
		totalDurationSeconds int
		turnCount            int
		clientRef            string
		updatedAt            time.Time
	}

	returnID    string
	createdFlag bool
	err         error

	matchForUser    domain.Match
	matchForUserErr error

	matchByClientRef    domain.Match
	matchByClientRefErr error
}

func (s *stubMatchesStore) CreateMatch(ctx context.Context, createdBy string, startedAt, endedAt, playedAt *time.Time, winnerID string, participants []domain.MatchParticipantInput, format domain.GameFormat, totalDurationSeconds, turnCount int, clientRef string, updatedAt time.Time) (string, bool, error) {
	s.created.called = true
	s.created.createdBy = createdBy
	s.created.startedAt = startedAt
	s.created.endedAt = endedAt
	s.created.playedAt = playedAt
	s.created.winnerID = winnerID
	s.created.participants = append([]domain.MatchParticipantInput(nil), participants...)
	s.created.format = format
	s.created.totalDurationSeconds = totalDurationSeconds
	s.created.turnCount = turnCount
	s.created.clientRef = clientRef
	s.created.updatedAt = updatedAt
	return s.returnID, s.createdFlag, s.err
}

func (s *stubMatchesStore) GetMatchByClientRef(ctx context.Context, createdBy, clientRef string) (domain.Match, error) {
	if s.matchByClientRefErr != nil {
		return domain.Match{}, s.matchByClientRefErr
	}
	if s.matchByClientRef.ID == "" {
		return domain.Match{}, domain.ErrNotFound
	}
	return s.matchByClientRef, nil
}

func (s *stubMatchesStore) ListMatchesForUser(ctx context.Context, userID string, limit int) ([]domain.Match, error) {
	return nil, nil
}

func (s *stubMatchesStore) GetMatchForUser(ctx context.Context, userID, matchID string) (domain.Match, error) {
	if s.matchForUserErr != nil {
		return domain.Match{}, s.matchForUserErr
	}
	return s.matchForUser, nil
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
	_, _, err := svc.CreateMatch(context.Background(), "u1", CreateMatchParams{
		UpdatedAt: time.Now(),
		Results:   []domain.MatchResultInput{{ID: "u1", Rank: 1}},
	})
	expectValidation(t, err)
	if store.created.called {
		t.Fatal("store should not be called on validation error")
	}
}

func TestCreateMatchRejectsMissingWinnerRank(t *testing.T) {
	store := &stubMatchesStore{}
	svc := &MatchService{Matches: store}
	_, _, err := svc.CreateMatch(context.Background(), "u1", CreateMatchParams{
		UpdatedAt: time.Now(),
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
	_, _, err := svc.CreateMatch(context.Background(), "u1", CreateMatchParams{
		UpdatedAt: time.Now(),
		Results: []domain.MatchResultInput{
			{ID: "u1", Rank: 1},
			{ID: "u2", Rank: 1},
		},
	})
	expectValidation(t, err)
}

func TestCreateMatchAllowsTies(t *testing.T) {
	updatedAt := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	store := &stubMatchesStore{
		returnID:    "match-1",
		createdFlag: true,
		matchForUser: domain.Match{
			ID:        "match-1",
			UpdatedAt: updatedAt,
		},
	}
	svc := &MatchService{Matches: store}
	match, result, err := svc.CreateMatch(context.Background(), "u1", CreateMatchParams{
		Format:    domain.FormatModern,
		UpdatedAt: updatedAt,
		Results: []domain.MatchResultInput{
			{ID: "u1", Rank: 1},
			{ID: "u2", Rank: 2},
			{ID: "u3", Rank: 2},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != MatchCreateApplied {
		t.Fatalf("expected match applied, got %v", result)
	}
	if match.ID != "match-1" {
		t.Fatalf("expected match id to be returned, got %q", match.ID)
	}
	if store.created.winnerID != "u1" {
		t.Fatalf("expected winner to be u1, got %q", store.created.winnerID)
	}
	if store.created.format != domain.FormatModern {
		t.Fatalf("expected format to be modern, got %q", store.created.format)
	}
	if len(store.created.participants) != 3 {
		t.Fatalf("expected 3 participants, got %d", len(store.created.participants))
	}
	places := make(map[string]int, len(store.created.participants))
	for _, p := range store.created.participants {
		places[p.UserID] = p.Place
	}
	if places["u1"] != 1 || places["u2"] != 2 || places["u3"] != 2 {
		t.Fatalf("unexpected participant places: %v", places)
	}
}

func TestCreateMatchLegacyPayload(t *testing.T) {
	updatedAt := time.Date(2025, 12, 29, 20, 0, 0, 0, time.UTC)
	store := &stubMatchesStore{
		returnID:    "match-2",
		createdFlag: true,
		matchForUser: domain.Match{
			ID:        "match-2",
			UpdatedAt: updatedAt,
		},
	}
	svc := &MatchService{Matches: store}
	playedAt := time.Date(2025, 12, 29, 20, 0, 0, 0, time.UTC)
	match, result, err := svc.CreateMatch(context.Background(), "u1", CreateMatchParams{
		PlayedAt:      &playedAt,
		WinnerID:      "u2",
		PlayerIDs:     []string{"u2", "u3"},
		ClientMatchID: "  device-1 ",
		UpdatedAt:     updatedAt,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != MatchCreateApplied {
		t.Fatalf("expected match applied, got %v", result)
	}
	if match.ID != "match-2" {
		t.Fatalf("expected match id to be returned, got %q", match.ID)
	}
	if store.created.format != domain.FormatCommander {
		t.Fatalf("expected default format commander, got %q", store.created.format)
	}
	if store.created.clientRef != "device-1" {
		t.Fatalf("expected trimmed client ref, got %q", store.created.clientRef)
	}
	if store.created.winnerID != "u2" {
		t.Fatalf("expected winner u2, got %q", store.created.winnerID)
	}
	if len(store.created.participants) != 3 {
		t.Fatalf("expected 3 participants, got %d", len(store.created.participants))
	}
	places := make(map[string]int, len(store.created.participants))
	for _, p := range store.created.participants {
		places[p.UserID] = p.Place
	}
	if places["u2"] != 1 || places["u1"] != 2 || places["u3"] != 2 {
		t.Fatalf("unexpected participant places: %v", places)
	}
}

func TestCreateMatchInvalidFormat(t *testing.T) {
	store := &stubMatchesStore{}
	svc := &MatchService{Matches: store}
	_, _, err := svc.CreateMatch(context.Background(), "u1", CreateMatchParams{
		Format:    domain.GameFormat("vintage"),
		UpdatedAt: time.Now(),
		Results: []domain.MatchResultInput{
			{ID: "u1", Rank: 1},
			{ID: "u2", Rank: 2},
		},
	})
	expectValidation(t, err)
}

func TestCreateMatchConflictOnClientMatchID(t *testing.T) {
	existing := domain.Match{ID: "match-9", UpdatedAt: time.Now()}
	store := &stubMatchesStore{
		matchByClientRef: existing,
	}
	svc := &MatchService{Matches: store}

	match, result, err := svc.CreateMatch(context.Background(), "u1", CreateMatchParams{
		ClientMatchID: "client-9",
		UpdatedAt:     time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != MatchCreateConflict {
		t.Fatalf("expected conflict result, got %v", result)
	}
	if match.ID != existing.ID {
		t.Fatalf("expected existing match to be returned, got %q", match.ID)
	}
	if store.created.called {
		t.Fatal("store should not be called on conflict")
	}
}

func TestCreateMatchWithGuestParticipant(t *testing.T) {
	store := &stubMatchesStore{
		returnID:    "match-10",
		createdFlag: true,
		matchForUser: domain.Match{
			ID: "match-10",
		},
	}
	svc := &MatchService{Matches: store}

	match, result, err := svc.CreateMatch(context.Background(), "u1", CreateMatchParams{
		ClientMatchID: "client-10",
		UpdatedAt:     time.Now(),
		Players: []domain.MatchParticipantInput{
			{SeatIndex: 0, UserID: "u1", DisplayName: "Player One", Place: 1},
			{SeatIndex: 1, GuestName: "Guest", DisplayName: "Guest", Place: 2},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != MatchCreateApplied {
		t.Fatalf("expected match applied, got %v", result)
	}
	if match.ID != "match-10" {
		t.Fatalf("unexpected match id: %s", match.ID)
	}
	if len(store.created.participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(store.created.participants))
	}
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
