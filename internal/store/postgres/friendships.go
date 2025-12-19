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

func (s *FriendshipsStore) CreateRequest(ctx context.Context, requesterID, addresseeID string) (string, time.Time, error) {
	const q = `
		INSERT INTO friendships (requester_id, addressee_id, status)
		VALUES ($1, $2, 'pending')
		RETURNING id, created_at
	`

	var (
		idUUID    pgtype.UUID
		createdAt time.Time
	)
	err := s.pool.QueryRow(ctx, q, requesterID, addresseeID).Scan(&idUUID, &createdAt)
	if err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) && pgerr.Code == "23505" && pgerr.ConstraintName == "friendships_pair_uq" {
			return "", time.Time{}, domain.ErrFriendshipExists
		}
		return "", time.Time{}, fmt.Errorf("create friend request: %w", err)
	}

	return uuidOrEmpty(idUUID), createdAt, nil
}

func (s *FriendshipsStore) Accept(ctx context.Context, requestID, addresseeID string, when time.Time) error {
	const q = `
		UPDATE friendships
		SET status = 'accepted', responded_at = $3
		WHERE id = $1 AND addressee_id = $2 AND status = 'pending'
	`
	ct, err := s.pool.Exec(ctx, q, requestID, addresseeID, when)
	if err != nil {
		return fmt.Errorf("accept friend request: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *FriendshipsStore) Decline(ctx context.Context, requestID, addresseeID string, when time.Time) error {
	const q = `
		UPDATE friendships
		SET status = 'declined', responded_at = $3
		WHERE id = $1 AND addressee_id = $2 AND status = 'pending'
	`
	ct, err := s.pool.Exec(ctx, q, requestID, addresseeID, when)
	if err != nil {
		return fmt.Errorf("decline friend request: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
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
		SELECT u.id, u.username
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
		var idUUID pgtype.UUID
		var username string
		if err := rows.Scan(&idUUID, &username); err != nil {
			return nil, fmt.Errorf("scan friend: %w", err)
		}
		out = append(out, domain.UserSummary{ID: uuidOrEmpty(idUUID), Username: username})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list friends: %w", err)
	}
	return out, nil
}

func (s *FriendshipsStore) listIncoming(ctx context.Context, userID string) ([]domain.FriendRequest, error) {
	const q = `
		SELECT f.id, f.created_at, u.id, u.username
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
		var fromIDUUID pgtype.UUID
		var fromUsername string
		if err := rows.Scan(&reqIDUUID, &createdAt, &fromIDUUID, &fromUsername); err != nil {
			return nil, fmt.Errorf("scan incoming request: %w", err)
		}
		out = append(out, domain.FriendRequest{
			ID:        uuidOrEmpty(reqIDUUID),
			User:      domain.UserSummary{ID: uuidOrEmpty(fromIDUUID), Username: fromUsername},
			CreatedAt: createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list incoming requests: %w", err)
	}
	return out, nil
}

func (s *FriendshipsStore) listOutgoing(ctx context.Context, userID string) ([]domain.FriendRequest, error) {
	const q = `
		SELECT f.id, f.created_at, u.id, u.username
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
		var toIDUUID pgtype.UUID
		var toUsername string
		if err := rows.Scan(&reqIDUUID, &createdAt, &toIDUUID, &toUsername); err != nil {
			return nil, fmt.Errorf("scan outgoing request: %w", err)
		}
		out = append(out, domain.FriendRequest{
			ID:        uuidOrEmpty(reqIDUUID),
			User:      domain.UserSummary{ID: uuidOrEmpty(toIDUUID), Username: toUsername},
			CreatedAt: createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list outgoing requests: %w", err)
	}
	return out, nil
}

var _ = pgx.ErrNoRows
