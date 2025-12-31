package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"MtgLeaderwebserver/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
		SELECT id, email, username, status, created_at, updated_at, last_login_at, display_name, avatar_path, avatar_updated_at
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
			u              domain.User
			idUUID         pgtype.UUID
			emailText      pgtype.Text
			lastLoginTS    pgtype.Timestamptz
			displayName    pgtype.Text
			avatarPathText pgtype.Text
			avatarUpdated  pgtype.Timestamptz
		)
		if err := rows.Scan(&idUUID, &emailText, &u.Username, &u.Status, &u.CreatedAt, &u.UpdatedAt, &lastLoginTS, &displayName, &avatarPathText, &avatarUpdated); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		u.ID = uuidOrEmpty(idUUID)
		u.Email = textOrEmpty(emailText)
		u.LastLoginAt = timestamptzPtr(lastLoginTS)
		u.DisplayName = textOrEmpty(displayName)
		u.AvatarPath = textOrEmpty(avatarPathText)
		u.AvatarUpdatedAt = timestamptzPtr(avatarUpdated)
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	return out, nil
}

func (s *AdminUsersStore) GetUserByID(ctx context.Context, id string) (domain.User, error) {
	const q = `
		SELECT id, email, username, status, created_at, updated_at, last_login_at, display_name, avatar_path, avatar_updated_at
		FROM users
		WHERE id = $1
	`

	var (
		u              domain.User
		idUUID         pgtype.UUID
		emailText      pgtype.Text
		lastLoginTS    pgtype.Timestamptz
		displayName    pgtype.Text
		avatarPathText pgtype.Text
		avatarUpdated  pgtype.Timestamptz
	)
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&idUUID,
		&emailText,
		&u.Username,
		&u.Status,
		&u.CreatedAt,
		&u.UpdatedAt,
		&lastLoginTS,
		&displayName,
		&avatarPathText,
		&avatarUpdated,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, fmt.Errorf("get user by id: %w", err)
	}
	u.ID = uuidOrEmpty(idUUID)
	u.Email = textOrEmpty(emailText)
	u.LastLoginAt = timestamptzPtr(lastLoginTS)
	u.DisplayName = textOrEmpty(displayName)
	u.AvatarPath = textOrEmpty(avatarPathText)
	u.AvatarUpdatedAt = timestamptzPtr(avatarUpdated)
	return u, nil
}

func (s *AdminUsersStore) SetUserEmail(ctx context.Context, userID, email string) error {
	const q = `
		UPDATE users
		SET email = $2, updated_at = date_trunc('milliseconds', now())
		WHERE id = $1
	`
	tag, err := s.pool.Exec(ctx, q, userID, nullIfEmpty(email))
	if err != nil {
		return mapUserEmailError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *AdminUsersStore) DeleteUser(ctx context.Context, userID string) error {
	const q = `
		DELETE FROM users
		WHERE id = $1
	`
	tag, err := s.pool.Exec(ctx, q, userID)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *AdminUsersStore) SearchUsers(ctx context.Context, query string, limit, offset int) ([]domain.User, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return []domain.User{}, nil
	}

	like := "%" + query + "%"
	const q = `
		SELECT id, email, username, status, created_at, updated_at, last_login_at, display_name, avatar_path, avatar_updated_at
		FROM users
		WHERE id::text ILIKE $1
		   OR username ILIKE $1
		   OR email ILIKE $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.pool.Query(ctx, q, like, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer rows.Close()

	var out []domain.User
	for rows.Next() {
		var (
			u              domain.User
			idUUID         pgtype.UUID
			emailText      pgtype.Text
			lastLoginTS    pgtype.Timestamptz
			displayName    pgtype.Text
			avatarPathText pgtype.Text
			avatarUpdated  pgtype.Timestamptz
		)
		if err := rows.Scan(&idUUID, &emailText, &u.Username, &u.Status, &u.CreatedAt, &u.UpdatedAt, &lastLoginTS, &displayName, &avatarPathText, &avatarUpdated); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		u.ID = uuidOrEmpty(idUUID)
		u.Email = textOrEmpty(emailText)
		u.LastLoginAt = timestamptzPtr(lastLoginTS)
		u.DisplayName = textOrEmpty(displayName)
		u.AvatarPath = textOrEmpty(avatarPathText)
		u.AvatarUpdatedAt = timestamptzPtr(avatarUpdated)
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}

	return out, nil
}

func mapUserEmailError(err error) error {
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && pgerr.Code == "23505" && pgerr.ConstraintName == "users_email_uq" {
		return domain.ErrEmailTaken
	}
	return fmt.Errorf("set user email: %w", err)
}
