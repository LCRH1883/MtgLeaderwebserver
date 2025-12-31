package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"MtgLeaderwebserver/internal/domain"
)

type MatchesStore interface {
	CreateMatch(ctx context.Context, createdBy string, startedAt, endedAt, playedAt *time.Time, winnerID string, participants []domain.MatchParticipantInput, format domain.GameFormat, totalDurationSeconds, turnCount int, clientRef string, updatedAt time.Time) (string, bool, error)
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
	StartedAt            *time.Time
	EndedAt              *time.Time
	PlayedAt             *time.Time
	WinnerID             string
	PlayerIDs            []string
	Format               domain.GameFormat
	TotalDurationSeconds int
	TurnCount            int
	ClientMatchID        string
	UpdatedAt            time.Time
	Players              []domain.MatchParticipantInput
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

	participants, winnerID, err := s.buildParticipants(ctx, creatorID, p)
	if err != nil {
		return domain.Match{}, MatchCreateConflict, err
	}

	matchID, created, err := s.Matches.CreateMatch(ctx, creatorID, p.StartedAt, p.EndedAt, p.PlayedAt, winnerID, participants, format, p.TotalDurationSeconds, p.TurnCount, clientRef, p.UpdatedAt)
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
	raw := strings.ToLower(strings.TrimSpace(string(format)))
	if raw == "" {
		return domain.FormatCommander
	}
	if raw == "edh" {
		return domain.FormatCommander
	}
	return domain.GameFormat(raw)
}

func validFormat(format domain.GameFormat) bool {
	switch format {
	case domain.FormatCommander, domain.FormatBrawl, domain.FormatStandard, domain.FormatModern:
		return true
	default:
		return false
	}
}

func (s *MatchService) buildParticipants(ctx context.Context, creatorID string, p CreateMatchParams) ([]domain.MatchParticipantInput, string, error) {
	if len(p.Players) > 0 {
		return s.buildParticipantsFromPayload(ctx, creatorID, p.Players)
	}
	if len(p.Results) > 0 {
		return s.buildParticipantsFromResults(ctx, creatorID, p.Results)
	}
	return s.buildParticipantsFromLegacy(ctx, creatorID, p.PlayerIDs, p.WinnerID)
}

func (s *MatchService) buildParticipantsFromPayload(ctx context.Context, creatorID string, players []domain.MatchParticipantInput) ([]domain.MatchParticipantInput, string, error) {
	if len(players) < 2 {
		return nil, "", domain.NewValidationError(map[string]string{"players": "must have at least 2 players"})
	}

	seatSeen := make(map[int]bool, len(players))
	userSeen := make(map[string]bool, len(players))
	guestSeen := make(map[string]bool, len(players))
	winnerCount := 0
	winnerID := ""
	creatorFound := false

	out := make([]domain.MatchParticipantInput, 0, len(players))
	for _, p := range players {
		userID := strings.TrimSpace(p.UserID)
		guestName := strings.TrimSpace(p.GuestName)
		displayName := strings.TrimSpace(p.DisplayName)

		if (userID == "") == (guestName == "") {
			return nil, "", domain.NewValidationError(map[string]string{"players": "each player must include either user_id or guest_name"})
		}
		if p.Place < 1 {
			return nil, "", domain.NewValidationError(map[string]string{"players": "place must be >= 1"})
		}
		if p.SeatIndex < 0 {
			return nil, "", domain.NewValidationError(map[string]string{"players": "seat_index must be >= 0"})
		}
		if seatSeen[p.SeatIndex] {
			return nil, "", domain.NewValidationError(map[string]string{"players": "seat_index values must be unique"})
		}
		seatSeen[p.SeatIndex] = true

		if userID != "" {
			if userSeen[userID] {
				return nil, "", domain.NewValidationError(map[string]string{"players": "user_id values must be unique"})
			}
			userSeen[userID] = true
			if userID == creatorID {
				creatorFound = true
			}
		} else {
			if guestSeen[guestName] {
				return nil, "", domain.NewValidationError(map[string]string{"players": "guest_name values must be unique"})
			}
			guestSeen[guestName] = true
		}

		if displayName == "" && guestName != "" {
			displayName = guestName
		}

		if p.Place == 1 {
			winnerCount++
			if userID != "" {
				winnerID = userID
			}
		}

		if p.EliminatedTurn != nil && *p.EliminatedTurn < 0 {
			return nil, "", domain.NewValidationError(map[string]string{"players": "eliminated_turn_number must be >= 0"})
		}
		if p.EliminatedDuring != nil && *p.EliminatedDuring < 0 {
			return nil, "", domain.NewValidationError(map[string]string{"players": "eliminated_during_seat_index must be >= 0"})
		}
		if p.TotalTurnTimeMs != nil && *p.TotalTurnTimeMs < 0 {
			return nil, "", domain.NewValidationError(map[string]string{"players": "total_turn_time_ms must be >= 0"})
		}
		if p.TurnsTaken != nil && *p.TurnsTaken < 0 {
			return nil, "", domain.NewValidationError(map[string]string{"players": "turns_taken must be >= 0"})
		}

		out = append(out, domain.MatchParticipantInput{
			SeatIndex:        p.SeatIndex,
			UserID:           userID,
			GuestName:        guestName,
			DisplayName:      displayName,
			Place:            p.Place,
			EliminatedTurn:   p.EliminatedTurn,
			EliminatedDuring: p.EliminatedDuring,
			TotalTurnTimeMs:  p.TotalTurnTimeMs,
			TurnsTaken:       p.TurnsTaken,
		})
	}

	if !creatorFound {
		return nil, "", domain.NewValidationError(map[string]string{"players": "creator must be included"})
	}
	if winnerCount != 1 {
		return nil, "", domain.NewValidationError(map[string]string{"players": "exactly one player must have place 1"})
	}

	minSeat := out[0].SeatIndex
	maxSeat := out[0].SeatIndex
	for _, player := range out {
		if player.SeatIndex < minSeat {
			minSeat = player.SeatIndex
		}
		if player.SeatIndex > maxSeat {
			maxSeat = player.SeatIndex
		}
	}
	if minSeat != 0 || maxSeat != len(players)-1 {
		return nil, "", domain.NewValidationError(map[string]string{"players": "seat_index values must be contiguous from 0"})
	}

	if s.Friends != nil {
		for _, player := range out {
			if player.UserID == "" || player.UserID == creatorID {
				continue
			}
			ok, err := s.Friends.AreFriends(ctx, creatorID, player.UserID)
			if err != nil {
				return nil, "", err
			}
			if !ok {
				return nil, "", domain.ErrForbidden
			}
		}
	}

	return out, winnerID, nil
}

