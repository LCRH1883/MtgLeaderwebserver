package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"MtgLeaderwebserver/internal/domain"
	"MtgLeaderwebserver/internal/notifications"
)

type stubNotificationTokensStore struct {
	upsertFunc func(context.Context, string, string, string, time.Time) (domain.NotificationToken, error)
	deleteFunc func(context.Context, string, string) error
	listFunc   func(context.Context, string) ([]domain.NotificationToken, error)
}

func (s *stubNotificationTokensStore) UpsertToken(ctx context.Context, userID, token, platform string, when time.Time) (domain.NotificationToken, error) {
	if s.upsertFunc != nil {
		return s.upsertFunc(ctx, userID, token, platform, when)
	}
	return domain.NotificationToken{}, errors.New("upsert not stubbed")
}

func (s *stubNotificationTokensStore) DeleteToken(ctx context.Context, userID, token string) error {
	if s.deleteFunc != nil {
		return s.deleteFunc(ctx, userID, token)
	}
	return errors.New("delete not stubbed")
}

func (s *stubNotificationTokensStore) ListTokens(ctx context.Context, userID string) ([]domain.NotificationToken, error) {
	if s.listFunc != nil {
		return s.listFunc(ctx, userID)
	}
	return nil, errors.New("list not stubbed")
}

type stubNotificationUsersStore struct {
	getByIDFunc func(context.Context, string) (domain.User, error)
}

func (s *stubNotificationUsersStore) GetUserByID(ctx context.Context, id string) (domain.User, error) {
	if s.getByIDFunc != nil {
		return s.getByIDFunc(ctx, id)
	}
	return domain.User{}, errors.New("get user not stubbed")
}

type stubPushSender struct {
	sendFunc func(context.Context, string, map[string]string) error
}

func (s *stubPushSender) Send(ctx context.Context, token string, data map[string]string) error {
	if s.sendFunc != nil {
		return s.sendFunc(ctx, token, data)
	}
	return nil
}

func TestNotificationServiceRegisterTokenValidation(t *testing.T) {
	svc := &NotificationService{
		Tokens: &stubNotificationTokensStore{},
	}

	if _, err := svc.RegisterToken(context.Background(), "user-1", "", "android"); err == nil {
		t.Fatalf("expected validation error for empty token")
	}
	if _, err := svc.RegisterToken(context.Background(), "user-1", "token", ""); err == nil {
		t.Fatalf("expected validation error for empty platform")
	}
}

func TestNotificationServiceNotifyFriendRequestDeletesInvalidToken(t *testing.T) {
	deleted := false
	tokens := &stubNotificationTokensStore{
		listFunc: func(_ context.Context, userID string) ([]domain.NotificationToken, error) {
			if userID != "user-2" {
				t.Fatalf("unexpected user id: %s", userID)
			}
			return []domain.NotificationToken{{Token: "token-1"}}, nil
		},
		deleteFunc: func(_ context.Context, userID, token string) error {
			if userID != "user-2" || token != "token-1" {
				t.Fatalf("unexpected delete args: %s %s", userID, token)
			}
			deleted = true
			return nil
		},
	}

	users := &stubNotificationUsersStore{
		getByIDFunc: func(_ context.Context, id string) (domain.User, error) {
			if id != "user-1" {
				t.Fatalf("unexpected requester id: %s", id)
			}
			return domain.User{ID: "user-1", Username: "alice"}, nil
		},
	}

	sender := &stubPushSender{
		sendFunc: func(_ context.Context, token string, data map[string]string) error {
			if token != "token-1" {
				t.Fatalf("unexpected token: %s", token)
			}
			return notifications.ErrInvalidToken
		},
	}

	svc := &NotificationService{
		Tokens: tokens,
		Users:  users,
		Sender: sender,
	}

	err := svc.NotifyFriendRequest(context.Background(), FriendRequestNotification{
		RequestID:   "req-1",
		RequesterID: "user-1",
		AddresseeID: "user-2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Fatalf("expected invalid token to be deleted")
	}
}
