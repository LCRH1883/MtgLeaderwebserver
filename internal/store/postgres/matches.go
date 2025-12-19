package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"MtgLeaderwebserver/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MatchesStore struct {
	pool *pgxpool.Pool
}

func NewMatchesStore(pool *pgxpool.Pool) *MatchesStore {
	return &MatchesStore{pool: pool}
}

func (s *MatchesStore) CreateMatch(ctx context.Context, createdBy string, playedAt *time.Time, winnerID string, playerIDs []string) (string, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const insertMatch = `
		INSERT INTO matches (created_by, played_at, winner_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var matchIDUUID pgtype.UUID
	var playedAtAny any
	if playedAt != nil {
		playedAtAny = *playedAt
	}
	var winnerIDAny any
	if winnerID != "" {
		winnerIDAny = winnerID
	}
	if err := tx.QueryRow(ctx, insertMatch, createdBy, playedAtAny, winnerIDAny).Scan(&matchIDUUID); err != nil {
		return "", fmt.Errorf("insert match: %w", err)
	}
	matchID := uuidOrEmpty(matchIDUUID)

	const insertPlayer = `
		INSERT INTO match_players (match_id, user_id)
		VALUES ($1, $2)
	`
	for _, pid := range playerIDs {
		if _, err := tx.Exec(ctx, insertPlayer, matchID, pid); err != nil {
			var pgerr *pgconn.PgError
			if errors.As(err, &pgerr) && pgerr.Code == "23503" {
				return "", domain.ErrValidation
			}
			return "", fmt.Errorf("insert match player: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
	}
	return matchID, nil
}

func (s *MatchesStore) ListMatchesForUser(ctx context.Context, userID string, limit int) ([]domain.Match, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	// List matches where user participated.
	const q = `
		SELECT m.id, m.created_by, m.created_at, m.played_at, m.winner_id
		FROM matches m
		JOIN match_players mp ON mp.match_id = m.id
		WHERE mp.user_id = $1
		ORDER BY m.created_at DESC
		LIMIT $2
	`

	rows, err := s.pool.Query(ctx, q, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list matches: %w", err)
	}
	defer rows.Close()

	type matchRow struct {
		idUUID    pgtype.UUID
		createdBy pgtype.UUID
		createdAt time.Time
		playedAt  pgtype.Timestamptz
		winnerID  pgtype.UUID
	}

	var tmp []matchRow
	for rows.Next() {
		var r matchRow
		if err := rows.Scan(&r.idUUID, &r.createdBy, &r.createdAt, &r.playedAt, &r.winnerID); err != nil {
			return nil, fmt.Errorf("scan match: %w", err)
		}
		tmp = append(tmp, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list matches: %w", err)
	}

	out := make([]domain.Match, 0, len(tmp))
	for _, r := range tmp {
		matchID := uuidOrEmpty(r.idUUID)
		winnerID := uuidOrEmpty(r.winnerID)
		players, err := s.listPlayers(ctx, matchID, winnerID)
		if err != nil {
			return nil, err
		}

		out = append(out, domain.Match{
			ID:        matchID,
			CreatedBy: uuidOrEmpty(r.createdBy),
			CreatedAt: r.createdAt,
			PlayedAt:  timestamptzPtr(r.playedAt),
			WinnerID:  winnerID,
			Players:   players,
		})
	}

	return out, nil
}

func (s *MatchesStore) listPlayers(ctx context.Context, matchID string, winnerID string) ([]domain.MatchPlayer, error) {
	const q = `
		SELECT u.id, u.username
		FROM match_players mp
		JOIN users u ON u.id = mp.user_id
		WHERE mp.match_id = $1
		ORDER BY u.username ASC
	`

	rows, err := s.pool.Query(ctx, q, matchID)
	if err != nil {
		return nil, fmt.Errorf("list match players: %w", err)
	}
	defer rows.Close()

	var out []domain.MatchPlayer
	for rows.Next() {
		var idUUID pgtype.UUID
		var username string
		if err := rows.Scan(&idUUID, &username); err != nil {
			return nil, fmt.Errorf("scan match player: %w", err)
		}
		id := uuidOrEmpty(idUUID)
		out = append(out, domain.MatchPlayer{
			User:     domain.UserSummary{ID: id, Username: username},
			IsWinner: winnerID != "" && id == winnerID,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list match players: %w", err)
	}
	return out, nil
}

func (s *MatchesStore) StatsSummary(ctx context.Context, userID string) (domain.StatsSummary, error) {
	const q = `
		SELECT
			COUNT(*)::int AS matches_played,
			COALESCE(SUM(CASE WHEN m.winner_id = $1 THEN 1 ELSE 0 END), 0)::int AS wins,
			COALESCE(SUM(CASE WHEN m.winner_id IS NOT NULL AND m.winner_id <> $1 THEN 1 ELSE 0 END), 0)::int AS losses
		FROM matches m
		JOIN match_players mp ON mp.match_id = m.id
		WHERE mp.user_id = $1
	`
	var played, wins, losses int
	if err := s.pool.QueryRow(ctx, q, userID).Scan(&played, &wins, &losses); err != nil {
		return domain.StatsSummary{}, fmt.Errorf("stats summary: %w", err)
	}
	return domain.StatsSummary{MatchesPlayed: played, Wins: wins, Losses: losses}, nil
}

func (s *MatchesStore) HeadToHead(ctx context.Context, userID, opponentID string) (domain.HeadToHeadStats, error) {
	const qOpponent = `SELECT id, username FROM users WHERE id = $1`
	var oppIDUUID pgtype.UUID
	var oppUsername string
	if err := s.pool.QueryRow(ctx, qOpponent, opponentID).Scan(&oppIDUUID, &oppUsername); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.HeadToHeadStats{}, domain.ErrNotFound
		}
		return domain.HeadToHeadStats{}, fmt.Errorf("head-to-head opponent: %w", err)
	}

	const q = `
		SELECT
			COUNT(*)::int AS total,
			COALESCE(SUM(CASE WHEN m.winner_id = $1 THEN 1 ELSE 0 END), 0)::int AS wins
		FROM matches m
		WHERE
			EXISTS (SELECT 1 FROM match_players mp WHERE mp.match_id = m.id AND mp.user_id = $1)
			AND EXISTS (SELECT 1 FROM match_players mp WHERE mp.match_id = m.id AND mp.user_id = $2)
	`
	var total, wins int
	if err := s.pool.QueryRow(ctx, q, userID, opponentID).Scan(&total, &wins); err != nil {
		return domain.HeadToHeadStats{}, fmt.Errorf("head-to-head: %w", err)
	}

	return domain.HeadToHeadStats{
		Opponent: domain.UserSummary{ID: uuidOrEmpty(oppIDUUID), Username: oppUsername},
		Total:    total,
		Wins:     wins,
		Losses:   total - wins,
	}, nil
}
