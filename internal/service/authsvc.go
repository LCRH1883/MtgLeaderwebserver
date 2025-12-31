package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"MtgLeaderwebserver/internal/auth"
	"MtgLeaderwebserver/internal/domain"
)

type UsersStore interface {
	CreateUser(ctx context.Context, email, username, passwordHash string) (domain.User, error)
	GetUserByID(ctx context.Context, id string) (domain.User, error)
	GetUserByLogin(ctx context.Context, login string) (domain.UserWithPassword, error)
	GetUserByEmail(ctx context.Context, email string) (domain.UserWithPassword, error)
	GetUserByExternalAccount(ctx context.Context, provider, providerID string) (domain.User, domain.ExternalAccount, error)
	CreateUserWithExternalAccount(ctx context.Context, provider, providerID, email, username, passwordHash string) (domain.User, domain.ExternalAccount, error)
	LinkExternalAccount(ctx context.Context, userID, provider, providerID, email string) (domain.ExternalAccount, error)
	SetLastLogin(ctx context.Context, userID string, when time.Time) error
	SetPasswordHash(ctx context.Context, userID, passwordHash string) error
	DeleteUser(ctx context.Context, userID string) error
}

type SessionsStore interface {
	CreateSession(ctx context.Context, userID string, expiresAt time.Time, ip, userAgent string) (string, error)
	GetSession(ctx context.Context, sessionID string) (domain.Session, error)
	RevokeSession(ctx context.Context, sessionID string, when time.Time) error
}

type AuthService struct {
	Users      UsersStore
	Sessions   SessionsStore
	SessionTTL time.Duration
	Now        func() time.Time

	GoogleWebClientID   string
	AppleServiceID      string
	VerifyGoogleIDToken func(context.Context, string, string) (*auth.ExternalTokenClaims, error)
	VerifyAppleIDToken  func(context.Context, string, string) (*auth.ExternalTokenClaims, error)
}

func (s *AuthService) Register(ctx context.Context, email, username, password, ip, userAgent string) (domain.User, string, error) {
	if s.Now == nil {
		s.Now = time.Now
	}

	email = strings.TrimSpace(strings.ToLower(email))
	username = strings.TrimSpace(username)

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return domain.User{}, "", err
	}

	u, err := s.Users.CreateUser(ctx, email, username, passwordHash)
	if err != nil {
		return domain.User{}, "", err
	}

	sessID, err := s.Sessions.CreateSession(ctx, u.ID, s.Now().Add(s.SessionTTL), ip, userAgent)
	if err != nil {
		return domain.User{}, "", err
	}

	return u, sessID, nil
}

func (s *AuthService) Login(ctx context.Context, email, password, ip, userAgent string) (domain.User, string, error) {
	if s.Now == nil {
		s.Now = time.Now
	}

	email = strings.TrimSpace(strings.ToLower(email))

	u, err := s.Users.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.User{}, "", domain.ErrInvalidCredentials
		}
		return domain.User{}, "", err
	}
	if u.Status == domain.UserStatusDisabled {
		return domain.User{}, "", domain.ErrUserDisabled
	}

	ok, err := auth.VerifyPassword(u.PasswordHash, password)
	if err != nil {
		return domain.User{}, "", err
	}
	if !ok {
		return domain.User{}, "", domain.ErrInvalidCredentials
	}

	sessID, err := s.Sessions.CreateSession(ctx, u.ID, s.Now().Add(s.SessionTTL), ip, userAgent)
	if err != nil {
		return domain.User{}, "", err
	}

	_ = s.Users.SetLastLogin(ctx, u.ID, s.Now())

	return u.User, sessID, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, email, currentPassword, newPassword string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return domain.ErrInvalidCredentials
	}

	u, err := s.Users.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrInvalidCredentials
		}
		return err
	}
	if u.Status == domain.UserStatusDisabled {
		return domain.ErrUserDisabled
	}

	ok, err := auth.VerifyPassword(u.PasswordHash, currentPassword)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrInvalidCredentials
	}

	passwordHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.Users.SetPasswordHash(ctx, u.ID, passwordHash)
}

func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	if s.Now == nil {
		s.Now = time.Now
	}

	return s.Sessions.RevokeSession(ctx, sessionID, s.Now())
}

func (s *AuthService) DeleteUser(ctx context.Context, userID string) error {
	return s.Users.DeleteUser(ctx, userID)
}

func (s *AuthService) GetUserForSession(ctx context.Context, sessionID string) (domain.User, error) {
	sess, err := s.Sessions.GetSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.User{}, domain.ErrUnauthorized
		}
		return domain.User{}, err
	}

	u, err := s.Users.GetUserByID(ctx, sess.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.User{}, domain.ErrUnauthorized
		}
		return domain.User{}, err
	}
	if u.Status == domain.UserStatusDisabled {
		return domain.User{}, domain.ErrForbidden
	}

	return u, nil
}

func (s *AuthService) LoginWithGoogle(ctx context.Context, idToken, ip, userAgent string) (domain.User, string, error) {
	if s.GoogleWebClientID == "" {
		return domain.User{}, "", fmt.Errorf("google login not configured")
	}
	verifier := s.VerifyGoogleIDToken
	if verifier == nil {
		verifier = auth.VerifyGoogleIDToken
	}
	claims, err := verifier(ctx, idToken, s.GoogleWebClientID)
	if err != nil {
		return domain.User{}, "", domain.ErrInvalidCredentials
	}
	return s.loginWithExternalClaims(ctx, "google", claims, ip, userAgent)
}

