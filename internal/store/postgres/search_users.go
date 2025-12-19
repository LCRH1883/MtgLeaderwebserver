package postgres

import (
	"context"
	"fmt"
	"strings"

	"MtgLeaderwebserver/internal/domain"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserSearchStore struct {
	pool *pgxpool.Pool
}

func NewUserSearchStore(pool *pgxpool.Pool) *UserSearchStore {
	return &UserSearchStore{pool: pool}
}

func (s *UserSearchStore) SearchUsers(ctx context.Context, q string, limit int, excludeUserID string) ([]domain.UserSummary, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	q = strings.TrimSpace(q)
	if q == "" {
		return []domain.UserSummary{}, nil
	}

	like := "%" + q + "%"
	const query = `
		SELECT id, username
		FROM users
		WHERE status = 'active'
		  AND id <> $3
		  AND (username ILIKE $1 OR email ILIKE $1)
		ORDER BY username ASC
		LIMIT $2
	`

	rows, err := s.pool.Query(ctx, query, like, limit, excludeUserID)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer rows.Close()

	var out []domain.UserSummary
	for rows.Next() {
		var idUUID pgtype.UUID
		var username string
		if err := rows.Scan(&idUUID, &username); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		out = append(out, domain.UserSummary{ID: uuidOrEmpty(idUUID), Username: username})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}

	return out, nil
}
