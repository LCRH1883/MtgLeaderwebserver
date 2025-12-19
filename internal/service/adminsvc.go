package service

import (
	"context"

	"MtgLeaderwebserver/internal/domain"
)

type AdminUsersStore interface {
	ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error)
}

type AdminService struct {
	Users AdminUsersStore
}

func (s *AdminService) ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error) {
	return s.Users.ListUsers(ctx, limit, offset)
}
