package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"MtgLeaderwebserver/internal/auth"
	"MtgLeaderwebserver/internal/domain"
	"MtgLeaderwebserver/internal/service"
)

type stubUsersStoreDelete struct {
	t *testing.T

	deleteUserFunc func(context.Context, string) error
}

func (s *stubUsersStoreDelete) CreateUser(context.Context, string, string, string) (domain.User, error) {
	s.t.Fatalf("CreateUser called unexpectedly")
	return domain.User{}, context.Canceled
}

func (s *stubUsersStoreDelete) GetUserByID(context.Context, string) (domain.User, error) {
	s.t.Fatalf("GetUserByID called unexpectedly")
	return domain.User{}, context.Canceled
}

func (s *stubUsersStoreDelete) GetUserByLogin(context.Context, string) (domain.UserWithPassword, error) {
	s.t.Fatalf("GetUserByLogin called unexpectedly")
	return domain.UserWithPassword{}, context.Canceled
}

func (s *stubUsersStoreDelete) GetUserByEmail(context.Context, string) (domain.UserWithPassword, error) {
	s.t.Fatalf("GetUserByEmail called unexpectedly")
	return domain.UserWithPassword{}, context.Canceled
}

func (s *stubUsersStoreDelete) GetUserByExternalAccount(context.Context, string, string) (domain.User, domain.ExternalAccount, error) {
	s.t.Fatalf("GetUserByExternalAccount called unexpectedly")
	return domain.User{}, domain.ExternalAccount{}, context.Canceled
}

func (s *stubUsersStoreDelete) CreateUserWithExternalAccount(context.Context, string, string, string, string, string) (domain.User, domain.ExternalAccount, error) {
	s.t.Fatalf("CreateUserWithExternalAccount called unexpectedly")
	return domain.User{}, domain.ExternalAccount{}, context.Canceled
}

func (s *stubUsersStoreDelete) LinkExternalAccount(context.Context, string, string, string, string) (domain.ExternalAccount, error) {
	s.t.Fatalf("LinkExternalAccount called unexpectedly")
	return domain.ExternalAccount{}, context.Canceled
}

func (s *stubUsersStoreDelete) SetLastLogin(context.Context, string, time.Time) error {
	s.t.Fatalf("SetLastLogin called unexpectedly")
	return context.Canceled
}

func (s *stubUsersStoreDelete) SetPasswordHash(context.Context, string, string) error {
	s.t.Fatalf("SetPasswordHash called unexpectedly")
	return context.Canceled
}

func (s *stubUsersStoreDelete) DeleteUser(ctx context.Context, userID string) error {
	if s.deleteUserFunc != nil {
		return s.deleteUserFunc(ctx, userID)
	}
	s.t.Fatalf("DeleteUser called unexpectedly")
	return context.Canceled
}

func TestUsersMeDeleteRejectsMissingConfirm(t *testing.T) {
	api := &api{
		authSvc: &service.AuthService{
			Users: &stubUsersStoreDelete{
				t: t,
				deleteUserFunc: func(_ context.Context, userID string) error {
					if userID != "user-1" {
						t.Fatalf("unexpected user id: %s", userID)
					}
					return nil
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/users/me/delete", nil)
	req = req.WithContext(context.WithValue(req.Context(), authUserKey, domain.User{ID: "user-1"}))
	rr := httptest.NewRecorder()

	api.handleUsersMeDelete(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
}

func TestUsersMeDeleteRemovesAvatarAndClearsCookie(t *testing.T) {
	tmpDir := t.TempDir()
	avatarName := "avatar.jpg"
	avatarPath := filepath.Join(tmpDir, avatarName)
	if err := os.WriteFile(avatarPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("create avatar file: %v", err)
	}

	api := &api{
		avatarDir: tmpDir,
		authSvc: &service.AuthService{
			Users: &stubUsersStoreDelete{
				t: t,
				deleteUserFunc: func(_ context.Context, userID string) error {
					if userID != "user-1" {
						t.Fatalf("unexpected user id: %s", userID)
					}
					return nil
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/users/me/delete", nil)
	req = req.WithContext(context.WithValue(req.Context(), authUserKey, domain.User{ID: "user-1", AvatarPath: avatarName}))
	rr := httptest.NewRecorder()

	api.handleUsersMeDelete(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	if _, err := os.Stat(avatarPath); !os.IsNotExist(err) {
		t.Fatalf("expected avatar file to be removed")
	}

	found := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == auth.SessionCookieName {
			found = true
			if c.Value != "" {
				t.Fatalf("expected cleared cookie value")
			}
			if c.Path != "/" {
				t.Fatalf("unexpected cookie path: %s", c.Path)
			}
		}
	}
	if !found {
		t.Fatalf("expected a Set-Cookie header for %s", auth.SessionCookieName)
	}
}

func TestUsersMeDeleteAcceptsQueryConfirm(t *testing.T) {
	api := &api{
		authSvc: &service.AuthService{
			Users: &stubUsersStoreDelete{
				t: t,
				deleteUserFunc: func(_ context.Context, userID string) error {
					if userID != "user-1" {
						t.Fatalf("unexpected user id: %s", userID)
					}
					return nil
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodDelete, "/v1/users/me", nil)
	req = req.WithContext(context.WithValue(req.Context(), authUserKey, domain.User{ID: "user-1"}))
	rr := httptest.NewRecorder()

	api.handleUsersMeDelete(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
}

func TestUsersMeDeleteNotFoundStillClearsCookie(t *testing.T) {
	api := &api{
		authSvc: &service.AuthService{
			Users: &stubUsersStoreDelete{
				t: t,
				deleteUserFunc: func(context.Context, string) error {
					return domain.ErrNotFound
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/users/me/delete", nil)
	req = req.WithContext(context.WithValue(req.Context(), authUserKey, domain.User{ID: "user-1"}))
	rr := httptest.NewRecorder()

	api.handleUsersMeDelete(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	found := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == auth.SessionCookieName {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected cookie to be cleared")
	}
}
