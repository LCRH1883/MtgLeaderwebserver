package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"MtgLeaderwebserver/internal/domain"
	"MtgLeaderwebserver/internal/notifications"
)

type NotificationTokensStore interface {
	UpsertToken(ctx context.Context, userID, token, platform string, when time.Time) (domain.NotificationToken, error)
	DeleteToken(ctx context.Context, userID, token string) error
	ListTokens(ctx context.Context, userID string) ([]domain.NotificationToken, error)
}

type NotificationUsersStore interface {
	GetUserByID(ctx context.Context, id string) (domain.User, error)
}

type PushSender interface {
	Send(ctx context.Context, token string, msg notifications.Message) error
}

type FriendRequestNotification struct {
	RequestID   string
	RequesterID string
	AddresseeID string
}

type FriendRequestNotifier interface {
	NotifyFriendRequest(ctx context.Context, notification FriendRequestNotification) error
}

type NotificationService struct {
	Tokens NotificationTokensStore
	Users  NotificationUsersStore
	Sender PushSender
	Logger *slog.Logger
	Now    func() time.Time
}

func (s *NotificationService) RegisterToken(ctx context.Context, userID, token, platform string) (domain.NotificationToken, error) {
	if s.Tokens == nil {
		return domain.NotificationToken{}, errors.New("notifications unavailable")
	}
	token = strings.TrimSpace(token)
	platform = strings.TrimSpace(strings.ToLower(platform))
	if token == "" || platform == "" {
		return domain.NotificationToken{}, domain.NewValidationError(map[string]string{"token": "required", "platform": "required"})
	}
	switch platform {
	case "android", "ios":
	default:
		return domain.NotificationToken{}, domain.NewValidationError(map[string]string{"platform": "must be ios or android"})
	}
	if s.Now == nil {
		s.Now = time.Now
	}
	when := s.Now().UTC().Truncate(time.Millisecond)
	return s.Tokens.UpsertToken(ctx, userID, token, platform, when)
}

func (s *NotificationService) DeleteToken(ctx context.Context, userID, token string) error {
	if s.Tokens == nil {
		return errors.New("notifications unavailable")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return domain.NewValidationError(map[string]string{"token": "required"})
	}
	return s.Tokens.DeleteToken(ctx, userID, token)
}

func (s *NotificationService) NotifyFriendRequest(ctx context.Context, notification FriendRequestNotification) error {
	if s.Tokens == nil || s.Sender == nil || s.Users == nil {
		return nil
	}
	logger := s.Logger
	if logger == nil {
		logger = slog.Default()
	}

	tokens, err := s.Tokens.ListTokens(ctx, notification.AddresseeID)
	if err != nil {
		logger.Error("notifications: list tokens failed", "err", err, "user_id", notification.AddresseeID)
		return err
	}
	if len(tokens) == 0 {
		return nil
	}

	requester, err := s.Users.GetUserByID(ctx, notification.RequesterID)
	if err != nil {
		logger.Error("notifications: requester lookup failed", "err", err, "user_id", notification.RequesterID)
		return err
	}

	display := strings.TrimSpace(requester.DisplayName)
	if display == "" {
		display = requester.Username
	}
	payload := map[string]string{
		"type":         "friend_request",
		"display_name": display,
		"username":     requester.Username,
		"request_id":   notification.RequestID,
	}

	title := "Friend request"
	body := "You received a friend request."
	if display != "" {
		body = display + " sent you a friend request."
	}
	dataOnlyMsg := notifications.Message{
		Data: payload,
	}
	iosAlertMsg := notifications.Message{
		Data: payload,
		Notification: &notifications.Notification{
			Title: title,
			Body:  body,
		},
	}

	for _, token := range tokens {
		msg := dataOnlyMsg
		if strings.TrimSpace(strings.ToLower(token.Platform)) == "ios" {
			msg = iosAlertMsg
		}
		if err := s.Sender.Send(ctx, token.Token, msg); err != nil {
			if errors.Is(err, notifications.ErrInvalidToken) {
				if delErr := s.Tokens.DeleteToken(ctx, notification.AddresseeID, token.Token); delErr != nil {
					logger.Error("notifications: delete invalid token failed", "err", delErr, "user_id", notification.AddresseeID)
				}
				continue
			}
			logger.Error("notifications: send failed", "err", err, "user_id", notification.AddresseeID)
		}
	}

	return nil
}
