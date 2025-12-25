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

func (s *UsersStore) SetPasswordHash(ctx context.Context, userID, passwordHash string) error {
	const q = `
		UPDATE users
		SET password_hash = $2, updated_at = now()
		WHERE id = $1
	`
	tag, err := s.pool.Exec(ctx, q, userID, passwordHash)
	if err != nil {
		return fmt.Errorf("set password hash: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *UsersStore) GetUserByExternalAccount(ctx context.Context, provider, providerID string) (domain.User, domain.ExternalAccount, error) {
	const q = `
		SELECT
			u.id, u.email, u.username, u.status, u.created_at, u.updated_at, u.last_login_at,
			ea.id, ea.user_id, ea.provider, ea.provider_id, ea.email, ea.created_at
		FROM user_external_accounts ea
		JOIN users u ON u.id = ea.user_id
		WHERE ea.provider = $1 AND ea.provider_id = $2
		LIMIT 1
	`

	var (
		u           domain.User
		ext         domain.ExternalAccount
		userIDUUID  pgtype.UUID
		emailText   pgtype.Text
		lastLoginTS pgtype.Timestamptz
		extIDUUID   pgtype.UUID
		extUserUUID pgtype.UUID
		extEmail    pgtype.Text
	)
	err := s.pool.QueryRow(ctx, q, provider, providerID).Scan(
		&userIDUUID,
		&emailText,
		&u.Username,
		&u.Status,
		&u.CreatedAt,
		&u.UpdatedAt,
		&lastLoginTS,
		&extIDUUID,
		&extUserUUID,
		&ext.Provider,
		&ext.ProviderID,
		&extEmail,
		&ext.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ExternalAccount{}, domain.ErrNotFound
		}
		return domain.User{}, domain.ExternalAccount{}, fmt.Errorf("get user by external account: %w", err)
	}

	u.ID = uuidOrEmpty(userIDUUID)
	u.Email = textOrEmpty(emailText)
	u.LastLoginAt = timestamptzPtr(lastLoginTS)

	ext.ID = uuidOrEmpty(extIDUUID)
	ext.UserID = uuidOrEmpty(extUserUUID)
	ext.Email = textOrEmpty(extEmail)

	return u, ext, nil
}

func (s *UsersStore) CreateUserWithExternalAccount(ctx context.Context, provider, providerID, email, username, passwordHash string) (domain.User, domain.ExternalAccount, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, domain.ExternalAccount{}, fmt.Errorf("begin create user with external account: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	const userQ = `
		INSERT INTO users (email, username, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, email, username, status, created_at, updated_at, last_login_at
	`

	var (
		u           domain.User
		userIDUUID  pgtype.UUID
		emailText   pgtype.Text
		lastLoginTS pgtype.Timestamptz
	)
	err = tx.QueryRow(ctx, userQ, email, username, passwordHash).Scan(
		&userIDUUID,
		&emailText,
		&u.Username,
		&u.Status,
		&u.CreatedAt,
		&u.UpdatedAt,
		&lastLoginTS,
	)
	if err != nil {
		return domain.User{}, domain.ExternalAccount{}, mapUserWriteError(err)
	}

	const extQ = `
		INSERT INTO user_external_accounts (user_id, provider, provider_id, email)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	var (
		ext       domain.ExternalAccount
		extIDUUID pgtype.UUID
		extTime   time.Time
	)
	err = tx.QueryRow(ctx, extQ, uuidOrEmpty(userIDUUID), provider, providerID, email).Scan(&extIDUUID, &extTime)
	if err != nil {
		return domain.User{}, domain.ExternalAccount{}, mapExternalAccountWriteError(err)
	}

	if err = tx.Commit(ctx); err != nil {
		return domain.User{}, domain.ExternalAccount{}, fmt.Errorf("commit create user with external account: %w", err)
	}

	u.ID = uuidOrEmpty(userIDUUID)
	u.Email = textOrEmpty(emailText)
	u.LastLoginAt = timestamptzPtr(lastLoginTS)

	ext.ID = uuidOrEmpty(extIDUUID)
	ext.UserID = u.ID
	ext.Provider = provider
	ext.ProviderID = providerID
	ext.Email = email
	ext.CreatedAt = extTime

	return u, ext, nil
}

func (s *UsersStore) LinkExternalAccount(ctx context.Context, userID, provider, providerID, email string) (domain.ExternalAccount, error) {
	const lookupQ = `
		SELECT id, user_id, provider, provider_id, email, created_at
		FROM user_external_accounts
		WHERE provider = $1 AND provider_id = $2
		LIMIT 1
	`

	var (
		ext       domain.ExternalAccount
		extIDUUID pgtype.UUID
		userUUID  pgtype.UUID
		emailText pgtype.Text
	)
	err := s.pool.QueryRow(ctx, lookupQ, provider, providerID).Scan(
		&extIDUUID,
		&userUUID,
		&ext.Provider,
		&ext.ProviderID,
		&emailText,
		&ext.CreatedAt,
	)
	if err == nil {
		ext.ID = uuidOrEmpty(extIDUUID)
		ext.UserID = uuidOrEmpty(userUUID)
		ext.Email = textOrEmpty(emailText)
		if ext.UserID != userID {
			return domain.ExternalAccount{}, domain.ErrExternalAccountExists
		}
		return ext, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.ExternalAccount{}, fmt.Errorf("lookup external account: %w", err)
	}

	const insertQ = `
		INSERT INTO user_external_accounts (user_id, provider, provider_id, email)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	var createdAt time.Time
	err = s.pool.QueryRow(ctx, insertQ, userID, provider, providerID, email).Scan(&extIDUUID, &createdAt)
	if err != nil {
		return domain.ExternalAccount{}, mapExternalAccountWriteError(err)
	}

	ext.ID = uuidOrEmpty(extIDUUID)
	ext.UserID = userID
	ext.Provider = provider
	ext.ProviderID = providerID
	ext.Email = email
	ext.CreatedAt = createdAt
	return ext, nil
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

func mapExternalAccountWriteError(err error) error {
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && pgerr.Code == "23505" {
		if pgerr.ConstraintName == "user_external_accounts_provider_provider_id_uq" {
			return domain.ErrExternalAccountExists
		}
		return fmt.Errorf("unique violation (%s): %w", pgerr.ConstraintName, err)
	}
	return fmt.Errorf("create external account: %w", err)
}

// helpers in scan.go
