package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"MtgLeaderwebserver/internal/domain"
	"MtgLeaderwebserver/internal/service"
)

type stubNotificationTokensStore struct {
	t *testing.T

	upsertFunc func(context.Context, string, string, string, time.Time) (domain.NotificationToken, error)
}

func (s *stubNotificationTokensStore) UpsertToken(ctx context.Context, userID, token, platform string, when time.Time) (domain.NotificationToken, error) {
	if s.upsertFunc != nil {
		return s.upsertFunc(ctx, userID, token, platform, when)
	}
	s.t.Fatalf("UpsertToken called unexpectedly")
	return domain.NotificationToken{}, context.Canceled
}

func (s *stubNotificationTokensStore) DeleteToken(context.Context, string, string) error {
	s.t.Fatalf("DeleteToken called unexpectedly")
	return context.Canceled
}

func (s *stubNotificationTokensStore) ListTokens(context.Context, string) ([]domain.NotificationToken, error) {
	s.t.Fatalf("ListTokens called unexpectedly")
	return nil, context.Canceled
}

func TestNotificationsTokenUpsertRejectsInvalidPlatform(t *testing.T) {
	api := &api{
		notificationsSvc: &service.NotificationService{
			Tokens: &stubNotificationTokensStore{t: t},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/notifications/token", strings.NewReader(`{"token":"t","platform":"web"}`))
	req = req.WithContext(context.WithValue(req.Context(), authUserKey, domain.User{ID: "user-1"}))
	rr := httptest.NewRecorder()

	api.handleNotificationsTokenUpsert(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	var resp errorEnvelope
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error.Code != "validation_error" {
		t.Fatalf("unexpected error code: %s", resp.Error.Code)
	}
}

func TestNotificationsTokenUpsertAcceptsIOSPlatform(t *testing.T) {
	called := false
	api := &api{
		notificationsSvc: &service.NotificationService{
			Tokens: &stubNotificationTokensStore{
				t: t,
				upsertFunc: func(_ context.Context, userID, token, platform string, _ time.Time) (domain.NotificationToken, error) {
					called = true
					if userID != "user-1" || token != "t" || platform != "ios" {
						t.Fatalf("unexpected args: %s %s %s", userID, token, platform)
					}
					return domain.NotificationToken{Token: token, Platform: platform}, nil
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/notifications/token", strings.NewReader(`{"token":"t","platform":"ios"}`))
	req = req.WithContext(context.WithValue(req.Context(), authUserKey, domain.User{ID: "user-1"}))
	rr := httptest.NewRecorder()

	api.handleNotificationsTokenUpsert(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	if !called {
		t.Fatalf("expected token upsert to be called")
	}
}
