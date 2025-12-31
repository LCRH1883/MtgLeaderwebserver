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

type FriendshipsStore struct {
	pool *pgxpool.Pool
}

func NewFriendshipsStore(pool *pgxpool.Pool) *FriendshipsStore {
	return &FriendshipsStore{pool: pool}
}

func (s *FriendshipsStore) CreateRequest(ctx context.Context, requesterID, addresseeID string) (string, time.Time, time.Time, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", time.Time{}, time.Time{}, fmt.Errorf("begin create friend request: %w", err)
	}
	defer tx.Rollback(ctx)

	const clearDeclined = `
		DELETE FROM friendships
		WHERE status = 'declined'
		  AND (
		    (requester_id = $1 AND addressee_id = $2)
		    OR
		    (requester_id = $2 AND addressee_id = $1)
		  )
	`
	if _, err := tx.Exec(ctx, clearDeclined, requesterID, addresseeID); err != nil {
		return "", time.Time{}, time.Time{}, fmt.Errorf("clear declined requests: %w", err)
	}

	const q = `
		INSERT INTO friendships (requester_id, addressee_id, status, created_at, updated_at)
		VALUES ($1, $2, 'pending', date_trunc('milliseconds', now()), date_trunc('milliseconds', now()))
		RETURNING id, created_at, updated_at
	`

	var (
		idUUID    pgtype.UUID
		createdAt time.Time
		updatedAt time.Time
	)
	err = tx.QueryRow(ctx, q, requesterID, addresseeID).Scan(&idUUID, &createdAt, &updatedAt)
	if err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) && pgerr.Code == "23505" && pgerr.ConstraintName == "friendships_pair_uq" {
			return "", time.Time{}, time.Time{}, domain.ErrFriendshipExists
		}
		return "", time.Time{}, time.Time{}, fmt.Errorf("create friend request: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", time.Time{}, time.Time{}, fmt.Errorf("commit friend request: %w", err)
	}

	return uuidOrEmpty(idUUID), createdAt, updatedAt, nil
}

func (s *FriendshipsStore) Accept(ctx context.Context, requestID, addresseeID string, when time.Time, checkUpdatedAt bool) (bool, error) {
	if checkUpdatedAt {
		const q = `
			UPDATE friendships
			SET status = 'accepted', responded_at = $3, updated_at = $3
			WHERE id = $1 AND addressee_id = $2 AND status = 'pending' AND updated_at < $3
		`
		ct, err := s.pool.Exec(ctx, q, requestID, addresseeID, when)
		if err != nil {
			return false, fmt.Errorf("accept friend request: %w", err)
		}
		if ct.RowsAffected() == 1 {
			return true, nil
		}
		exists, err := s.pendingForAddressee(ctx, requestID, addresseeID)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, domain.ErrNotFound
		}
		return false, nil
	}

	const q = `
		UPDATE friendships
		SET status = 'accepted', responded_at = $3, updated_at = $3
		WHERE id = $1 AND addressee_id = $2 AND status = 'pending'
	`
	ct, err := s.pool.Exec(ctx, q, requestID, addresseeID, when)
	if err != nil {
		return false, fmt.Errorf("accept friend request: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return false, domain.ErrNotFound
	}
	return true, nil
}

func (s *FriendshipsStore) Decline(ctx context.Context, requestID, addresseeID string, when time.Time, checkUpdatedAt bool) (bool, error) {
	if checkUpdatedAt {
		const q = `
			UPDATE friendships
			SET status = 'declined', responded_at = $3, updated_at = $3
			WHERE id = $1 AND addressee_id = $2 AND status = 'pending' AND updated_at < $3
		`
		ct, err := s.pool.Exec(ctx, q, requestID, addresseeID, when)
		if err != nil {
			return false, fmt.Errorf("decline friend request: %w", err)
		}
		if ct.RowsAffected() == 1 {
			return true, nil
		}
		exists, err := s.pendingForAddressee(ctx, requestID, addresseeID)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, domain.ErrNotFound
		}
		return false, nil
	}

	const q = `
		UPDATE friendships
		SET status = 'declined', responded_at = $3, updated_at = $3
		WHERE id = $1 AND addressee_id = $2 AND status = 'pending'
	`
	ct, err := s.pool.Exec(ctx, q, requestID, addresseeID, when)
	if err != nil {
		return false, fmt.Errorf("decline friend request: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return false, domain.ErrNotFound
	}
	return true, nil
}

