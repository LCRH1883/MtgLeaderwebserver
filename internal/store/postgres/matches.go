package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

func normalizeFormat(format pgtype.Text) domain.GameFormat {
	f := textOrEmpty(format)
	if f == "" {
		return domain.FormatCommander
	}
	return domain.GameFormat(f)
}

func (s *MatchesStore) CreateMatch(ctx context.Context, createdBy string, playedAt *time.Time, winnerID string, playerIDs []string, format domain.GameFormat, totalDurationSeconds, turnCount int, clientRef string, updatedAt time.Time, results []domain.MatchResultInput) (string, bool, error) {
	if clientRef != "" {
		var existingID pgtype.UUID
		err := s.pool.QueryRow(ctx, `SELECT id FROM matches WHERE created_by = $1 AND client_ref = $2`, createdBy, clientRef).Scan(&existingID)
		if err == nil {
			return uuidOrEmpty(existingID), false, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", false, fmt.Errorf("lookup client_ref: %w", err)
		}
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", false, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const insertMatch = `
		INSERT INTO matches (created_by, played_at, winner_id, format, total_duration_seconds, turn_count, client_ref, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
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
	if err := tx.QueryRow(ctx, insertMatch, createdBy, playedAtAny, winnerIDAny, format, totalDurationSeconds, turnCount, nullIfEmpty(clientRef), updatedAt).Scan(&matchIDUUID); err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) && pgerr.ConstraintName == "matches_client_ref_uq" {
			var existingID pgtype.UUID
			if lookupErr := s.pool.QueryRow(ctx, `SELECT id FROM matches WHERE created_by = $1 AND client_ref = $2`, createdBy, clientRef).Scan(&existingID); lookupErr == nil {
				return uuidOrEmpty(existingID), false, nil
			}
			return "", false, domain.NewValidationError(map[string]string{"client_ref": "already used"})
		}
		return "", false, fmt.Errorf("insert match: %w", err)
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
				return "", false, domain.ErrValidation
			}
			return "", false, fmt.Errorf("insert match player: %w", err)
		}
	}

	if len(results) > 0 {
		const insertResult = `
			INSERT INTO match_player_results (match_id, user_id, rank, elimination_turn, elimination_batch)
			VALUES ($1, $2, $3, $4, $5)
		`
		for _, res := range results {
			var elimTurn any
			if res.EliminationTurn != nil {
				elimTurn = *res.EliminationTurn
			}
			var elimBatch any
			if res.EliminationBatch != nil {
				elimBatch = *res.EliminationBatch
			}
			if _, err := tx.Exec(ctx, insertResult, matchID, res.ID, res.Rank, elimTurn, elimBatch); err != nil {
				var pgerr *pgconn.PgError
				if errors.As(err, &pgerr) && pgerr.Code == "23503" {
					return "", false, domain.ErrValidation
				}
				return "", false, fmt.Errorf("insert match result: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", false, fmt.Errorf("commit tx: %w", err)
	}
	return matchID, true, nil
}

func (s *MatchesStore) ListMatchesForUser(ctx context.Context, userID string, limit int) ([]domain.Match, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	// List matches where user participated.
	const q = `
		SELECT m.id, m.created_by, m.created_at, m.updated_at, m.played_at, m.winner_id, m.format, m.total_duration_seconds, m.turn_count, m.client_ref
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
		idUUID       pgtype.UUID
		createdBy    pgtype.UUID
		createdAt    time.Time
		updatedAt    time.Time
		playedAt     pgtype.Timestamptz
		winnerID     pgtype.UUID
		format       pgtype.Text
		durationSecs int
		turnCount    int
		clientRef    pgtype.Text
	}

	var tmp []matchRow
	for rows.Next() {
		var r matchRow
		if err := rows.Scan(&r.idUUID, &r.createdBy, &r.createdAt, &r.updatedAt, &r.playedAt, &r.winnerID, &r.format, &r.durationSecs, &r.turnCount, &r.clientRef); err != nil {
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
			ID:                   matchID,
			CreatedBy:            uuidOrEmpty(r.createdBy),
			CreatedAt:            r.createdAt,
			UpdatedAt:            r.updatedAt,
			ClientMatchID:        textOrEmpty(r.clientRef),
			PlayedAt:             timestamptzPtr(r.playedAt),
			WinnerID:             winnerID,
			Format:               normalizeFormat(r.format),
			TotalDurationSeconds: r.durationSecs,
			TurnCount:            r.turnCount,
			Players:              players,
		})
	}

	return out, nil
}

func (s *MatchesStore) GetMatchForUser(ctx context.Context, userID, matchID string) (domain.Match, error) {
	const q = `
		SELECT m.id, m.created_by, m.created_at, m.updated_at, m.played_at, m.winner_id, m.format, m.total_duration_seconds, m.turn_count, m.client_ref
		FROM matches m
		JOIN match_players mp ON mp.match_id = m.id
		WHERE m.id = $1 AND mp.user_id = $2
		LIMIT 1
	`
	var (
		idUUID       pgtype.UUID
		createdBy    pgtype.UUID
		createdAt    time.Time
		updatedAt    time.Time
		playedAt     pgtype.Timestamptz
		winnerID     pgtype.UUID
		format       pgtype.Text
		durationSecs int
		turnCount    int
		clientRef    pgtype.Text
	)
	if err := s.pool.QueryRow(ctx, q, matchID, userID).Scan(
		&idUUID,
		&createdBy,
		&createdAt,
		&updatedAt,
		&playedAt,
		&winnerID,
		&format,
		&durationSecs,
		&turnCount,
		&clientRef,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Match{}, domain.ErrNotFound
		}
		return domain.Match{}, fmt.Errorf("get match: %w", err)
	}

	matchID = uuidOrEmpty(idUUID)
	winner := uuidOrEmpty(winnerID)
	players, err := s.listPlayers(ctx, matchID, winner)
	if err != nil {
		return domain.Match{}, err
	}

	return domain.Match{
		ID:                   matchID,
		CreatedBy:            uuidOrEmpty(createdBy),
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
		ClientMatchID:        textOrEmpty(clientRef),
		PlayedAt:             timestamptzPtr(playedAt),
		WinnerID:             winner,
		Format:               normalizeFormat(format),
		TotalDurationSeconds: durationSecs,
		TurnCount:            turnCount,
		Players:              players,
	}, nil
}

func (s *MatchesStore) GetMatchByClientRef(ctx context.Context, createdBy, clientRef string) (domain.Match, error) {
	if strings.TrimSpace(clientRef) == "" {
		return domain.Match{}, domain.ErrNotFound
	}

	const q = `
		SELECT m.id, m.created_by, m.created_at, m.updated_at, m.played_at, m.winner_id, m.format, m.total_duration_seconds, m.turn_count, m.client_ref
		FROM matches m
		WHERE m.created_by = $1 AND m.client_ref = $2
		LIMIT 1
	`
	var (
		idUUID        pgtype.UUID
		createdByUUID pgtype.UUID
		createdAt     time.Time
		updatedAt     time.Time
		playedAt      pgtype.Timestamptz
		winnerID      pgtype.UUID
		format        pgtype.Text
		durationSecs  int
		turnCount     int
		clientRefText pgtype.Text
	)
	if err := s.pool.QueryRow(ctx, q, createdBy, clientRef).Scan(
		&idUUID,
		&createdByUUID,
		&createdAt,
		&updatedAt,
		&playedAt,
		&winnerID,
		&format,
		&durationSecs,
		&turnCount,
		&clientRefText,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Match{}, domain.ErrNotFound
		}
		return domain.Match{}, fmt.Errorf("get match by client_ref: %w", err)
	}

	matchID := uuidOrEmpty(idUUID)
	winner := uuidOrEmpty(winnerID)
	players, err := s.listPlayers(ctx, matchID, winner)
	if err != nil {
		return domain.Match{}, err
	}

	return domain.Match{
		ID:                   matchID,
		CreatedBy:            uuidOrEmpty(createdByUUID),
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
		ClientMatchID:        textOrEmpty(clientRefText),
		PlayedAt:             timestamptzPtr(playedAt),
		WinnerID:             winner,
		Format:               normalizeFormat(format),
		TotalDurationSeconds: durationSecs,
		TurnCount:            turnCount,
		Players:              players,
	}, nil
}

func (s *MatchesStore) listPlayers(ctx context.Context, matchID string, winnerID string) ([]domain.MatchPlayer, error) {
	const q = `
		SELECT u.id, u.username, r.rank, r.elimination_turn, r.elimination_batch
		FROM match_players mp
		JOIN users u ON u.id = mp.user_id
		LEFT JOIN match_player_results r ON r.match_id = mp.match_id AND r.user_id = mp.user_id
		WHERE mp.match_id = $1
		ORDER BY COALESCE(r.rank, 2147483647), u.username ASC
	`

	rows, err := s.pool.Query(ctx, q, matchID)
	if err != nil {
		return nil, fmt.Errorf("list match players: %w", err)
	}
	defer rows.Close()

	var out []domain.MatchPlayer
	for rows.Next() {
		var (
			idUUID           pgtype.UUID
			username         string
			rank             pgtype.Int4
			eliminationTurn  pgtype.Int4
			eliminationBatch pgtype.Int4
		)
		if err := rows.Scan(&idUUID, &username, &rank, &eliminationTurn, &eliminationBatch); err != nil {
			return nil, fmt.Errorf("scan match player: %w", err)
		}
		id := uuidOrEmpty(idUUID)
		out = append(out, domain.MatchPlayer{
			User:             domain.UserSummary{ID: id, Username: username},
			IsWinner:         winnerID != "" && id == winnerID,
			Rank:             int4Ptr(rank),
			EliminationTurn:  int4Ptr(eliminationTurn),
			EliminationBatch: int4Ptr(eliminationBatch),
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
			COUNT(*) FILTER (WHERE m.winner_id IS NOT NULL)::int AS matches_played,
			COALESCE(SUM(CASE WHEN m.winner_id = $1 THEN 1 ELSE 0 END), 0)::int AS wins,
			COALESCE(SUM(CASE WHEN m.winner_id IS NOT NULL AND m.winner_id <> $1 THEN 1 ELSE 0 END), 0)::int AS losses,
			COALESCE(SUM(CASE WHEN m.winner_id IS NOT NULL AND m.turn_count > 0 THEN m.total_duration_seconds ELSE 0 END), 0)::int AS total_seconds,
			COALESCE(SUM(CASE WHEN m.winner_id IS NOT NULL AND m.turn_count > 0 THEN m.turn_count ELSE 0 END), 0)::int AS total_turns
		FROM matches m
		JOIN match_players mp ON mp.match_id = m.id
		WHERE mp.user_id = $1
	`
	var played, wins, losses, totalSeconds, totalTurns int
	if err := s.pool.QueryRow(ctx, q, userID).Scan(&played, &wins, &losses, &totalSeconds, &totalTurns); err != nil {
		return domain.StatsSummary{}, fmt.Errorf("stats summary: %w", err)
	}
	avgTurn := 0
	if totalTurns > 0 {
		avgTurn = totalSeconds / totalTurns
	}

	byFormat, err := s.statsSummaryByFormat(ctx, userID)
	if err != nil {
		return domain.StatsSummary{}, err
	}
	mostBeat, err := s.mostOftenBeat(ctx, userID)
	if err != nil {
		return domain.StatsSummary{}, err
	}
	mostBeats, err := s.mostOftenBeatsYou(ctx, userID)
	if err != nil {
		return domain.StatsSummary{}, err
	}

	return domain.StatsSummary{
		MatchesPlayed:     played,
		Wins:              wins,
		Losses:            losses,
		AvgTurnSeconds:    avgTurn,
		ByFormat:          byFormat,
		MostOftenBeat:     mostBeat,
		MostOftenBeatsYou: mostBeats,
	}, nil
}

func (s *MatchesStore) statsSummaryByFormat(ctx context.Context, userID string) (map[string]domain.StatsSummary, error) {
	const q = `
		SELECT
			m.format,
			COUNT(*) FILTER (WHERE m.winner_id IS NOT NULL)::int AS matches_played,
			COALESCE(SUM(CASE WHEN m.winner_id = $1 THEN 1 ELSE 0 END), 0)::int AS wins,
			COALESCE(SUM(CASE WHEN m.winner_id IS NOT NULL AND m.winner_id <> $1 THEN 1 ELSE 0 END), 0)::int AS losses,
			COALESCE(SUM(CASE WHEN m.winner_id IS NOT NULL AND m.turn_count > 0 THEN m.total_duration_seconds ELSE 0 END), 0)::int AS total_seconds,
			COALESCE(SUM(CASE WHEN m.winner_id IS NOT NULL AND m.turn_count > 0 THEN m.turn_count ELSE 0 END), 0)::int AS total_turns
		FROM matches m
		JOIN match_players mp ON mp.match_id = m.id
		WHERE mp.user_id = $1
		GROUP BY m.format
	`

	rows, err := s.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("stats by format: %w", err)
	}
	defer rows.Close()

	out := make(map[string]domain.StatsSummary)
	for rows.Next() {
		var (
			formatText   pgtype.Text
			played       int
			wins         int
			losses       int
			totalSeconds int
			totalTurns   int
		)
		if err := rows.Scan(&formatText, &played, &wins, &losses, &totalSeconds, &totalTurns); err != nil {
			return nil, fmt.Errorf("scan stats by format: %w", err)
		}
		avgTurn := 0
		if totalTurns > 0 {
			avgTurn = totalSeconds / totalTurns
		}
		format := string(normalizeFormat(formatText))
		out[format] = domain.StatsSummary{
			MatchesPlayed:  played,
			Wins:           wins,
			Losses:         losses,
			AvgTurnSeconds: avgTurn,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("stats by format: %w", err)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func (s *MatchesStore) mostOftenBeat(ctx context.Context, userID string) (*domain.OpponentStat, error) {
	const q = `
		SELECT u.id, u.username, COUNT(*)::int AS wins
		FROM matches m
		JOIN match_players mp ON mp.match_id = m.id
		JOIN users u ON u.id = mp.user_id
		WHERE m.winner_id = $1 AND mp.user_id <> $1
		GROUP BY u.id, u.username
		ORDER BY wins DESC, u.username ASC
		LIMIT 1
	`

	var idUUID pgtype.UUID
	var username string
	var count int
	if err := s.pool.QueryRow(ctx, q, userID).Scan(&idUUID, &username, &count); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("most often beat: %w", err)
	}
	return &domain.OpponentStat{
		Opponent: domain.UserSummary{ID: uuidOrEmpty(idUUID), Username: username},
		Count:    count,
	}, nil
}

func (s *MatchesStore) mostOftenBeatsYou(ctx context.Context, userID string) (*domain.OpponentStat, error) {
	const q = `
		SELECT u.id, u.username, COUNT(*)::int AS wins
		FROM matches m
		JOIN match_players mp ON mp.match_id = m.id AND mp.user_id = $1
		JOIN users u ON u.id = m.winner_id
		WHERE m.winner_id IS NOT NULL AND m.winner_id <> $1
		GROUP BY u.id, u.username
		ORDER BY wins DESC, u.username ASC
		LIMIT 1
	`

	var idUUID pgtype.UUID
	var username string
	var count int
	if err := s.pool.QueryRow(ctx, q, userID).Scan(&idUUID, &username, &count); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("most often beats you: %w", err)
	}
	return &domain.OpponentStat{
		Opponent: domain.UserSummary{ID: uuidOrEmpty(idUUID), Username: username},
		Count:    count,
	}, nil
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
			COALESCE(SUM(CASE WHEN m.winner_id = $1 THEN 1 ELSE 0 END), 0)::int AS wins,
			COALESCE(SUM(CASE WHEN m.winner_id = $2 THEN 1 ELSE 0 END), 0)::int AS losses,
			COALESCE(SUM(CASE WHEN m.winner_id IS NOT NULL AND m.winner_id <> $1 AND m.winner_id <> $2 THEN 1 ELSE 0 END), 0)::int AS co_losses
		FROM matches m
		WHERE
			m.winner_id IS NOT NULL
			EXISTS (SELECT 1 FROM match_players mp WHERE mp.match_id = m.id AND mp.user_id = $1)
			AND EXISTS (SELECT 1 FROM match_players mp WHERE mp.match_id = m.id AND mp.user_id = $2)
	`
	var total, wins, losses, coLosses int
	if err := s.pool.QueryRow(ctx, q, userID, opponentID).Scan(&total, &wins, &losses, &coLosses); err != nil {
		return domain.HeadToHeadStats{}, fmt.Errorf("head-to-head: %w", err)
	}

	byFormat, err := s.headToHeadByFormat(ctx, userID, opponentID)
	if err != nil {
		return domain.HeadToHeadStats{}, err
	}

	return domain.HeadToHeadStats{
		Opponent: domain.UserSummary{ID: uuidOrEmpty(oppIDUUID), Username: oppUsername},
		Total:    total,
		Wins:     wins,
		Losses:   losses,
		CoLosses: coLosses,
		ByFormat: byFormat,
	}, nil
}

func (s *MatchesStore) headToHeadByFormat(ctx context.Context, userID, opponentID string) (map[string]domain.HeadToHeadStats, error) {
	const q = `
		SELECT
			m.format,
			COUNT(*)::int AS total,
			COALESCE(SUM(CASE WHEN m.winner_id = $1 THEN 1 ELSE 0 END), 0)::int AS wins,
			COALESCE(SUM(CASE WHEN m.winner_id = $2 THEN 1 ELSE 0 END), 0)::int AS losses,
			COALESCE(SUM(CASE WHEN m.winner_id IS NOT NULL AND m.winner_id <> $1 AND m.winner_id <> $2 THEN 1 ELSE 0 END), 0)::int AS co_losses
		FROM matches m
		WHERE
			m.winner_id IS NOT NULL
			AND EXISTS (SELECT 1 FROM match_players mp WHERE mp.match_id = m.id AND mp.user_id = $1)
			AND EXISTS (SELECT 1 FROM match_players mp WHERE mp.match_id = m.id AND mp.user_id = $2)
		GROUP BY m.format
	`
	rows, err := s.pool.Query(ctx, q, userID, opponentID)
	if err != nil {
		return nil, fmt.Errorf("head-to-head by format: %w", err)
	}
	defer rows.Close()

	out := make(map[string]domain.HeadToHeadStats)
	for rows.Next() {
		var (
			formatText pgtype.Text
			total      int
			wins       int
			losses     int
			coLosses   int
		)
		if err := rows.Scan(&formatText, &total, &wins, &losses, &coLosses); err != nil {
			return nil, fmt.Errorf("scan head-to-head by format: %w", err)
		}
		format := string(normalizeFormat(formatText))
		out[format] = domain.HeadToHeadStats{
			Total:    total,
			Wins:     wins,
			Losses:   losses,
			CoLosses: coLosses,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("head-to-head by format: %w", err)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}
