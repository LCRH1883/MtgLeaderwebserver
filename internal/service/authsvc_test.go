package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"MtgLeaderwebserver/internal/auth"
	"MtgLeaderwebserver/internal/domain"
)

type stubUsersStore struct {
	t *testing.T

	createUserFunc             func(context.Context, string, string, string) (domain.User, error)
	getUserByIDFunc            func(context.Context, string) (domain.User, error)
	getUserByLoginFunc         func(context.Context, string) (domain.UserWithPassword, error)
	getUserByEmailFunc         func(context.Context, string) (domain.UserWithPassword, error)
	getUserByExternalFunc      func(context.Context, string, string) (domain.User, domain.ExternalAccount, error)
	createUserWithExternalFunc func(context.Context, string, string, string, string, string) (domain.User, domain.ExternalAccount, error)
	linkExternalAccountFunc    func(context.Context, string, string, string, string) (domain.ExternalAccount, error)
	setLastLoginFunc           func(context.Context, string, time.Time) error
	setPasswordHashFunc        func(context.Context, string, string) error
	deleteUserFunc             func(context.Context, string) error
}

func (s *stubUsersStore) CreateUser(ctx context.Context, email, username, passwordHash string) (domain.User, error) {
	if s.createUserFunc != nil {
		return s.createUserFunc(ctx, email, username, passwordHash)
	}
	s.t.Fatalf("CreateUser called unexpectedly")
	return domain.User{}, errors.New("unexpected call")
}

func (s *stubUsersStore) GetUserByID(ctx context.Context, id string) (domain.User, error) {
	if s.getUserByIDFunc != nil {
		return s.getUserByIDFunc(ctx, id)
	}
	s.t.Fatalf("GetUserByID called unexpectedly")
	return domain.User{}, errors.New("unexpected call")
}

func (s *stubUsersStore) GetUserByLogin(ctx context.Context, login string) (domain.UserWithPassword, error) {
	if s.getUserByLoginFunc != nil {
		return s.getUserByLoginFunc(ctx, login)
	}
	s.t.Fatalf("GetUserByLogin called unexpectedly")
	return domain.UserWithPassword{}, errors.New("unexpected call")
}

func (s *stubUsersStore) GetUserByEmail(ctx context.Context, email string) (domain.UserWithPassword, error) {
	if s.getUserByEmailFunc != nil {
		return s.getUserByEmailFunc(ctx, email)
	}
	s.t.Fatalf("GetUserByEmail called unexpectedly")
	return domain.UserWithPassword{}, errors.New("unexpected call")
}

func (s *stubUsersStore) GetUserByExternalAccount(ctx context.Context, provider, providerID string) (domain.User, domain.ExternalAccount, error) {
	if s.getUserByExternalFunc != nil {
		return s.getUserByExternalFunc(ctx, provider, providerID)
	}
	s.t.Fatalf("GetUserByExternalAccount called unexpectedly")
	return domain.User{}, domain.ExternalAccount{}, errors.New("unexpected call")
}

func (s *stubUsersStore) CreateUserWithExternalAccount(ctx context.Context, provider, providerID, email, username, passwordHash string) (domain.User, domain.ExternalAccount, error) {
	if s.createUserWithExternalFunc != nil {
		return s.createUserWithExternalFunc(ctx, provider, providerID, email, username, passwordHash)
	}
	s.t.Fatalf("CreateUserWithExternalAccount called unexpectedly")
	return domain.User{}, domain.ExternalAccount{}, errors.New("unexpected call")
}

func (s *stubUsersStore) LinkExternalAccount(ctx context.Context, userID, provider, providerID, email string) (domain.ExternalAccount, error) {
	if s.linkExternalAccountFunc != nil {
		return s.linkExternalAccountFunc(ctx, userID, provider, providerID, email)
	}
	s.t.Fatalf("LinkExternalAccount called unexpectedly")
	return domain.ExternalAccount{}, errors.New("unexpected call")
}

func (s *stubUsersStore) SetLastLogin(ctx context.Context, userID string, when time.Time) error {
	if s.setLastLoginFunc != nil {
		return s.setLastLoginFunc(ctx, userID, when)
	}
	s.t.Fatalf("SetLastLogin called unexpectedly")
	return errors.New("unexpected call")
}

