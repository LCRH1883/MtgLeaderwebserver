package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"MtgLeaderwebserver/internal/domain"
)

type MatchesStore interface {
	CreateMatch(ctx context.Context, createdBy string, playedAt *time.Time, winnerID string, playerIDs []string, format domain.GameFormat, totalDurationSeconds, turnCount int, clientRef string, updatedAt time.Time, results []domain.MatchResultInput) (string, bool, error)
	GetMatchByClientRef(ctx context.Context, createdBy, clientRef string) (domain.Match, error)
	ListMatchesForUser(ctx context.Context, userID string, limit int) ([]domain.Match, error)
	GetMatchForUser(ctx context.Context, userID, matchID string) (domain.Match, error)
	StatsSummary(ctx context.Context, userID string) (domain.StatsSummary, error)
	HeadToHead(ctx context.Context, userID, opponentID string) (domain.HeadToHeadStats, error)
}

type FriendshipChecker interface {
	AreFriends(ctx context.Context, userA, userB string) (bool, error)
}

type MatchService struct {
	Matches MatchesStore
	Friends FriendshipChecker
	Now     func() time.Time
}

type CreateMatchParams struct {
	PlayedAt             *time.Time
	WinnerID             string
	PlayerIDs            []string
	Format               domain.GameFormat
	TotalDurationSeconds int
	TurnCount            int
	ClientMatchID        string
	UpdatedAt            time.Time
	Results              []domain.MatchResultInput
}

type MatchCreateResult int

const (
	MatchCreateApplied MatchCreateResult = iota
	MatchCreateConflict
)

func (s *MatchService) CreateMatch(ctx context.Context, creatorID string, p CreateMatchParams) (domain.Match, MatchCreateResult, error) {
	if s.Now == nil {
		s.Now = time.Now
	}

	clientRef := strings.TrimSpace(p.ClientMatchID)
	if clientRef != "" {
		existing, err := s.Matches.GetMatchByClientRef(ctx, creatorID, clientRef)
		if err == nil {
			return existing, MatchCreateConflict, nil
		}
		if !errors.Is(err, domain.ErrNotFound) {
			return domain.Match{}, MatchCreateConflict, err
		}
	}

	format := normalizeFormat(p.Format)
	if !validFormat(format) {
		return domain.Match{}, MatchCreateConflict, domain.NewValidationError(map[string]string{"format": "must be commander, brawl, standard, or modern"})
	}
	if p.TotalDurationSeconds < 0 {
		return domain.Match{}, MatchCreateConflict, domain.NewValidationError(map[string]string{"total_duration_seconds": "must be >= 0"})
	}
	if p.TurnCount < 0 {
		return domain.Match{}, MatchCreateConflict, domain.NewValidationError(map[string]string{"turn_count": "must be >= 0"})
	}

	var (
		players  []string
		winnerID string
		results  []domain.MatchResultInput
	)

	if len(p.Results) > 0 {
		seen := make(map[string]bool, len(p.Results))
		winnerCount := 0
		results = make([]domain.MatchResultInput, 0, len(p.Results))
		for _, res := range p.Results {
			id := strings.TrimSpace(res.ID)
			if id == "" {
				return domain.Match{}, MatchCreateConflict, domain.NewValidationError(map[string]string{"results": "each result must include a player id"})
			}
			if seen[id] {
				return domain.Match{}, MatchCreateConflict, domain.NewValidationError(map[string]string{"results": "player ids must be unique"})
			}
			if res.Rank < 1 {
				return domain.Match{}, MatchCreateConflict, domain.NewValidationError(map[string]string{"results": "rank must be >= 1"})
			}
			if res.EliminationTurn != nil && *res.EliminationTurn < 0 {
				return domain.Match{}, MatchCreateConflict, domain.NewValidationError(map[string]string{"results": "elimination_turn must be >= 0"})
			}
			if res.EliminationBatch != nil && *res.EliminationBatch < 0 {
				return domain.Match{}, MatchCreateConflict, domain.NewValidationError(map[string]string{"results": "elimination_batch must be >= 0"})
			}
			seen[id] = true
			players = append(players, id)
			if res.Rank == 1 {
				winnerCount++
				winnerID = id
			}
			res.ID = id
			results = append(results, res)
		}
		if !seen[creatorID] {
			return domain.Match{}, MatchCreateConflict, domain.NewValidationError(map[string]string{"results": "creator must be included in results"})
		}
		if winnerCount != 1 {
			return domain.Match{}, MatchCreateConflict, domain.NewValidationError(map[string]string{"results": "exactly one player must have rank 1"})
		}
	} else {
		seen := map[string]bool{creatorID: false}
		players = make([]string, 0, len(p.PlayerIDs)+1)
		for _, id := range p.PlayerIDs {
			id = strings.TrimSpace(id)
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			players = append(players, id)
		}
		if !seen[creatorID] {
			players = append(players, creatorID)
			seen[creatorID] = true
		}
		winnerID = strings.TrimSpace(p.WinnerID)
		if winnerID != "" && !seen[winnerID] {
			return domain.Match{}, MatchCreateConflict, domain.NewValidationError(map[string]string{"winner_id": "winner must be one of the players"})
		}
	}

	if len(players) < 2 {
		return domain.Match{}, MatchCreateConflict, domain.NewValidationError(map[string]string{"players": "must have at least 2 players"})
	}

	// Enforce MVP invariant: all players are creator or accepted friends.
	if s.Friends != nil {
		for _, pid := range players {
			if pid == creatorID {
				continue
			}
			ok, err := s.Friends.AreFriends(ctx, creatorID, pid)
			if err != nil {
				return domain.Match{}, MatchCreateConflict, err
			}
			if !ok {
				return domain.Match{}, MatchCreateConflict, domain.ErrForbidden
			}
		}
	}

	matchID, created, err := s.Matches.CreateMatch(ctx, creatorID, p.PlayedAt, winnerID, players, format, p.TotalDurationSeconds, p.TurnCount, clientRef, p.UpdatedAt, results)
	if err != nil {
		return domain.Match{}, MatchCreateConflict, err
	}

	match, err := s.Matches.GetMatchForUser(ctx, creatorID, matchID)
	if err != nil {
		return domain.Match{}, MatchCreateConflict, err
	}
	if !created {
		return match, MatchCreateConflict, nil
	}
	return match, MatchCreateApplied, nil
}

func (s *MatchService) ListMatches(ctx context.Context, userID string, limit int) ([]domain.Match, error) {
	return s.Matches.ListMatchesForUser(ctx, userID, limit)
}

func (s *MatchService) GetMatch(ctx context.Context, userID, matchID string) (domain.Match, error) {
	return s.Matches.GetMatchForUser(ctx, userID, matchID)
}

func (s *MatchService) Summary(ctx context.Context, userID string) (domain.StatsSummary, error) {
	return s.Matches.StatsSummary(ctx, userID)
}

func (s *MatchService) HeadToHead(ctx context.Context, userID, opponentID string) (domain.HeadToHeadStats, error) {
	return s.Matches.HeadToHead(ctx, userID, opponentID)
}

func normalizeFormat(format domain.GameFormat) domain.GameFormat {
	if format == "" {
		return domain.FormatCommander
	}
	return format
}

func validFormat(format domain.GameFormat) bool {
	switch format {
	case domain.FormatCommander, domain.FormatBrawl, domain.FormatStandard, domain.FormatModern:
		return true
	default:
		return false
	}
}
