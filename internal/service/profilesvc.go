package service

import (
	"context"
	"strings"
	"time"

	"MtgLeaderwebserver/internal/domain"
)

type ProfileStore interface {
	SetDisplayName(ctx context.Context, userID, displayName string) error
	SetAvatar(ctx context.Context, userID, avatarPath string, updatedAt time.Time) error
}

type ProfileService struct {
	Store ProfileStore
}

func (s *ProfileService) UpdateDisplayName(ctx context.Context, userID, displayName string) error {
	displayName = strings.TrimSpace(displayName)
	if displayName != "" {
		if len(displayName) > 48 {
			return domain.NewValidationError(map[string]string{"display_name": "must be 48 characters or less"})
		}
		for _, r := range displayName {
			if r < 32 {
				return domain.NewValidationError(map[string]string{"display_name": "contains invalid characters"})
			}
		}
	}
	return s.Store.SetDisplayName(ctx, userID, displayName)
}

func (s *ProfileService) UpdateAvatar(ctx context.Context, userID, avatarPath string, updatedAt time.Time) error {
	if strings.TrimSpace(avatarPath) == "" {
		return domain.NewValidationError(map[string]string{"avatar": "file is required"})
	}
	return s.Store.SetAvatar(ctx, userID, avatarPath, updatedAt)
}
