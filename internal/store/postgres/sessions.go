package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"MtgLeaderwebserver/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionsStore struct {
	pool *pgxpool.Pool
}

func NewSessionsStore(pool *pgxpool.Pool) *SessionsStore {
	return &SessionsStore{pool: pool}
}

func (s *SessionsStore) CreateSession(ctx context.Context, userID string, expiresAt time.Time, ip, userAgent string) (string, error) {
	const q = `
		INSERT INTO sessions (user_id, expires_at, ip, user_agent)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	var idUUID pgtype.UUID
	err := s.pool.QueryRow(ctx, q, userID, expiresAt, nullIfEmpty(ip), nullIfEmpty(userAgent)).Scan(&idUUID)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	return uuidOrEmpty(idUUID), nil
}

func (s *SessionsStore) GetSession(ctx context.Context, sessionID string) (domain.Session, error) {
	const q = `
		SELECT id, user_id, created_at, expires_at, revoked_at
		FROM sessions
		WHERE id = $1 AND revoked_at IS NULL AND expires_at > now()
	`

	var (
		sess      domain.Session
		idUUID    pgtype.UUID
		userIDUU  pgtype.UUID
		revokedTS pgtype.Timestamptz
	)
	err := s.pool.QueryRow(ctx, q, sessionID).Scan(
		&idUUID,
		&userIDUU,
		&sess.CreatedAt,
		&sess.ExpiresAt,
		&revokedTS,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Session{}, domain.ErrNotFound
		}
		return domain.Session{}, fmt.Errorf("get session: %w", err)
	}

	sess.ID = uuidOrEmpty(idUUID)
	sess.UserID = uuidOrEmpty(userIDUU)
	sess.RevokedAt = timestamptzPtr(revokedTS)
	return sess, nil
}

func (s *SessionsStore) RevokeSession(ctx context.Context, sessionID string, when time.Time) error {
	const q = `
		UPDATE sessions
		SET revoked_at = $2
		WHERE id = $1 AND revoked_at IS NULL
	`

	_, err := s.pool.Exec(ctx, q, sessionID, when)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}