func (s *AuthService) LoginWithApple(ctx context.Context, idToken, ip, userAgent string) (domain.User, string, error) {
	if s.AppleServiceID == "" {
		return domain.User{}, "", fmt.Errorf("apple login not configured")
	}
	verifier := s.VerifyAppleIDToken
	if verifier == nil {
		verifier = auth.VerifyAppleIDToken
	}
	claims, err := verifier(ctx, idToken, s.AppleServiceID)
	if err != nil {
		return domain.User{}, "", domain.ErrInvalidCredentials
	}
	return s.loginWithExternalClaims(ctx, "apple", claims, ip, userAgent)
}

func (s *AuthService) loginWithExternalClaims(ctx context.Context, provider string, claims *auth.ExternalTokenClaims, ip, userAgent string) (domain.User, string, error) {
	if s.Now == nil {
		s.Now = time.Now
	}
	if claims == nil || strings.TrimSpace(claims.Subject) == "" {
		return domain.User{}, "", domain.ErrInvalidCredentials
	}

	providerID := strings.TrimSpace(claims.Subject)
	email := normalizeExternalEmail(claims.Email)

	u, _, err := s.Users.GetUserByExternalAccount(ctx, provider, providerID)
	if err == nil {
		if u.Status == domain.UserStatusDisabled {
			return domain.User{}, "", domain.ErrUserDisabled
		}
		return s.createSessionForUser(ctx, u, ip, userAgent)
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, "", err
	}

	if email != "" {
		existing, err := s.Users.GetUserByEmail(ctx, email)
		if err == nil {
			if existing.Status == domain.UserStatusDisabled {
				return domain.User{}, "", domain.ErrUserDisabled
			}
			if _, err := s.Users.LinkExternalAccount(ctx, existing.ID, provider, providerID, email); err != nil {
				return domain.User{}, "", err
			}
			return s.createSessionForUser(ctx, existing.User, ip, userAgent)
		}
		if !errors.Is(err, domain.ErrNotFound) {
			return domain.User{}, "", err
		}
	}

	if email == "" {
		return domain.User{}, "", domain.NewValidationError(map[string]string{"email": "required from provider"})
	}

	passwordHash, err := hashExternalPassword()
	if err != nil {
		return domain.User{}, "", err
	}

	baseUsername := defaultExternalUsername(email, provider, providerID)
	for i := 0; i < 6; i++ {
		username := applyUsernameSuffix(baseUsername, i)
		u, _, err := s.Users.CreateUserWithExternalAccount(ctx, provider, providerID, email, username, passwordHash)
		if err == nil {
			return s.createSessionForUser(ctx, u, ip, userAgent)
		}
		if errors.Is(err, domain.ErrUsernameTaken) {
			continue
		}
		if errors.Is(err, domain.ErrEmailTaken) {
			existing, err := s.Users.GetUserByEmail(ctx, email)
			if err != nil {
				return domain.User{}, "", err
			}
			if existing.Status == domain.UserStatusDisabled {
				return domain.User{}, "", domain.ErrUserDisabled
			}
			if _, err := s.Users.LinkExternalAccount(ctx, existing.ID, provider, providerID, email); err != nil {
				return domain.User{}, "", err
			}
			return s.createSessionForUser(ctx, existing.User, ip, userAgent)
		}
		return domain.User{}, "", err
	}

	return domain.User{}, "", fmt.Errorf("unable to allocate username for external account")
}

func (s *AuthService) createSessionForUser(ctx context.Context, u domain.User, ip, userAgent string) (domain.User, string, error) {
	sessID, err := s.Sessions.CreateSession(ctx, u.ID, s.Now().Add(s.SessionTTL), ip, userAgent)
	if err != nil {
		return domain.User{}, "", err
	}
	_ = s.Users.SetLastLogin(ctx, u.ID, s.Now())
	return u, sessID, nil
}

func hashExternalPassword() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("random password: %w", err)
	}
	secret := base64.RawURLEncoding.EncodeToString(buf[:])
	return auth.HashPassword(secret)
}

func normalizeExternalEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func defaultExternalUsername(email, provider, providerID string) string {
	if email != "" {
		local, _, _ := strings.Cut(email, "@")
		base := sanitizeUsername(local)
		if base != "" {
			return base
		}
	}

	base := sanitizeUsername(provider + "_" + providerID)
	if len(base) < 3 {
		base = provider + "_user"
	}
	if len(base) > 24 {
		base = base[:24]
	}
	return base
}

func sanitizeUsername(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if len(out) < 3 {
		return ""
	}
	if len(out) > 24 {
		out = out[:24]
	}
	return out
}

func applyUsernameSuffix(base string, suffix int) string {
	if suffix <= 0 {
		return base
	}
	suffixStr := strconv.Itoa(suffix)
	maxBase := 24 - 1 - len(suffixStr)
	if maxBase < 3 {
		maxBase = 3
	}
	if len(base) > maxBase {
		base = base[:maxBase]
	}
	return base + "_" + suffixStr
}
