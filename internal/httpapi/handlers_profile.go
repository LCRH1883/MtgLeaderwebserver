package httpapi

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"MtgLeaderwebserver/internal/domain"
)

const defaultAvatarURL = "/app/static/skull.svg"

type updateProfileRequest struct {
	DisplayName *string `json:"display_name"`
}

func (a *api) handleUsersMeUpdate(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}
	if a.profileSvc == nil {
		WriteDomainError(w, domain.ErrNotFound)
		return
	}

	var req updateProfileRequest
	if err := decodeJSON(w, r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_json", "invalid json")
		return
	}
	if req.DisplayName == nil {
		WriteDomainError(w, domain.NewValidationError(map[string]string{"display_name": "required"}))
		return
	}

	if err := a.profileSvc.UpdateDisplayName(r.Context(), u.ID, *req.DisplayName); err != nil {
		WriteDomainError(w, err)
		return
	}

	updated, ok := a.currentUserSnapshot(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}
	writeUser(w, http.StatusOK, updated, nil)
}

func (a *api) handleUsersMeAvatar(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}
	if a.profileSvc == nil {
		WriteDomainError(w, domain.ErrNotFound)
		return
	}

	const maxAvatarSize = 8 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarSize)
	if err := r.ParseMultipartForm(maxAvatarSize); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_avatar", "avatar file is too large")
		return
	}

	file, _, err := r.FormFile("avatar")
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_avatar", "avatar file is required")
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_avatar", "avatar must be a valid image file")
		return
	}
	bounds := img.Bounds()
	if bounds.Dx() != 512 || bounds.Dy() != 512 {
		WriteError(w, http.StatusBadRequest, "invalid_avatar", "avatar must be 512x512")
		return
	}

	if err := os.MkdirAll(a.avatarDir, 0o755); err != nil {
		a.logger.Error("create avatar dir failed", "err", err)
		WriteError(w, http.StatusInternalServerError, "internal_error", "failed to store avatar")
		return
	}

	filename := fmt.Sprintf("%s.jpg", u.ID)
	targetPath := filepath.Join(a.avatarDir, filename)
	tmpFile, err := os.CreateTemp(a.avatarDir, "avatar-*")
	if err != nil {
		a.logger.Error("create avatar file failed", "err", err)
		WriteError(w, http.StatusInternalServerError, "internal_error", "failed to store avatar")
		return
	}

	writeErr := func(err error) {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		a.logger.Error("write avatar failed", "err", err)
		WriteError(w, http.StatusInternalServerError, "internal_error", "failed to store avatar")
	}

	dst := image.NewRGBA(image.Rect(0, 0, 512, 512))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), img, bounds.Min, draw.Over)
	if err := jpeg.Encode(tmpFile, dst, &jpeg.Options{Quality: 85}); err != nil {
		writeErr(err)
		return
	}
	if err := tmpFile.Close(); err != nil {
		writeErr(err)
		return
	}
	if err := os.Rename(tmpFile.Name(), targetPath); err != nil {
		writeErr(err)
		return
	}
	if err := os.Chmod(targetPath, 0o644); err != nil {
		a.logger.Error("chmod avatar failed", "err", err)
	}

	updatedAt := time.Now()
	if err := a.profileSvc.UpdateAvatar(r.Context(), u.ID, filename, updatedAt); err != nil {
		_ = os.Remove(targetPath)
		WriteDomainError(w, err)
		return
	}

	updated, ok := a.currentUserSnapshot(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}
	writeUser(w, http.StatusOK, updated, nil)
}

func (a *api) currentUserSnapshot(ctx context.Context) (domain.User, bool) {
	u, ok := CurrentUser(ctx)
	if !ok {
		return domain.User{}, false
	}
	sessID, ok := CurrentSessionID(ctx)
	if !ok || a.authSvc == nil {
		return u, true
	}
	updated, err := a.authSvc.GetUserForSession(ctx, sessID)
	if err != nil {
		return u, true
	}
	return updated, true
}

func avatarURL(u domain.User) string {
	updatedAt := u.AvatarUpdatedAt
	if updatedAt == nil {
		updatedAt = &u.UpdatedAt
	}
	return avatarURLWithUpdated(u.AvatarPath, updatedAt)
}

func avatarURLWithUpdated(path string, updatedAt *time.Time) string {
	if path == "" {
		return defaultAvatarURL
	}
	escaped := url.PathEscape(path)
	if updatedAt == nil {
		return "/app/avatars/" + escaped
	}
	return "/app/avatars/" + escaped + "?v=" + strconv.FormatInt(updatedAt.Unix(), 10)
}
