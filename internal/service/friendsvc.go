package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"MtgLeaderwebserver/internal/domain"
)

type FriendshipsStore interface {
	CreateRequest(ctx context.Context, requesterID, addresseeID string) (string, time.Time, error)
	Accept(ctx context.Context, requestID, addresseeID string, when time.Time) error
	Decline(ctx context.Context, requestID, addresseeID string, when time.Time) error
	Cancel(ctx context.Context, requestID, requesterID string) error
	ListOverview(ctx context.Context, userID string) (domain.FriendsOverview, error)
	AreFriends(ctx context.Context, userA, userB string) (bool, error)
}

type FriendsService struct {
	Users       UsersStore
	Friendships FriendshipsStore
	Now         func() time.Time
}

func (s *FriendsService) ListOverview(ctx context.Context, userID string) (domain.FriendsOverview, error) {
	return s.Friendships.ListOverview(ctx, userID)
}

func (s *FriendsService) ListConnections(ctx context.Context, userID string) ([]domain.FriendConnection, error) {
	overview, err := s.Friendships.ListOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	total := len(overview.Friends) + len(overview.Incoming) + len(overview.Outgoing)
	out := make([]domain.FriendConnection, 0, total)

	for _, friend := range overview.Friends {
		out = append(out, domain.FriendConnection{
			User:   friend,
			Status: domain.FriendStatusAccepted,
		})
	}

	for _, req := range overview.Incoming {
		out = append(out, domain.FriendConnection{
			User:      req.User,
			Status:    domain.FriendStatusIncoming,
			RequestID: req.ID,
			CreatedAt: req.CreatedAt,
		})
	}

	for _, req := range overview.Outgoing {
		out = append(out, domain.FriendConnection{
			User:      req.User,
			Status:    domain.FriendStatusOutgoing,
			RequestID: req.ID,
			CreatedAt: req.CreatedAt,
		})
	}

	return out, nil
}

func (s *FriendsService) CreateRequest(ctx context.Context, requesterID, addresseeUsername string) (domain.FriendRequest, error) {
	if s.Now == nil {
		s.Now = time.Now
	}

	addresseeUsername = strings.TrimSpace(addresseeUsername)
	if addresseeUsername == "" {
		return domain.FriendRequest{}, domain.NewValidationError(map[string]string{"username": "required"})
	}

	target, err := s.Users.GetUserByLogin(ctx, addresseeUsername)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.FriendRequest{}, domain.ErrNotFound
		}
		return domain.FriendRequest{}, err
	}
	if target.ID == requesterID {
		return domain.FriendRequest{}, domain.NewValidationError(map[string]string{"username": "cannot friend yourself"})
	}
	if target.Status == domain.UserStatusDisabled {
		return domain.FriendRequest{}, domain.ErrForbidden
	}

	id, createdAt, err := s.Friendships.CreateRequest(ctx, requesterID, target.ID)
	if err != nil {
		return domain.FriendRequest{}, err
	}

	return domain.FriendRequest{
		ID: id,
		User: domain.UserSummary{
			ID:              target.ID,
			Username:        target.Username,
			DisplayName:     target.DisplayName,
			AvatarPath:      target.AvatarPath,
			AvatarUpdatedAt: target.AvatarUpdatedAt,
		},
		CreatedAt: createdAt,
	}, nil
}

func (s *FriendsService) Accept(ctx context.Context, addresseeID, requestID string) error {
	if s.Now == nil {
		s.Now = time.Now
	}
	return s.Friendships.Accept(ctx, requestID, addresseeID, s.Now())
}

func (s *FriendsService) Decline(ctx context.Context, addresseeID, requestID string) error {
	if s.Now == nil {
		s.Now = time.Now
	}
	return s.Friendships.Decline(ctx, requestID, addresseeID, s.Now())
}

func (s *FriendsService) Cancel(ctx context.Context, requesterID, requestID string) error {
	return s.Friendships.Cancel(ctx, requestID, requesterID)
}

func (s *FriendsService) AreFriends(ctx context.Context, userA, userB string) (bool, error) {
	return s.Friendships.AreFriends(ctx, userA, userB)
}