func (s *FriendshipsStore) Cancel(ctx context.Context, requestID, requesterID string, when time.Time, checkUpdatedAt bool) (bool, error) {
	if checkUpdatedAt {
		const q = `
			UPDATE friendships
			SET status = 'declined', responded_at = $3, updated_at = $3
			WHERE id = $1 AND requester_id = $2 AND status = 'pending' AND updated_at < $3
		`
		ct, err := s.pool.Exec(ctx, q, requestID, requesterID, when)
		if err != nil {
			return false, fmt.Errorf("cancel friend request: %w", err)
		}
		if ct.RowsAffected() == 1 {
			return true, nil
		}
		exists, err := s.pendingForRequester(ctx, requestID, requesterID)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, domain.ErrNotFound
		}
		return false, nil
	}

	const q = `
		UPDATE friendships
		SET status = 'declined', responded_at = $3, updated_at = $3
		WHERE id = $1 AND requester_id = $2 AND status = 'pending'
	`
	ct, err := s.pool.Exec(ctx, q, requestID, requesterID, when)
	if err != nil {
		return false, fmt.Errorf("cancel friend request: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return false, domain.ErrNotFound
	}
	return true, nil
}

func (s *FriendshipsStore) ListOverview(ctx context.Context, userID string) (domain.FriendsOverview, error) {
	friends, err := s.listFriends(ctx, userID)
	if err != nil {
		return domain.FriendsOverview{}, err
	}
	incoming, err := s.listIncoming(ctx, userID)
	if err != nil {
		return domain.FriendsOverview{}, err
	}
	outgoing, err := s.listOutgoing(ctx, userID)
	if err != nil {
		return domain.FriendsOverview{}, err
	}

	return domain.FriendsOverview{
		Friends:  friends,
		Incoming: incoming,
		Outgoing: outgoing,
	}, nil
}

