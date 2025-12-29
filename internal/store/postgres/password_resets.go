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

type PasswordResetStore struct {
	pool *pgxpool.Pool
}

func NewPasswordResetStore(pool *pgxpool.Pool) *PasswordResetStore {
	return &PasswordResetStore{pool: pool}
}

func (s *PasswordResetStore) CreateResetToken(ctx context.Context, token domain.PasswordResetToken) error {
	const q = `
		INSERT INTO password_reset_tokens (
			user_id, token_hash, sent_to_email, created_by_admin_id, created_at, expires_at, used_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	var createdBy any
	if token.CreatedBy != "" {
		createdBy = token.CreatedBy
	}
	_, err := s.pool.Exec(ctx, q,
		token.UserID,
		token.TokenHash,
		token.SentToEmail,
		createdBy,
		token.CreatedAt,
		token.ExpiresAt,
		token.UsedAt,
	)
	if err != nil {
		return fmt.Errorf("create reset token: %w", err)
	}
	return nil
}

func (s *PasswordResetStore) GetResetTokenByHash(ctx context.Context, tokenHash string) (domain.PasswordResetToken, error) {
	const q = `
		SELECT id, user_id, token_hash, sent_to_email, created_by_admin_id, created_at, expires_at, used_at
		FROM password_reset_tokens
		WHERE token_hash = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var (
		token       domain.PasswordResetToken
		idUUID      pgtype.UUID
		userIDUUID  pgtype.UUID
		createdBy   pgtype.UUID
		usedAt      pgtype.Timestamptz
		createdAt   pgtype.Timestamptz
		expiresAt   pgtype.Timestamptz
		sentToEmail string
	)
	err := s.pool.QueryRow(ctx, q, tokenHash).Scan(
		&idUUID,
		&userIDUUID,
		&token.TokenHash,
		&sentToEmail,
		&createdBy,
		&createdAt,
		&expiresAt,
		&usedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.PasswordResetToken{}, domain.ErrNotFound
		}
		return domain.PasswordResetToken{}, fmt.Errorf("get reset token: %w", err)
	}
	token.ID = uuidOrEmpty(idUUID)
	token.UserID = uuidOrEmpty(userIDUUID)
	token.SentToEmail = sentToEmail
	token.CreatedBy = uuidOrEmpty(createdBy)
	if createdAt.Valid {
		token.CreatedAt = createdAt.Time
	}
	if expiresAt.Valid {
		token.ExpiresAt = expiresAt.Time
	}
	token.UsedAt = timestamptzPtr(usedAt)
	return token, nil
}

func (s *PasswordResetStore) MarkResetTokenUsed(ctx context.Context, tokenHash string, when time.Time) error {
	const q = `
		UPDATE password_reset_tokens
		SET used_at = $2
		WHERE token_hash = $1
	`
	tag, err := s.pool.Exec(ctx, q, tokenHash, when)
	if err != nil {
		return fmt.Errorf("mark reset token used: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
