package service

import (
	"context"
	"strings"
	"time"

	"MtgLeaderwebserver/internal/domain"
)

type ProfileStore interface {
	UpdateDisplayName(ctx context.Context, userID, displayName string, updatedAt time.Time) (domain.User, bool, error)
	UpdateAvatar(ctx context.Context, userID, avatarPath string, updatedAt time.Time) (domain.User, bool, error)
}

type ProfileService struct {
	Store ProfileStore
}

type ProfileUpdateResult int

const (
	ProfileUpdateApplied ProfileUpdateResult = iota
	ProfileUpdateNoop
	ProfileUpdateConflict
)

func (s *ProfileService) UpdateDisplayName(ctx context.Context, userID, displayName string, updatedAt time.Time) (domain.User, ProfileUpdateResult, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName != "" {
		if len(displayName) > 48 {
			return domain.User{}, ProfileUpdateConflict, domain.NewValidationError(map[string]string{"display_name": "must be 48 characters or less"})
		}
		for _, r := range displayName {
			if r < 32 {
				return domain.User{}, ProfileUpdateConflict, domain.NewValidationError(map[string]string{"display_name": "contains invalid characters"})
			}
		}
	}
	u, applied, err := s.Store.UpdateDisplayName(ctx, userID, displayName, updatedAt)
	if err != nil {
		return domain.User{}, ProfileUpdateConflict, err
	}
	if applied {
		return u, ProfileUpdateApplied, nil
	}
	if u.UpdatedAt.Equal(updatedAt) && u.DisplayName == displayName {
		return u, ProfileUpdateNoop, nil
	}
	return u, ProfileUpdateConflict, nil
}

func (s *ProfileService) UpdateAvatar(ctx context.Context, userID, avatarPath string, updatedAt time.Time) (domain.User, ProfileUpdateResult, error) {
	if strings.TrimSpace(avatarPath) == "" {
		return domain.User{}, ProfileUpdateConflict, domain.NewValidationError(map[string]string{"avatar": "file is required"})
	}
	u, applied, err := s.Store.UpdateAvatar(ctx, userID, avatarPath, updatedAt)
	if err != nil {
		return domain.User{}, ProfileUpdateConflict, err
	}
	if applied {
		return u, ProfileUpdateApplied, nil
	}
	if u.UpdatedAt.Equal(updatedAt) && u.AvatarPath == avatarPath {
		return u, ProfileUpdateNoop, nil
	}
	return u, ProfileUpdateConflict, nil
}
