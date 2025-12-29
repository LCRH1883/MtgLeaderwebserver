package service

import (
	"context"
	"fmt"

	"MtgLeaderwebserver/internal/auth"
	"MtgLeaderwebserver/internal/domain"
)

type AdminUsersStore interface {
	ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error)
	SearchUsers(ctx context.Context, query string, limit, offset int) ([]domain.User, error)
}

type AdminPasswordsStore interface {
	SetPasswordHash(ctx context.Context, userID, passwordHash string) error
}

type AdminService struct {
	Users     AdminUsersStore
	Passwords AdminPasswordsStore
}

func (s *AdminService) ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error) {
	return s.Users.ListUsers(ctx, limit, offset)
}

func (s *AdminService) SearchUsers(ctx context.Context, query string, limit, offset int) ([]domain.User, error) {
	return s.Users.SearchUsers(ctx, query, limit, offset)
}

func (s *AdminService) ResetUserPassword(ctx context.Context, userID, newPassword string) error {
	if s.Passwords == nil {
		return fmt.Errorf("admin passwords store unavailable")
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := s.Passwords.SetPasswordHash(ctx, userID, hash); err != nil {
		return fmt.Errorf("set password: %w", err)
	}
	return nil
}
