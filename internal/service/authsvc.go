package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"MtgLeaderwebserver/internal/auth"
	"MtgLeaderwebserver/internal/domain"
)

type UsersStore interface {
	CreateUser(ctx context.Context, email, username, passwordHash string) (domain.User, error)
	GetUserByID(ctx context.Context, id string) (domain.User, error)
	GetUserByLogin(ctx context.Context, login string) (domain.UserWithPassword, error)
	SetLastLogin(ctx context.Context, userID string, when time.Time) error
}

type SessionsStore interface {
	CreateSession(ctx context.Context, userID string, expiresAt time.Time, ip, userAgent string) (string, error)
	GetSession(ctx context.Context, sessionID string) (domain.Session, error)
	RevokeSession(ctx context.Context, sessionID string, when time.Time) error
}

type AuthService struct {
	Users      UsersStore
	Sessions   SessionsStore
	SessionTTL time.Duration
	Now        func() time.Time
}

func (s *AuthService) Register(ctx context.Context, email, username, password, ip, userAgent string) (domain.User, string, error) {
	if s.Now == nil {
		s.Now = time.Now
	}

	email = strings.TrimSpace(strings.ToLower(email))
	username = strings.TrimSpace(username)

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return domain.User{}, "", err
	}

	u, err := s.Users.CreateUser(ctx, email, username, passwordHash)
	if err != nil {
		return domain.User{}, "", err
	}

	sessID, err := s.Sessions.CreateSession(ctx, u.ID, s.Now().Add(s.SessionTTL), ip, userAgent)
	if err != nil {
		return domain.User{}, "", err
	}

	return u, sessID, nil
}

func (s *AuthService) Login(ctx context.Context, login, password, ip, userAgent string) (domain.User, string, error) {
	if s.Now == nil {
		s.Now = time.Now
	}

	login = strings.TrimSpace(login)

	u, err := s.Users.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.User{}, "", domain.ErrInvalidCredentials
		}
		return domain.User{}, "", err
	}
	if u.Status == domain.UserStatusDisabled {
		return domain.User{}, "", domain.ErrUserDisabled
	}

	ok, err := auth.VerifyPassword(u.PasswordHash, password)
	if err != nil {
		return domain.User{}, "", err
	}
	if !ok {
		return domain.User{}, "", domain.ErrInvalidCredentials
	}

	sessID, err := s.Sessions.CreateSession(ctx, u.ID, s.Now().Add(s.SessionTTL), ip, userAgent)
	if err != nil {
		return domain.User{}, "", err
	}

	_ = s.Users.SetLastLogin(ctx, u.ID, s.Now())

	return u.User, sessID, nil
}

func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	if s.Now == nil {
		s.Now = time.Now
	}

	return s.Sessions.RevokeSession(ctx, sessionID, s.Now())
}

func (s *AuthService) GetUserForSession(ctx context.Context, sessionID string) (domain.User, error) {
	sess, err := s.Sessions.GetSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.User{}, domain.ErrUnauthorized
		}
		return domain.User{}, err
	}

	u, err := s.Users.GetUserByID(ctx, sess.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.User{}, domain.ErrUnauthorized
		}
		return domain.User{}, err
	}
	if u.Status == domain.UserStatusDisabled {
		return domain.User{}, domain.ErrForbidden
	}

	return u, nil
}