func (s *MatchService) buildParticipantsFromResults(ctx context.Context, creatorID string, results []domain.MatchResultInput) ([]domain.MatchParticipantInput, string, error) {
	if len(results) < 2 {
		return nil, "", domain.NewValidationError(map[string]string{"results": "must have at least 2 players"})
	}

	seen := make(map[string]bool, len(results))
	winnerCount := 0
	winnerID := ""
	participants := make([]domain.MatchParticipantInput, 0, len(results))

	for idx, res := range results {
		id := strings.TrimSpace(res.ID)
		if id == "" {
			return nil, "", domain.NewValidationError(map[string]string{"results": "each result must include a player id"})
		}
		if seen[id] {
			return nil, "", domain.NewValidationError(map[string]string{"results": "player ids must be unique"})
		}
		if res.Rank < 1 {
			return nil, "", domain.NewValidationError(map[string]string{"results": "rank must be >= 1"})
		}
		if res.EliminationTurn != nil && *res.EliminationTurn < 0 {
			return nil, "", domain.NewValidationError(map[string]string{"results": "elimination_turn must be >= 0"})
		}
		if res.EliminationBatch != nil && *res.EliminationBatch < 0 {
			return nil, "", domain.NewValidationError(map[string]string{"results": "elimination_batch must be >= 0"})
		}

		seen[id] = true
		if res.Rank == 1 {
			winnerCount++
			winnerID = id
		}
		seat := idx
		participants = append(participants, domain.MatchParticipantInput{
			SeatIndex:      seat,
			UserID:         id,
			DisplayName:    "",
			Place:          res.Rank,
			EliminatedTurn: res.EliminationTurn,
		})
	}
	if !seen[creatorID] {
		return nil, "", domain.NewValidationError(map[string]string{"results": "creator must be included in results"})
	}
	if winnerCount != 1 {
		return nil, "", domain.NewValidationError(map[string]string{"results": "exactly one player must have rank 1"})
	}

	if s.Friends != nil {
		for _, player := range participants {
			if player.UserID == creatorID {
				continue
			}
			ok, err := s.Friends.AreFriends(ctx, creatorID, player.UserID)
			if err != nil {
				return nil, "", err
			}
			if !ok {
				return nil, "", domain.ErrForbidden
			}
		}
	}

	return participants, winnerID, nil
}

func (s *MatchService) buildParticipantsFromLegacy(ctx context.Context, creatorID string, playerIDs []string, winnerRaw string) ([]domain.MatchParticipantInput, string, error) {
	seen := map[string]bool{creatorID: false}
	players := make([]string, 0, len(playerIDs)+1)
	for _, id := range playerIDs {
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
	winnerID := strings.TrimSpace(winnerRaw)
	if winnerID == "" {
		return nil, "", domain.NewValidationError(map[string]string{"winner_id": "winner_id is required"})
	}
	if !seen[winnerID] {
		return nil, "", domain.NewValidationError(map[string]string{"winner_id": "winner must be one of the players"})
	}
	if len(players) < 2 {
		return nil, "", domain.NewValidationError(map[string]string{"players": "must have at least 2 players"})
	}

	participants := make([]domain.MatchParticipantInput, 0, len(players))
	for idx, id := range players {
		place := 2
		if id == winnerID {
			place = 1
		}
		participants = append(participants, domain.MatchParticipantInput{
			SeatIndex: idx,
			UserID:    id,
			Place:     place,
		})
	}

	if s.Friends != nil {
		for _, pid := range players {
			if pid == creatorID {
				continue
			}
			ok, err := s.Friends.AreFriends(ctx, creatorID, pid)
			if err != nil {
				return nil, "", err
			}
			if !ok {
				return nil, "", domain.ErrForbidden
			}
		}
	}

	return participants, winnerID, nil
}
