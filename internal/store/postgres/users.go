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

type UsersStore struct {
	pool *pgxpool.Pool
}

func NewUsersStore(pool *pgxpool.Pool) *UsersStore {
	return &UsersStore{pool: pool}
}

func (s *UsersStore) CreateUser(ctx context.Context, email, username, passwordHash string) (domain.User, error) {
	const q = `
		INSERT INTO users (email, username, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, email, username, status, created_at, updated_at, last_login_at
	`

	var (
		u           domain.User
		idUUID      pgtype.UUID
		emailText   pgtype.Text
		lastLoginTS pgtype.Timestamptz
	)
	err := s.pool.QueryRow(ctx, q, nullIfEmpty(email), username, passwordHash).Scan(
		&idUUID,
		&emailText,
		&u.Username,
		&u.Status,
		&u.CreatedAt,
		&u.UpdatedAt,
		&lastLoginTS,
	)
	if err != nil {
		return domain.User{}, mapUserWriteError(err)
	}

	u.ID = uuidOrEmpty(idUUID)
	u.Email = textOrEmpty(emailText)
	u.LastLoginAt = timestamptzPtr(lastLoginTS)
	return u, nil
}

func (s *UsersStore) GetUserByID(ctx context.Context, id string) (domain.User, error) {
	const q = `
		SELECT id, email, username, status, created_at, updated_at, last_login_at
		FROM users
		WHERE id = $1
	`

	var (
		u           domain.User
		idUUID      pgtype.UUID
		emailText   pgtype.Text
		lastLoginTS pgtype.Timestamptz
	)
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&idUUID,
		&emailText,
		&u.Username,
		&u.Status,
		&u.CreatedAt,
		&u.UpdatedAt,
		&lastLoginTS,
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
	return u, nil
}

func (s *UsersStore) GetUserByLogin(ctx context.Context, login string) (domain.UserWithPassword, error) {
	const q = `
		SELECT id, email, username, password_hash, status, created_at, updated_at, last_login_at
		FROM users
		WHERE username = $1 OR (email IS NOT NULL AND email = $1)
		ORDER BY (username = $1) DESC
		LIMIT 1
	`

	var (
		u           domain.UserWithPassword
		idUUID      pgtype.UUID
		emailText   pgtype.Text
		lastLoginTS pgtype.Timestamptz
	)
	err := s.pool.QueryRow(ctx, q, login).Scan(
		&idUUID,
		&emailText,
		&u.Username,
		&u.PasswordHash,
		&u.Status,
		&u.CreatedAt,
		&u.UpdatedAt,
		&lastLoginTS,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.UserWithPassword{}, domain.ErrNotFound
		}
		return domain.UserWithPassword{}, fmt.Errorf("get user by login: %w", err)
	}

	u.ID = uuidOrEmpty(idUUID)
	u.Email = textOrEmpty(emailText)
	u.LastLoginAt = timestamptzPtr(lastLoginTS)
	return u, nil
}

func (s *UsersStore) GetUserByEmail(ctx context.Context, email string) (domain.UserWithPassword, error) {
	const q = `
		SELECT id, email, username, password_hash, status, created_at, updated_at, last_login_at
		FROM users
		WHERE email = $1
		LIMIT 1
	`

	var (
		u           domain.UserWithPassword
		idUUID      pgtype.UUID
		emailText   pgtype.Text
		lastLoginTS pgtype.Timestamptz
	)
	err := s.pool.QueryRow(ctx, q, email).Scan(
		&idUUID,
		&emailText,
		&u.Username,
		&u.PasswordHash,
		&u.Status,
		&u.CreatedAt,
		&u.UpdatedAt,
		&lastLoginTS,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.UserWithPassword{}, domain.ErrNotFound
		}
		return domain.UserWithPassword{}, fmt.Errorf("get user by email: %w", err)
	}

	u.ID = uuidOrEmpty(idUUID)
	u.Email = textOrEmpty(emailText)
	u.LastLoginAt = timestamptzPtr(lastLoginTS)
	return u, nil
}

func (s *UsersStore) SetLastLogin(ctx context.Context, userID string, when time.Time) error {
	const q = `
		UPDATE users
		SET last_login_at = $2, updated_at = now()
		WHERE id = $1
	`
	_, err := s.pool.Exec(ctx, q, userID, when)
	if err != nil {
		return fmt.Errorf("set last login: %w", err)
	}
	return nil
}

func mapUserWriteError(err error) error {
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && pgerr.Code == "23505" {
		switch pgerr.ConstraintName {
		case "users_username_uq":
			return domain.ErrUsernameTaken
		case "users_email_uq":
			return domain.ErrEmailTaken
		default:
			return fmt.Errorf("unique violation (%s): %w", pgerr.ConstraintName, err)
		}
	}
	return fmt.Errorf("create user: %w", err)
}

// helpers in scan.go
