package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"MtgLeaderwebserver/internal/auth"
	"MtgLeaderwebserver/internal/domain"
)

type PasswordResetStore interface {
	CreateResetToken(ctx context.Context, token domain.PasswordResetToken) error
	GetResetTokenByHash(ctx context.Context, tokenHash string) (domain.PasswordResetToken, error)
	MarkResetTokenUsed(ctx context.Context, tokenHash string, when time.Time) error
}

type ResetUsersStore interface {
	SetPasswordHash(ctx context.Context, userID, passwordHash string) error
}

type PasswordResetService struct {
	Store    PasswordResetStore
	Users    ResetUsersStore
	TokenTTL time.Duration
	Now      func() time.Time
}

func (s *PasswordResetService) CreateResetToken(ctx context.Context, userID, sentToEmail, createdBy string) (string, error) {
	if s.Store == nil {
		return "", fmt.Errorf("reset store unavailable")
	}
	if userID == "" || sentToEmail == "" {
		return "", fmt.Errorf("user id and email are required")
	}
	if s.Now == nil {
		s.Now = time.Now
	}
	if s.TokenTTL == 0 {
		s.TokenTTL = 2 * time.Hour
	}

	raw, tokenHash, err := newResetToken()
	if err != nil {
		return "", err
	}

	now := s.Now()
	token := domain.PasswordResetToken{
		UserID:      userID,
		TokenHash:   tokenHash,
		SentToEmail: sentToEmail,
		CreatedBy:   createdBy,
		CreatedAt:   now,
		ExpiresAt:   now.Add(s.TokenTTL),
	}
	if err := s.Store.CreateResetToken(ctx, token); err != nil {
		return "", err
	}
	return raw, nil
}

func (s *PasswordResetService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	if s.Store == nil || s.Users == nil {
		return fmt.Errorf("reset service unavailable")
	}
	if s.Now == nil {
		s.Now = time.Now
	}

	tokenHash := hashResetToken(rawToken)
	token, err := s.Store.GetResetTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrResetTokenInvalid
		}
		return err
	}
	if token.UsedAt != nil {
		return domain.ErrResetTokenInvalid
	}
	if token.ExpiresAt.Before(s.Now()) {
		return domain.ErrResetTokenExpired
	}

	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := s.Users.SetPasswordHash(ctx, token.UserID, hash); err != nil {
		return err
	}
	if err := s.Store.MarkResetTokenUsed(ctx, tokenHash, s.Now()); err != nil {
		return err
	}
	return nil
}

func newResetToken() (string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("read token: %w", err)
	}
	raw := base64.RawURLEncoding.EncodeToString(buf)
	return raw, hashResetToken(raw), nil
}

func hashResetToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
