package postgres

import (
	"context"
	"fmt"

	"MtgLeaderwebserver/internal/domain"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminUsersStore struct {
	pool *pgxpool.Pool
}

func NewAdminUsersStore(pool *pgxpool.Pool) *AdminUsersStore {
	return &AdminUsersStore{pool: pool}
}

func (s *AdminUsersStore) ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	const q = `
		SELECT id, email, username, status, created_at, updated_at, last_login_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := s.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var out []domain.User
	for rows.Next() {
		var (
			u           domain.User
			idUUID      pgtype.UUID
			emailText   pgtype.Text
			lastLoginTS pgtype.Timestamptz
		)
		if err := rows.Scan(&idUUID, &emailText, &u.Username, &u.Status, &u.CreatedAt, &u.UpdatedAt, &lastLoginTS); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		u.ID = uuidOrEmpty(idUUID)
		u.Email = textOrEmpty(emailText)
		u.LastLoginAt = timestamptzPtr(lastLoginTS)
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	return out, nil
}
