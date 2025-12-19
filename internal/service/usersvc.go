package service

import (
	"context"
	"strings"

	"MtgLeaderwebserver/internal/domain"
)

type UsersSearchStore interface {
	SearchUsers(ctx context.Context, q string, limit int, excludeUserID string) ([]domain.UserSummary, error)
}

type UsersService struct {
	Store UsersSearchStore
}

func (s *UsersService) Search(ctx context.Context, q string, limit int, excludeUserID string) ([]domain.UserSummary, error) {
	q = strings.TrimSpace(q)
	if len(q) < 3 {
		return nil, domain.NewValidationError(map[string]string{"q": "must be at least 3 characters"})
	}
	return s.Store.SearchUsers(ctx, q, limit, excludeUserID)
}
