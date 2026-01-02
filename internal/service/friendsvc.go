package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"MtgLeaderwebserver/internal/domain"
)

type FriendshipsStore interface {
	CreateRequest(ctx context.Context, requesterID, addresseeID string) (string, time.Time, time.Time, error)
	Accept(ctx context.Context, requestID, addresseeID string, when time.Time, checkUpdatedAt bool) (bool, error)
	Decline(ctx context.Context, requestID, addresseeID string, when time.Time, checkUpdatedAt bool) (bool, error)
	Cancel(ctx context.Context, requestID, requesterID string, when time.Time, checkUpdatedAt bool) (bool, error)
	ListOverview(ctx context.Context, userID string) (domain.FriendsOverview, error)
	AreFriends(ctx context.Context, userA, userB string) (bool, error)
	LatestFriendshipUpdate(ctx context.Context, userID string) (time.Time, error)
}

type FriendsService struct {
	Users       UsersStore
	Friendships FriendshipsStore
	Notifier    FriendRequestNotifier
	Now         func() time.Time
}

type FriendRequestActionResult int

const (
	FriendRequestActionApplied FriendRequestActionResult = iota
	FriendRequestActionConflict
)

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
			UpdatedAt: req.UpdatedAt,
		})
	}

	for _, req := range overview.Outgoing {
		out = append(out, domain.FriendConnection{
			User:      req.User,
			Status:    domain.FriendStatusOutgoing,
			RequestID: req.ID,
			CreatedAt: req.CreatedAt,
			UpdatedAt: req.UpdatedAt,
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

	id, createdAt, updatedAt, err := s.Friendships.CreateRequest(ctx, requesterID, target.ID)
	if err != nil {
		return domain.FriendRequest{}, err
	}

	if s.Notifier != nil {
		_ = s.Notifier.NotifyFriendRequest(ctx, FriendRequestNotification{
			RequestID:   id,
			RequesterID: requesterID,
			AddresseeID: target.ID,
		})
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
		UpdatedAt: updatedAt,
	}, nil
}

func (s *FriendsService) Accept(ctx context.Context, addresseeID, requestID string, updatedAt *time.Time) (FriendRequestActionResult, error) {
	when, checkUpdatedAt := s.actionTime(updatedAt)
	applied, err := s.Friendships.Accept(ctx, requestID, addresseeID, when, checkUpdatedAt)
	if err != nil {
		return FriendRequestActionConflict, err
	}
	if !applied {
		return FriendRequestActionConflict, nil
	}
	return FriendRequestActionApplied, nil
}

func (s *FriendsService) Decline(ctx context.Context, addresseeID, requestID string, updatedAt *time.Time) (FriendRequestActionResult, error) {
	when, checkUpdatedAt := s.actionTime(updatedAt)
	applied, err := s.Friendships.Decline(ctx, requestID, addresseeID, when, checkUpdatedAt)
	if err != nil {
		return FriendRequestActionConflict, err
	}
	if !applied {
		return FriendRequestActionConflict, nil
	}
	return FriendRequestActionApplied, nil
}

func (s *FriendsService) Cancel(ctx context.Context, requesterID, requestID string, updatedAt *time.Time) (FriendRequestActionResult, error) {
	when, checkUpdatedAt := s.actionTime(updatedAt)
	applied, err := s.Friendships.Cancel(ctx, requestID, requesterID, when, checkUpdatedAt)
	if err != nil {
		return FriendRequestActionConflict, err
	}
	if !applied {
		return FriendRequestActionConflict, nil
	}
	return FriendRequestActionApplied, nil
}

func (s *FriendsService) AreFriends(ctx context.Context, userA, userB string) (bool, error) {
	return s.Friendships.AreFriends(ctx, userA, userB)
}

func (s *FriendsService) LatestFriendshipUpdate(ctx context.Context, userID string) (time.Time, error) {
	return s.Friendships.LatestFriendshipUpdate(ctx, userID)
}

func (s *FriendsService) actionTime(updatedAt *time.Time) (time.Time, bool) {
	if updatedAt != nil {
		return updatedAt.UTC().Truncate(time.Millisecond), true
	}
	if s.Now == nil {
		s.Now = time.Now
	}
	when := s.Now().UTC()
	return when.Truncate(time.Millisecond), false
}
