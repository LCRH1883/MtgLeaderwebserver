package postgres

import (
	"context"
	"fmt"
	"time"

	"MtgLeaderwebserver/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationTokensStore struct {
	pool *pgxpool.Pool
}

func NewNotificationTokensStore(pool *pgxpool.Pool) *NotificationTokensStore {
	return &NotificationTokensStore{pool: pool}
}

func (s *NotificationTokensStore) UpsertToken(ctx context.Context, userID, token, platform string, when time.Time) (domain.NotificationToken, error) {
	const q = `
		INSERT INTO notification_tokens (user_id, token, platform, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (token)
		DO UPDATE SET
			user_id = EXCLUDED.user_id,
			platform = EXCLUDED.platform,
			updated_at = EXCLUDED.updated_at
		RETURNING id, user_id, token, platform, created_at, updated_at
	`

	var (
		idUUID    pgtype.UUID
		userUUID  pgtype.UUID
		createdAt time.Time
		updatedAt time.Time
	)
	err := s.pool.QueryRow(ctx, q, userID, token, platform, when).Scan(
		&idUUID,
		&userUUID,
		&token,
		&platform,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return domain.NotificationToken{}, fmt.Errorf("upsert notification token: %w", err)
	}

	return domain.NotificationToken{
		ID:        uuidOrEmpty(idUUID),
		UserID:    uuidOrEmpty(userUUID),
		Token:     token,
		Platform:  platform,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func (s *NotificationTokensStore) DeleteToken(ctx context.Context, userID, token string) error {
	const q = `
		DELETE FROM notification_tokens
		WHERE user_id = $1 AND token = $2
	`
	if _, err := s.pool.Exec(ctx, q, userID, token); err != nil {
		return fmt.Errorf("delete notification token: %w", err)
	}
	return nil
}

func (s *NotificationTokensStore) ListTokens(ctx context.Context, userID string) ([]domain.NotificationToken, error) {
	const q = `
		SELECT id, user_id, token, platform, created_at, updated_at
		FROM notification_tokens
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`

	rows, err := s.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list notification tokens: %w", err)
	}
	defer rows.Close()

	var out []domain.NotificationToken
	for rows.Next() {
		var (
			idUUID   pgtype.UUID
			userUUID pgtype.UUID
			token    string
			platform string
			created  time.Time
			updated  time.Time
		)
		if err := rows.Scan(&idUUID, &userUUID, &token, &platform, &created, &updated); err != nil {
			return nil, fmt.Errorf("scan notification token: %w", err)
		}
		out = append(out, domain.NotificationToken{
			ID:        uuidOrEmpty(idUUID),
			UserID:    uuidOrEmpty(userUUID),
			Token:     token,
			Platform:  platform,
			CreatedAt: created,
			UpdatedAt: updated,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list notification tokens: %w", err)
	}
	return out, nil
}

var _ = pgx.ErrNoRows
