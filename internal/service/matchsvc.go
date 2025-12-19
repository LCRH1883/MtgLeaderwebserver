package service

import (
	"context"
	"strings"
	"time"

	"MtgLeaderwebserver/internal/domain"
)

type MatchesStore interface {
	CreateMatch(ctx context.Context, createdBy string, playedAt *time.Time, winnerID string, playerIDs []string) (string, error)
	ListMatchesForUser(ctx context.Context, userID string, limit int) ([]domain.Match, error)
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
	PlayedAt  *time.Time
	WinnerID  string
	PlayerIDs []string
}

func (s *MatchService) CreateMatch(ctx context.Context, creatorID string, p CreateMatchParams) (string, error) {
	if s.Now == nil {
		s.Now = time.Now
	}

	seen := map[string]bool{creatorID: false}
	players := make([]string, 0, len(p.PlayerIDs)+1)
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
	if len(players) < 2 {
		return "", domain.NewValidationError(map[string]string{"players": "must have at least 2 players"})
	}
	if p.WinnerID != "" && !seen[p.WinnerID] {
		return "", domain.NewValidationError(map[string]string{"winner_id": "winner must be one of the players"})
	}

	// Enforce MVP invariant: all players are creator or accepted friends.
	if s.Friends != nil {
		for _, pid := range players {
			if pid == creatorID {
				continue
			}
			ok, err := s.Friends.AreFriends(ctx, creatorID, pid)
			if err != nil {
				return "", err
			}
			if !ok {
				return "", domain.ErrForbidden
			}
		}
	}

	return s.Matches.CreateMatch(ctx, creatorID, p.PlayedAt, p.WinnerID, players)
}

func (s *MatchService) ListMatches(ctx context.Context, userID string, limit int) ([]domain.Match, error) {
	return s.Matches.ListMatchesForUser(ctx, userID, limit)
}

func (s *MatchService) Summary(ctx context.Context, userID string) (domain.StatsSummary, error) {
	return s.Matches.StatsSummary(ctx, userID)
}

func (s *MatchService) HeadToHead(ctx context.Context, userID, opponentID string) (domain.HeadToHeadStats, error) {
	return s.Matches.HeadToHead(ctx, userID, opponentID)
}