func (s *stubUsersStore) SetPasswordHash(ctx context.Context, userID, passwordHash string) error {
	if s.setPasswordHashFunc != nil {
		return s.setPasswordHashFunc(ctx, userID, passwordHash)
	}
	s.t.Fatalf("SetPasswordHash called unexpectedly")
	return errors.New("unexpected call")
}

func (s *stubUsersStore) DeleteUser(ctx context.Context, userID string) error {
	if s.deleteUserFunc != nil {
		return s.deleteUserFunc(ctx, userID)
	}
	s.t.Fatalf("DeleteUser called unexpectedly")
	return errors.New("unexpected call")
}

type stubSessionsStore struct {
	t *testing.T

	createSessionFunc func(context.Context, string, time.Time, string, string) (string, error)
	getSessionFunc    func(context.Context, string) (domain.Session, error)
	revokeSessionFunc func(context.Context, string, time.Time) error
}

func (s *stubSessionsStore) CreateSession(ctx context.Context, userID string, expiresAt time.Time, ip, userAgent string) (string, error) {
	if s.createSessionFunc != nil {
		return s.createSessionFunc(ctx, userID, expiresAt, ip, userAgent)
	}
	s.t.Fatalf("CreateSession called unexpectedly")
	return "", errors.New("unexpected call")
}

func (s *stubSessionsStore) GetSession(ctx context.Context, sessionID string) (domain.Session, error) {
	if s.getSessionFunc != nil {
		return s.getSessionFunc(ctx, sessionID)
	}
	s.t.Fatalf("GetSession called unexpectedly")
	return domain.Session{}, errors.New("unexpected call")
}

func (s *stubSessionsStore) RevokeSession(ctx context.Context, sessionID string, when time.Time) error {
	if s.revokeSessionFunc != nil {
		return s.revokeSessionFunc(ctx, sessionID, when)
	}
	s.t.Fatalf("RevokeSession called unexpectedly")
	return errors.New("unexpected call")
}

func TestAuthServiceLoginWithGoogleExistingAccount(t *testing.T) {
	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)

	users := &stubUsersStore{
		t: t,
		getUserByExternalFunc: func(_ context.Context, provider, providerID string) (domain.User, domain.ExternalAccount, error) {
			if provider != "google" || providerID != "sub-123" {
				t.Fatalf("unexpected provider lookup: %s %s", provider, providerID)
			}
			return domain.User{ID: "user-1", Email: "player@example.com", Username: "player"}, domain.ExternalAccount{}, nil
		},
		setLastLoginFunc: func(_ context.Context, userID string, when time.Time) error {
			if userID != "user-1" {
				t.Fatalf("unexpected user id: %s", userID)
			}
			if !when.Equal(now) {
				t.Fatalf("unexpected last login time: %s", when)
			}
			return nil
		},
	}
	sessions := &stubSessionsStore{
		t: t,
		createSessionFunc: func(_ context.Context, userID string, expiresAt time.Time, ip, userAgent string) (string, error) {
			if userID != "user-1" {
				t.Fatalf("unexpected user id: %s", userID)
			}
			if !expiresAt.Equal(now.Add(24 * time.Hour)) {
				t.Fatalf("unexpected expiry: %s", expiresAt)
			}
			if ip != "1.2.3.4" || userAgent != "unit-test" {
				t.Fatalf("unexpected client info")
			}
			return "sess-1", nil
		},
	}

	svc := &AuthService{
		Users:             users,
		Sessions:          sessions,
		SessionTTL:        24 * time.Hour,
		Now:               func() time.Time { return now },
		GoogleWebClientID: "google-client",
		VerifyGoogleIDToken: func(_ context.Context, token, aud string) (*auth.ExternalTokenClaims, error) {
			if token != "token-123" || aud != "google-client" {
				t.Fatalf("unexpected token/aud")
			}
			return &auth.ExternalTokenClaims{Subject: "sub-123", Email: "Player@Example.com"}, nil
		},
	}

	user, sessID, err := svc.LoginWithGoogle(context.Background(), "token-123", "1.2.3.4", "unit-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "user-1" || sessID != "sess-1" {
		t.Fatalf("unexpected login result: %+v %s", user, sessID)
	}
}

