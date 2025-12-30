package httpapi

import (
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
	"strings"
	"time"

	"MtgLeaderwebserver/internal/domain"
	"MtgLeaderwebserver/internal/service"
)

const defaultAvatarURL = "/app/static/skull.svg"

type updateProfileRequest struct {
	DisplayName *string `json:"display_name"`
	UpdatedAt   string  `json:"updated_at"`
}

func (a *api) handleUsersMeUpdate(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}
	if a.profileSvc == nil {
		WriteError(w, http.StatusServiceUnavailable, "profile_unavailable", "profile unavailable")
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

	updatedAt, err := parseUpdatedAt(req.UpdatedAt)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_updated_at", "updated_at must be RFC3339 UTC with milliseconds")
		return
	}

	updated, result, err := a.profileSvc.UpdateDisplayName(r.Context(), u.ID, *req.DisplayName, updatedAt)
	if err != nil {
		WriteDomainError(w, err)
		return
	}

	switch result {
	case service.ProfileUpdateConflict:
		writeUser(w, http.StatusConflict, updated, nil)
	default:
		writeUser(w, http.StatusOK, updated, nil)
	}
}

func (a *api) handleUsersMeAvatar(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		WriteDomainError(w, domain.ErrUnauthorized)
		return
	}
	if a.profileSvc == nil {
		WriteError(w, http.StatusServiceUnavailable, "profile_unavailable", "profile unavailable")
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

	updatedAt, err := parseUpdatedAt(r.FormValue("updated_at"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_updated_at", "updated_at must be RFC3339 UTC with milliseconds")
		return
	}

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

	filename := fmt.Sprintf("%s-%d.jpg", u.ID, updatedAt.UnixNano())
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

	updated, result, err := a.profileSvc.UpdateAvatar(r.Context(), u.ID, filename, updatedAt)
	if err != nil {
		_ = os.Remove(targetPath)
		WriteDomainError(w, err)
		return
	}
	if result != service.ProfileUpdateApplied {
		_ = os.Remove(targetPath)
		if result == service.ProfileUpdateConflict {
			writeUser(w, http.StatusConflict, updated, nil)
			return
		}
		writeUser(w, http.StatusOK, updated, nil)
		return
	}

	if oldPath := strings.TrimSpace(u.AvatarPath); oldPath != "" && oldPath != filename {
		_ = os.Remove(filepath.Join(a.avatarDir, oldPath))
	}

	writeUser(w, http.StatusOK, updated, nil)
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

func parseUpdatedAt(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("updated_at required")
	}
	if !strings.HasSuffix(raw, "Z") {
		return time.Time{}, fmt.Errorf("updated_at must be utc")
	}
	dot := strings.LastIndex(raw, ".")
	if dot == -1 {
		return time.Time{}, fmt.Errorf("updated_at must include milliseconds")
	}
	fraction := raw[dot+1 : len(raw)-1]
	if len(fraction) != 3 {
		return time.Time{}, fmt.Errorf("updated_at must have 3 digits of milliseconds")
	}
	for _, r := range fraction {
		if r < '0' || r > '9' {
			return time.Time{}, fmt.Errorf("updated_at invalid")
		}
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}
