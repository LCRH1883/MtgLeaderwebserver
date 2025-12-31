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

func (s *MatchesStore) CreateMatch(ctx context.Context, createdBy string, startedAt, endedAt, playedAt *time.Time, winnerID string, participants []domain.MatchParticipantInput, format domain.GameFormat, totalDurationSeconds, turnCount int, clientRef string, updatedAt time.Time) (string, bool, error) {
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
		INSERT INTO matches (created_by, played_at, winner_id, format, total_duration_seconds, turn_count, client_ref, updated_at, started_at, ended_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	var matchIDUUID pgtype.UUID
	var playedAtAny any
	if playedAt != nil {
		playedAtAny = *playedAt
	} else if endedAt != nil {
		playedAtAny = *endedAt
	}
	var winnerIDAny any
	if winnerID != "" {
		winnerIDAny = winnerID
	}
	var startedAtAny any
	if startedAt != nil {
		startedAtAny = *startedAt
	}
	var endedAtAny any
	if endedAt != nil {
		endedAtAny = *endedAt
	}
	if err := tx.QueryRow(ctx, insertMatch, createdBy, playedAtAny, winnerIDAny, format, totalDurationSeconds, turnCount, nullIfEmpty(clientRef), updatedAt, startedAtAny, endedAtAny).Scan(&matchIDUUID); err != nil {
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

	const insertParticipant = `
		INSERT INTO match_participants (
			match_id, seat_index, user_id, guest_name, display_name, place,
			eliminated_turn_number, eliminated_during_seat_index, total_turn_time_ms, turns_taken
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	for _, participant := range participants {
		var userIDAny any
		if participant.UserID != "" {
			userIDAny = participant.UserID
		}
		var guestNameAny any
		if participant.GuestName != "" {
			guestNameAny = participant.GuestName
		}
		var eliminatedTurnAny any
		if participant.EliminatedTurn != nil {
			eliminatedTurnAny = *participant.EliminatedTurn
		}
		var eliminatedDuringAny any
		if participant.EliminatedDuring != nil {
			eliminatedDuringAny = *participant.EliminatedDuring
		}
		var totalTurnTimeAny any
		if participant.TotalTurnTimeMs != nil {
			totalTurnTimeAny = *participant.TotalTurnTimeMs
		}
		var turnsTakenAny any
		if participant.TurnsTaken != nil {
			turnsTakenAny = *participant.TurnsTaken
		}
		if _, err := tx.Exec(
			ctx,
			insertParticipant,
			matchID,
			participant.SeatIndex,
			userIDAny,
			guestNameAny,
			participant.DisplayName,
			participant.Place,
			eliminatedTurnAny,
			eliminatedDuringAny,
			totalTurnTimeAny,
			turnsTakenAny,
		); err != nil {
			var pgerr *pgconn.PgError
			if errors.As(err, &pgerr) && pgerr.Code == "23503" {
				return "", false, domain.ErrValidation
			}
			return "", false, fmt.Errorf("insert match participant: %w", err)
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

	// List matches where user participated (new participants table or legacy match_players).
	const q = `
		SELECT m.id, m.created_by, m.created_at, m.updated_at, m.started_at, m.ended_at, m.played_at, m.winner_id,
		       m.format, m.total_duration_seconds, m.turn_count, m.client_ref
		FROM matches m
		WHERE
			EXISTS (SELECT 1 FROM match_participants p WHERE p.match_id = m.id AND p.user_id = $1)
			OR (
				NOT EXISTS (SELECT 1 FROM match_participants p WHERE p.match_id = m.id)
				AND EXISTS (SELECT 1 FROM match_players mp WHERE mp.match_id = m.id AND mp.user_id = $1)
			)
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
		startedAt    pgtype.Timestamptz
		endedAt      pgtype.Timestamptz
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
		if err := rows.Scan(&r.idUUID, &r.createdBy, &r.createdAt, &r.updatedAt, &r.startedAt, &r.endedAt, &r.playedAt, &r.winnerID, &r.format, &r.durationSecs, &r.turnCount, &r.clientRef); err != nil {
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
			StartedAt:            timestamptzPtr(r.startedAt),
			EndedAt:              timestamptzPtr(r.endedAt),
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
		SELECT m.id, m.created_by, m.created_at, m.updated_at, m.started_at, m.ended_at, m.played_at, m.winner_id,
		       m.format, m.total_duration_seconds, m.turn_count, m.client_ref
		FROM matches m
		WHERE m.id = $1
		  AND (
		    EXISTS (SELECT 1 FROM match_participants p WHERE p.match_id = m.id AND p.user_id = $2)
		    OR (
		      NOT EXISTS (SELECT 1 FROM match_participants p WHERE p.match_id = m.id)
		      AND EXISTS (SELECT 1 FROM match_players mp WHERE mp.match_id = m.id AND mp.user_id = $2)
		    )
		  )
		LIMIT 1
	`
	var (
		idUUID       pgtype.UUID
		createdBy    pgtype.UUID
		createdAt    time.Time
		updatedAt    time.Time
		startedAt    pgtype.Timestamptz
		endedAt      pgtype.Timestamptz
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
		&startedAt,
		&endedAt,
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
		StartedAt:            timestamptzPtr(startedAt),
		EndedAt:              timestamptzPtr(endedAt),
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
		SELECT m.id, m.created_by, m.created_at, m.updated_at, m.started_at, m.ended_at, m.played_at, m.winner_id,
		       m.format, m.total_duration_seconds, m.turn_count, m.client_ref
		FROM matches m
		WHERE m.created_by = $1 AND m.client_ref = $2
		LIMIT 1
	`
	var (
		idUUID        pgtype.UUID
		createdByUUID pgtype.UUID
		createdAt     time.Time
		updatedAt     time.Time
		startedAt     pgtype.Timestamptz
		endedAt       pgtype.Timestamptz
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
		&startedAt,
		&endedAt,
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
		StartedAt:            timestamptzPtr(startedAt),
		EndedAt:              timestamptzPtr(endedAt),
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
		SELECT
			p.seat_index,
			p.user_id,
			u.username,
			u.display_name,
			p.guest_name,
			p.display_name,
			p.place,
			p.eliminated_turn_number,
			p.eliminated_during_seat_index,
			p.total_turn_time_ms,
			p.turns_taken
		FROM match_participants p
		LEFT JOIN users u ON u.id = p.user_id
		WHERE p.match_id = $1
		ORDER BY p.place ASC, p.seat_index ASC
	`

	rows, err := s.pool.Query(ctx, q, matchID)
	if err != nil {
		return nil, fmt.Errorf("list match participants: %w", err)
	}
	defer rows.Close()

	var out []domain.MatchPlayer
	for rows.Next() {
		var (
			seatIndex        int
			userID           pgtype.UUID
			username         pgtype.Text
			userDisplayName  pgtype.Text
			guestName        pgtype.Text
			displayName      string
			place            int
			eliminatedTurn   pgtype.Int4
			eliminatedDuring pgtype.Int4
			totalTurnTimeMs  pgtype.Int8
			turnsTaken       pgtype.Int4
		)
		if err := rows.Scan(&seatIndex, &userID, &username, &userDisplayName, &guestName, &displayName, &place, &eliminatedTurn, &eliminatedDuring, &totalTurnTimeMs, &turnsTaken); err != nil {
			return nil, fmt.Errorf("scan match participant: %w", err)
		}
		id := uuidOrEmpty(userID)
		name := textOrEmpty(username)
		display := strings.TrimSpace(displayName)
		if display == "" {
			display = textOrEmpty(userDisplayName)
		}

		placeCopy := place
		seatCopy := seatIndex
		elimTurn := int4Ptr(eliminatedTurn)
		elimDuring := int4Ptr(eliminatedDuring)
		turns := int4Ptr(turnsTaken)
		var totalMs *int64
		if totalTurnTimeMs.Valid {
			v := totalTurnTimeMs.Int64
			totalMs = &v
		}

		isWinner := false
		if winnerID != "" {
			isWinner = id != "" && id == winnerID
		} else if place == 1 {
			isWinner = true
		}

		out = append(out, domain.MatchPlayer{
			User:             domain.UserSummary{ID: id, Username: name, DisplayName: display},
			IsWinner:         isWinner,
			Rank:             &placeCopy,
			SeatIndex:        &seatCopy,
			GuestName:        textOrEmpty(guestName),
			DisplayName:      display,
			Place:            &placeCopy,
			EliminatedTurn:   elimTurn,
			EliminationTurn:  elimTurn,
			EliminatedDuring: elimDuring,
			TotalTurnTimeMs:  totalMs,
			TurnsTaken:       turns,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list match participants: %w", err)
	}
	if len(out) > 0 {
		return out, nil
	}
	return s.listLegacyPlayers(ctx, matchID, winnerID)
}

func (s *MatchesStore) listLegacyPlayers(ctx context.Context, matchID string, winnerID string) ([]domain.MatchPlayer, error) {
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
			Place:            int4Ptr(rank),
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
		WITH participants AS (
			SELECT match_id, user_id, guest_name, place
			FROM match_participants
			UNION ALL
			SELECT mp.match_id, mp.user_id, NULL::text AS guest_name,
			       CASE
			         WHEN m.winner_id = mp.user_id THEN 1
			         WHEN m.winner_id IS NOT NULL THEN 2
			         ELSE NULL
			       END AS place
			FROM match_players mp
			JOIN matches m ON m.id = mp.match_id
			WHERE NOT EXISTS (SELECT 1 FROM match_participants p WHERE p.match_id = mp.match_id)
		),
		completed AS (
			SELECT DISTINCT match_id FROM participants WHERE place = 1
		),
		user_matches AS (
			SELECT DISTINCT p.match_id
			FROM participants p
			JOIN completed c ON c.match_id = p.match_id
			WHERE p.user_id = $1
		)
		SELECT
			COUNT(*)::int AS matches_played,
			COALESCE(SUM(CASE WHEN p.place = 1 THEN 1 ELSE 0 END), 0)::int AS wins,
			COALESCE(SUM(CASE WHEN m.turn_count > 0 THEN m.total_duration_seconds ELSE 0 END), 0)::int AS total_seconds,
			COALESCE(SUM(CASE WHEN m.turn_count > 0 THEN m.turn_count ELSE 0 END), 0)::int AS total_turns
		FROM user_matches um
		JOIN matches m ON m.id = um.match_id
		JOIN participants p ON p.match_id = um.match_id AND p.user_id = $1
	`
	var played, wins, totalSeconds, totalTurns int
	if err := s.pool.QueryRow(ctx, q, userID).Scan(&played, &wins, &totalSeconds, &totalTurns); err != nil {
		return domain.StatsSummary{}, fmt.Errorf("stats summary: %w", err)
	}
	losses := played - wins
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
	guestHeadToHead, err := s.guestHeadToHead(ctx, userID)
	if err != nil {
		return domain.StatsSummary{}, err
	}
	winPct := 0.0
	if played > 0 {
		winPct = float64(wins) / float64(played)
	}

	return domain.StatsSummary{
		MatchesPlayed:     played,
		Wins:              wins,
		Losses:            losses,
		WinPct:            winPct,
		AvgTurnSeconds:    avgTurn,
		ByFormat:          byFormat,
		MostOftenBeat:     mostBeat,
		MostOftenBeatsYou: mostBeats,
		GuestHeadToHead:   guestHeadToHead,
	}, nil
}

func (s *MatchesStore) statsSummaryByFormat(ctx context.Context, userID string) (map[string]domain.StatsSummary, error) {
	const q = `
		WITH participants AS (
			SELECT match_id, user_id, guest_name, place
			FROM match_participants
			UNION ALL
			SELECT mp.match_id, mp.user_id, NULL::text AS guest_name,
			       CASE
			         WHEN m.winner_id = mp.user_id THEN 1
			         WHEN m.winner_id IS NOT NULL THEN 2
			         ELSE NULL
			       END AS place
			FROM match_players mp
			JOIN matches m ON m.id = mp.match_id
			WHERE NOT EXISTS (SELECT 1 FROM match_participants p WHERE p.match_id = mp.match_id)
		),
		completed AS (
			SELECT DISTINCT match_id FROM participants WHERE place = 1
		),
		user_matches AS (
			SELECT DISTINCT p.match_id
			FROM participants p
			JOIN completed c ON c.match_id = p.match_id
			WHERE p.user_id = $1
		)
		SELECT
			m.format,
			COUNT(*)::int AS matches_played,
			COALESCE(SUM(CASE WHEN p.place = 1 THEN 1 ELSE 0 END), 0)::int AS wins,
			COALESCE(SUM(CASE WHEN m.turn_count > 0 THEN m.total_duration_seconds ELSE 0 END), 0)::int AS total_seconds,
			COALESCE(SUM(CASE WHEN m.turn_count > 0 THEN m.turn_count ELSE 0 END), 0)::int AS total_turns
		FROM user_matches um
		JOIN matches m ON m.id = um.match_id
		JOIN participants p ON p.match_id = um.match_id AND p.user_id = $1
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
			totalSeconds int
			totalTurns   int
		)
		if err := rows.Scan(&formatText, &played, &wins, &totalSeconds, &totalTurns); err != nil {
			return nil, fmt.Errorf("scan stats by format: %w", err)
		}
		avgTurn := 0
		if totalTurns > 0 {
			avgTurn = totalSeconds / totalTurns
		}
		format := string(normalizeFormat(formatText))
		losses := played - wins
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
		WITH participants AS (
			SELECT match_id, user_id, guest_name, place
			FROM match_participants
			UNION ALL
			SELECT mp.match_id, mp.user_id, NULL::text AS guest_name,
			       CASE
			         WHEN m.winner_id = mp.user_id THEN 1
			         WHEN m.winner_id IS NOT NULL THEN 2
			         ELSE NULL
			       END AS place
			FROM match_players mp
			JOIN matches m ON m.id = mp.match_id
			WHERE NOT EXISTS (SELECT 1 FROM match_participants p WHERE p.match_id = mp.match_id)
		),
		winners AS (
			SELECT match_id
			FROM participants
			WHERE user_id = $1 AND place = 1
		)
		SELECT u.id, u.username, COUNT(*)::int AS wins
		FROM participants p
		JOIN winners w ON w.match_id = p.match_id
		JOIN users u ON u.id = p.user_id
		WHERE p.user_id <> $1
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
		WITH participants AS (
			SELECT match_id, user_id, guest_name, place
			FROM match_participants
			UNION ALL
			SELECT mp.match_id, mp.user_id, NULL::text AS guest_name,
			       CASE
			         WHEN m.winner_id = mp.user_id THEN 1
			         WHEN m.winner_id IS NOT NULL THEN 2
			         ELSE NULL
			       END AS place
			FROM match_players mp
			JOIN matches m ON m.id = mp.match_id
			WHERE NOT EXISTS (SELECT 1 FROM match_participants p WHERE p.match_id = mp.match_id)
		),
		winners AS (
			SELECT match_id, user_id
			FROM participants
			WHERE place = 1 AND user_id IS NOT NULL
		)
		SELECT u.id, u.username, COUNT(*)::int AS wins
		FROM winners w
		JOIN users u ON u.id = w.user_id
		WHERE w.user_id <> $1
		  AND EXISTS (SELECT 1 FROM participants p WHERE p.match_id = w.match_id AND p.user_id = $1)
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

func (s *MatchesStore) guestHeadToHead(ctx context.Context, userID string) ([]domain.GuestHeadToHeadStat, error) {
	const q = `
		WITH participants AS (
			SELECT match_id, user_id, guest_name, place
			FROM match_participants
			UNION ALL
			SELECT mp.match_id, mp.user_id, NULL::text AS guest_name,
			       CASE
			         WHEN m.winner_id = mp.user_id THEN 1
			         WHEN m.winner_id IS NOT NULL THEN 2
			         ELSE NULL
			       END AS place
			FROM match_players mp
			JOIN matches m ON m.id = mp.match_id
			WHERE NOT EXISTS (SELECT 1 FROM match_participants p WHERE p.match_id = mp.match_id)
		),
		completed AS (
			SELECT DISTINCT match_id FROM participants WHERE place = 1
		),
		user_matches AS (
			SELECT DISTINCT p.match_id
			FROM participants p
			JOIN completed c ON c.match_id = p.match_id
			WHERE p.user_id = $1
		),
		winners AS (
			SELECT match_id, user_id, guest_name
			FROM participants
			WHERE place = 1
		)
		SELECT
			g.guest_name,
			COALESCE(SUM(CASE WHEN w.user_id = $1 THEN 1 ELSE 0 END), 0)::int AS wins,
			COALESCE(SUM(CASE WHEN w.guest_name = g.guest_name THEN 1 ELSE 0 END), 0)::int AS losses
		FROM participants g
		JOIN user_matches um ON um.match_id = g.match_id
		JOIN winners w ON w.match_id = g.match_id
		WHERE g.guest_name IS NOT NULL
		GROUP BY g.guest_name
		ORDER BY g.guest_name ASC
	`

	rows, err := s.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("guest head-to-head: %w", err)
	}
	defer rows.Close()

	var out []domain.GuestHeadToHeadStat
	for rows.Next() {
		var name string
		var wins, losses int
		if err := rows.Scan(&name, &wins, &losses); err != nil {
			return nil, fmt.Errorf("scan guest head-to-head: %w", err)
		}
		out = append(out, domain.GuestHeadToHeadStat{
			GuestName: name,
			Wins:      wins,
			Losses:    losses,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("guest head-to-head: %w", err)
	}
	return out, nil
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
		WITH participants AS (
			SELECT match_id, user_id, guest_name, place
			FROM match_participants
			UNION ALL
			SELECT mp.match_id, mp.user_id, NULL::text AS guest_name,
			       CASE
			         WHEN m.winner_id = mp.user_id THEN 1
			         WHEN m.winner_id IS NOT NULL THEN 2
			         ELSE NULL
			       END AS place
			FROM match_players mp
			JOIN matches m ON m.id = mp.match_id
			WHERE NOT EXISTS (SELECT 1 FROM match_participants p WHERE p.match_id = mp.match_id)
		),
		winners AS (
			SELECT match_id, user_id
			FROM participants
			WHERE place = 1 AND user_id IS NOT NULL
		)
		SELECT
			COALESCE(SUM(CASE WHEN w.user_id = $1 THEN 1 ELSE 0 END), 0)::int AS wins,
			COALESCE(SUM(CASE WHEN w.user_id = $2 THEN 1 ELSE 0 END), 0)::int AS losses
		FROM winners w
		WHERE
			EXISTS (SELECT 1 FROM participants p WHERE p.match_id = w.match_id AND p.user_id = $1)
			AND EXISTS (SELECT 1 FROM participants p WHERE p.match_id = w.match_id AND p.user_id = $2)
	`
	var wins, losses int
	if err := s.pool.QueryRow(ctx, q, userID, opponentID).Scan(&wins, &losses); err != nil {
		return domain.HeadToHeadStats{}, fmt.Errorf("head-to-head: %w", err)
	}

	byFormat, err := s.headToHeadByFormat(ctx, userID, opponentID)
	if err != nil {
		return domain.HeadToHeadStats{}, err
	}

	return domain.HeadToHeadStats{
		Opponent: domain.UserSummary{ID: uuidOrEmpty(oppIDUUID), Username: oppUsername},
		Total:    wins + losses,
		Wins:     wins,
		Losses:   losses,
		CoLosses: 0,
		ByFormat: byFormat,
	}, nil
}

func (s *MatchesStore) headToHeadByFormat(ctx context.Context, userID, opponentID string) (map[string]domain.HeadToHeadStats, error) {
	const q = `
		WITH participants AS (
			SELECT match_id, user_id, guest_name, place
			FROM match_participants
			UNION ALL
			SELECT mp.match_id, mp.user_id, NULL::text AS guest_name,
			       CASE
			         WHEN m.winner_id = mp.user_id THEN 1
			         WHEN m.winner_id IS NOT NULL THEN 2
			         ELSE NULL
			       END AS place
			FROM match_players mp
			JOIN matches m ON m.id = mp.match_id
			WHERE NOT EXISTS (SELECT 1 FROM match_participants p WHERE p.match_id = mp.match_id)
		),
		winners AS (
			SELECT match_id, user_id
			FROM participants
			WHERE place = 1 AND user_id IS NOT NULL
		)
		SELECT
			m.format,
			COALESCE(SUM(CASE WHEN w.user_id = $1 THEN 1 ELSE 0 END), 0)::int AS wins,
			COALESCE(SUM(CASE WHEN w.user_id = $2 THEN 1 ELSE 0 END), 0)::int AS losses
		FROM winners w
		JOIN matches m ON m.id = w.match_id
		WHERE
			EXISTS (SELECT 1 FROM participants p WHERE p.match_id = w.match_id AND p.user_id = $1)
			AND EXISTS (SELECT 1 FROM participants p WHERE p.match_id = w.match_id AND p.user_id = $2)
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
			wins       int
			losses     int
		)
		if err := rows.Scan(&formatText, &wins, &losses); err != nil {
			return nil, fmt.Errorf("scan head-to-head by format: %w", err)
		}
		format := string(normalizeFormat(formatText))
		out[format] = domain.HeadToHeadStats{
			Total:    wins + losses,
			Wins:     wins,
			Losses:   losses,
			CoLosses: 0,
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