func TestAuthServiceLoginWithGoogleCreatesUser(t *testing.T) {
	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)

	users := &stubUsersStore{
		t: t,
		getUserByExternalFunc: func(_ context.Context, _, _ string) (domain.User, domain.ExternalAccount, error) {
			return domain.User{}, domain.ExternalAccount{}, domain.ErrNotFound
		},
		getUserByEmailFunc: func(_ context.Context, email string) (domain.UserWithPassword, error) {
			if email != "player@example.com" {
				t.Fatalf("unexpected email lookup: %s", email)
			}
			return domain.UserWithPassword{}, domain.ErrNotFound
		},
		createUserWithExternalFunc: func(_ context.Context, provider, providerID, email, username, passwordHash string) (domain.User, domain.ExternalAccount, error) {
			if provider != "google" || providerID != "sub-456" || email != "player@example.com" {
				t.Fatalf("unexpected create args")
			}
			if passwordHash == "" || username == "" || len(username) > 24 {
				t.Fatalf("unexpected username or password hash")
			}
			return domain.User{ID: "user-2", Email: email, Username: username}, domain.ExternalAccount{}, nil
		},
		setLastLoginFunc: func(_ context.Context, userID string, when time.Time) error {
			if userID != "user-2" {
				t.Fatalf("unexpected user id: %s", userID)
			}
			if !when.Equal(now) {
				t.Fatalf("unexpected last login time: %s", when)
			}
			return nil
		},
	}
	sessions := &stubSessionsStore{
		t: t,
		createSessionFunc: func(_ context.Context, userID string, expiresAt time.Time, ip, userAgent string) (string, error) {
			if userID != "user-2" {
				t.Fatalf("unexpected user id: %s", userID)
			}
			return "sess-2", nil
		},
	}

	svc := &AuthService{
		Users:             users,
		Sessions:          sessions,
		SessionTTL:        24 * time.Hour,
		Now:               func() time.Time { return now },
		GoogleWebClientID: "google-client",
		VerifyGoogleIDToken: func(_ context.Context, token, aud string) (*auth.ExternalTokenClaims, error) {
			return &auth.ExternalTokenClaims{Subject: "sub-456", Email: "player@example.com"}, nil
		},
	}

	user, sessID, err := svc.LoginWithGoogle(context.Background(), "token-456", "1.2.3.4", "unit-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "user-2" || sessID != "sess-2" {
		t.Fatalf("unexpected login result: %+v %s", user, sessID)
	}
}

func TestAuthServiceLoginWithGoogleInvalidToken(t *testing.T) {
	svc := &AuthService{
		Users:             &stubUsersStore{t: t},
		Sessions:          &stubSessionsStore{t: t},
		SessionTTL:        time.Hour,
		GoogleWebClientID: "google-client",
		VerifyGoogleIDToken: func(_ context.Context, token, aud string) (*auth.ExternalTokenClaims, error) {
			return nil, errors.New("bad token")
		},
	}

	_, _, err := svc.LoginWithGoogle(context.Background(), "bad-token", "1.2.3.4", "unit-test")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestAuthServiceLoginWithAppleLinkConflict(t *testing.T) {
	users := &stubUsersStore{
		t: t,
		getUserByExternalFunc: func(_ context.Context, _, _ string) (domain.User, domain.ExternalAccount, error) {
			return domain.User{}, domain.ExternalAccount{}, domain.ErrNotFound
		},
		getUserByEmailFunc: func(_ context.Context, email string) (domain.UserWithPassword, error) {
			return domain.UserWithPassword{User: domain.User{ID: "user-3", Email: email, Username: "player"}}, nil
		},
		linkExternalAccountFunc: func(_ context.Context, userID, provider, providerID, email string) (domain.ExternalAccount, error) {
			return domain.ExternalAccount{}, domain.ErrExternalAccountExists
		},
	}
	svc := &AuthService{
		Users:          users,
		Sessions:       &stubSessionsStore{t: t},
		SessionTTL:     time.Hour,
		AppleServiceID: "apple-service",
		VerifyAppleIDToken: func(_ context.Context, token, aud string) (*auth.ExternalTokenClaims, error) {
			return &auth.ExternalTokenClaims{Subject: "apple-sub", Email: "player@example.com"}, nil
		},
	}

	_, _, err := svc.LoginWithApple(context.Background(), "token", "1.2.3.4", "unit-test")
	if !errors.Is(err, domain.ErrExternalAccountExists) {
		t.Fatalf("expected external account exists, got %v", err)
	}
}
