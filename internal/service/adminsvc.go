package service

import (
	"context"

	"MtgLeaderwebserver/internal/domain"
)

type AdminUsersStore interface {
	ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error)
	SearchUsers(ctx context.Context, query string, limit, offset int) ([]domain.User, error)
	GetUserByID(ctx context.Context, id string) (domain.User, error)
	SetUserEmail(ctx context.Context, userID, email string) error
}

type AdminService struct {
	Users AdminUsersStore
}

func (s *AdminService) ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error) {
	return s.Users.ListUsers(ctx, limit, offset)
}

func (s *AdminService) SearchUsers(ctx context.Context, query string, limit, offset int) ([]domain.User, error) {
	return s.Users.SearchUsers(ctx, query, limit, offset)
}

func (s *AdminService) GetUserByID(ctx context.Context, id string) (domain.User, error) {
	return s.Users.GetUserByID(ctx, id)
}

func (s *AdminService) UpdateUserEmail(ctx context.Context, userID, email string) error {
	return s.Users.SetUserEmail(ctx, userID, email)
}