func (s *FriendshipsStore) listFriends(ctx context.Context, userID string) ([]domain.UserSummary, error) {
	const q = `
		SELECT u.id, u.username, u.display_name, u.avatar_path, u.avatar_updated_at, u.updated_at
		FROM friendships f
		JOIN users u ON u.id = CASE
			WHEN f.requester_id = $1 THEN f.addressee_id
			ELSE f.requester_id
		END
		WHERE f.status = 'accepted' AND (f.requester_id = $1 OR f.addressee_id = $1)
		ORDER BY u.username ASC
	`

	rows, err := s.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list friends: %w", err)
	}
	defer rows.Close()

	var out []domain.UserSummary
	for rows.Next() {
		var (
			idUUID        pgtype.UUID
			username      string
			displayName   pgtype.Text
			avatarPath    pgtype.Text
			avatarUpdated pgtype.Timestamptz
			userUpdated   pgtype.Timestamptz
		)
		if err := rows.Scan(&idUUID, &username, &displayName, &avatarPath, &avatarUpdated, &userUpdated); err != nil {
			return nil, fmt.Errorf("scan friend: %w", err)
		}
		out = append(out, domain.UserSummary{
			ID:              uuidOrEmpty(idUUID),
			Username:        username,
			DisplayName:     textOrEmpty(displayName),
			AvatarPath:      textOrEmpty(avatarPath),
			AvatarUpdatedAt: timestamptzPtr(avatarUpdated),
			UpdatedAt:       timestamptzPtr(userUpdated),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list friends: %w", err)
	}
	return out, nil
}

func (s *FriendshipsStore) listIncoming(ctx context.Context, userID string) ([]domain.FriendRequest, error) {
	const q = `
		SELECT f.id, f.created_at, f.updated_at, f.responded_at,
		       u.id, u.username, u.display_name, u.avatar_path, u.avatar_updated_at, u.updated_at
		FROM friendships f
		JOIN users u ON u.id = f.requester_id
		WHERE f.status = 'pending' AND f.addressee_id = $1
		ORDER BY f.created_at DESC
	`

	rows, err := s.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list incoming requests: %w", err)
	}
	defer rows.Close()

	var out []domain.FriendRequest
	for rows.Next() {
		var reqIDUUID pgtype.UUID
		var createdAt time.Time
		var updatedAt time.Time
		var resolvedAt pgtype.Timestamptz
		var fromIDUUID pgtype.UUID
		var fromUsername string
		var displayName pgtype.Text
		var avatarPath pgtype.Text
		var avatarUpdated pgtype.Timestamptz
		var userUpdated pgtype.Timestamptz
		if err := rows.Scan(&reqIDUUID, &createdAt, &updatedAt, &resolvedAt, &fromIDUUID, &fromUsername, &displayName, &avatarPath, &avatarUpdated, &userUpdated); err != nil {
			return nil, fmt.Errorf("scan incoming request: %w", err)
		}
		out = append(out, domain.FriendRequest{
			ID: uuidOrEmpty(reqIDUUID),
			User: domain.UserSummary{
				ID:              uuidOrEmpty(fromIDUUID),
				Username:        fromUsername,
				DisplayName:     textOrEmpty(displayName),
				AvatarPath:      textOrEmpty(avatarPath),
				AvatarUpdatedAt: timestamptzPtr(avatarUpdated),
				UpdatedAt:       timestamptzPtr(userUpdated),
			},
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt,
			ResolvedAt: timestamptzPtr(resolvedAt),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list incoming requests: %w", err)
	}
	return out, nil
}

func (s *FriendshipsStore) listOutgoing(ctx context.Context, userID string) ([]domain.FriendRequest, error) {
	const q = `
		SELECT f.id, f.created_at, f.updated_at, f.responded_at,
		       u.id, u.username, u.display_name, u.avatar_path, u.avatar_updated_at, u.updated_at
		FROM friendships f
		JOIN users u ON u.id = f.addressee_id
		WHERE f.status = 'pending' AND f.requester_id = $1
		ORDER BY f.created_at DESC
	`

	rows, err := s.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list outgoing requests: %w", err)
	}
	defer rows.Close()

	var out []domain.FriendRequest
	for rows.Next() {
		var reqIDUUID pgtype.UUID
		var createdAt time.Time
		var updatedAt time.Time
		var resolvedAt pgtype.Timestamptz
		var toIDUUID pgtype.UUID
		var toUsername string
		var displayName pgtype.Text
		var avatarPath pgtype.Text
		var avatarUpdated pgtype.Timestamptz
		var userUpdated pgtype.Timestamptz
		if err := rows.Scan(&reqIDUUID, &createdAt, &updatedAt, &resolvedAt, &toIDUUID, &toUsername, &displayName, &avatarPath, &avatarUpdated, &userUpdated); err != nil {
			return nil, fmt.Errorf("scan outgoing request: %w", err)
		}
		out = append(out, domain.FriendRequest{
			ID: uuidOrEmpty(reqIDUUID),
			User: domain.UserSummary{
				ID:              uuidOrEmpty(toIDUUID),
				Username:        toUsername,
				DisplayName:     textOrEmpty(displayName),
				AvatarPath:      textOrEmpty(avatarPath),
				AvatarUpdatedAt: timestamptzPtr(avatarUpdated),
				UpdatedAt:       timestamptzPtr(userUpdated),
			},
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt,
			ResolvedAt: timestamptzPtr(resolvedAt),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list outgoing requests: %w", err)
	}
	return out, nil
}

func (s *FriendshipsStore) AreFriends(ctx context.Context, userA, userB string) (bool, error) {
	const q = `
		SELECT 1
		FROM friendships
		WHERE status = 'accepted'
		  AND (
		    (requester_id = $1 AND addressee_id = $2)
		    OR
		    (requester_id = $2 AND addressee_id = $1)
		  )
		LIMIT 1
	`
	var one int
	err := s.pool.QueryRow(ctx, q, userA, userB).Scan(&one)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("are friends: %w", err)
	}
	return true, nil
}

func (s *FriendshipsStore) LatestFriendshipUpdate(ctx context.Context, userID string) (time.Time, error) {
	const q = `
		SELECT COALESCE(MAX(updated_at), TIMESTAMPTZ 'epoch')
		FROM friendships
		WHERE requester_id = $1 OR addressee_id = $1
	`
	var updatedAt time.Time
	if err := s.pool.QueryRow(ctx, q, userID).Scan(&updatedAt); err != nil {
		return time.Time{}, fmt.Errorf("latest friendship update: %w", err)
	}
	return updatedAt, nil
}

func (s *FriendshipsStore) pendingForAddressee(ctx context.Context, requestID, addresseeID string) (bool, error) {
	const q = `
		SELECT 1
		FROM friendships
		WHERE id = $1 AND addressee_id = $2 AND status = 'pending'
		LIMIT 1
	`
	var one int
	err := s.pool.QueryRow(ctx, q, requestID, addresseeID).Scan(&one)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check pending request: %w", err)
	}
	return true, nil
}

func (s *FriendshipsStore) pendingForRequester(ctx context.Context, requestID, requesterID string) (bool, error) {
	const q = `
		SELECT 1
		FROM friendships
		WHERE id = $1 AND requester_id = $2 AND status = 'pending'
		LIMIT 1
	`
	var one int
	err := s.pool.QueryRow(ctx, q, requestID, requesterID).Scan(&one)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check pending request: %w", err)
	}
	return true, nil
}

var _ = pgx.ErrNoRows
