package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"MtgLeaderwebserver/internal/domain"
	"MtgLeaderwebserver/internal/service"
)

type stubFriendshipsStore struct {
	t *testing.T

	createRequestFunc        func(context.Context, string, string) (string, time.Time, time.Time, error)
	acceptFunc               func(context.Context, string, string, time.Time, bool) (bool, error)
	declineFunc              func(context.Context, string, string, time.Time, bool) (bool, error)
	cancelFunc               func(context.Context, string, string, time.Time, bool) (bool, error)
	listOverviewFunc         func(context.Context, string) (domain.FriendsOverview, error)
	areFriendsFunc           func(context.Context, string, string) (bool, error)
	latestFriendshipUpdateFn func(context.Context, string) (time.Time, error)
}

func (s *stubFriendshipsStore) CreateRequest(ctx context.Context, requesterID, addresseeID string) (string, time.Time, time.Time, error) {
	if s.createRequestFunc != nil {
		return s.createRequestFunc(ctx, requesterID, addresseeID)
	}
	s.t.Fatalf("CreateRequest called unexpectedly")
	return "", time.Time{}, time.Time{}, context.Canceled
}

func (s *stubFriendshipsStore) Accept(ctx context.Context, requestID, addresseeID string, when time.Time, checkUpdatedAt bool) (bool, error) {
	if s.acceptFunc != nil {
		return s.acceptFunc(ctx, requestID, addresseeID, when, checkUpdatedAt)
	}
	s.t.Fatalf("Accept called unexpectedly")
	return false, context.Canceled
}

func (s *stubFriendshipsStore) Decline(ctx context.Context, requestID, addresseeID string, when time.Time, checkUpdatedAt bool) (bool, error) {
	if s.declineFunc != nil {
		return s.declineFunc(ctx, requestID, addresseeID, when, checkUpdatedAt)
	}
	s.t.Fatalf("Decline called unexpectedly")
	return false, context.Canceled
}

func (s *stubFriendshipsStore) Cancel(ctx context.Context, requestID, requesterID string, when time.Time, checkUpdatedAt bool) (bool, error) {
	if s.cancelFunc != nil {
		return s.cancelFunc(ctx, requestID, requesterID, when, checkUpdatedAt)
	}
	s.t.Fatalf("Cancel called unexpectedly")
	return false, context.Canceled
}

func (s *stubFriendshipsStore) ListOverview(ctx context.Context, userID string) (domain.FriendsOverview, error) {
	if s.listOverviewFunc != nil {
		return s.listOverviewFunc(ctx, userID)
	}
	s.t.Fatalf("ListOverview called unexpectedly")
	return domain.FriendsOverview{}, context.Canceled
}

func (s *stubFriendshipsStore) AreFriends(ctx context.Context, userA, userB string) (bool, error) {
	if s.areFriendsFunc != nil {
		return s.areFriendsFunc(ctx, userA, userB)
	}
	s.t.Fatalf("AreFriends called unexpectedly")
	return false, context.Canceled
}

func (s *stubFriendshipsStore) LatestFriendshipUpdate(ctx context.Context, userID string) (time.Time, error) {
	if s.latestFriendshipUpdateFn != nil {
		return s.latestFriendshipUpdateFn(ctx, userID)
	}
	s.t.Fatalf("LatestFriendshipUpdate called unexpectedly")
	return time.Time{}, context.Canceled
}

func TestFriendsAcceptConflictReturnsConnections(t *testing.T) {
	storedUpdatedAt := time.Date(2024, 6, 1, 12, 34, 56, 789000000, time.UTC)
	overview := domain.FriendsOverview{
		Friends: []domain.UserSummary{
			{ID: "user-2", Username: "alice"},
		},
	}

	store := &stubFriendshipsStore{
		t: t,
		acceptFunc: func(_ context.Context, requestID, addresseeID string, when time.Time, checkUpdatedAt bool) (bool, error) {
			if requestID != "req-1" || addresseeID != "user-1" {
				t.Fatalf("unexpected accept ids: %s %s", requestID, addresseeID)
			}
			if !checkUpdatedAt {
				t.Fatalf("expected updated_at check")
			}
			if !when.Equal(storedUpdatedAt) {
				t.Fatalf("unexpected updated_at: %s", when)
			}
			return false, nil
		},
		listOverviewFunc: func(_ context.Context, userID string) (domain.FriendsOverview, error) {
			if userID != "user-1" {
				t.Fatalf("unexpected user id: %s", userID)
			}
			return overview, nil
		},
	}

	api := &api{
		friendsSvc: &service.FriendsService{
			Friendships: store,
			Now:         func() time.Time { return storedUpdatedAt },
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/friends/requests/req-1/accept", strings.NewReader(`{"updated_at":"2024-06-01T12:34:56.789Z"}`))
	req.SetPathValue("id", "req-1")
	req = req.WithContext(context.WithValue(req.Context(), authUserKey, domain.User{ID: "user-1"}))

	rr := httptest.NewRecorder()
	api.handleFriendsAccept(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("unexpected status: %d", rr.Code)
	}

	var got []domain.FriendConnection
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	want := []domain.FriendConnection{
		{
			User: domain.UserSummary{
				ID:       "user-2",
				Username: "alice",
			},
			Status: domain.FriendStatusAccepted,
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected connections: %#v", got)
	}
}

func TestFriendsConnectionsETagNotModified(t *testing.T) {
	now := time.Date(2024, 6, 1, 12, 34, 58, 123000000, time.UTC)
	reqUpdatedAt := time.Date(2024, 6, 1, 12, 34, 57, 999000000, time.UTC)
	avatarUpdatedAt := time.Date(2024, 6, 1, 12, 34, 58, 120000000, time.UTC)

	overview := domain.FriendsOverview{
		Incoming: []domain.FriendRequest{
			{
				ID:        "req-1",
				CreatedAt: time.Date(2024, 6, 1, 12, 34, 50, 0, time.UTC),
				UpdatedAt: reqUpdatedAt,
				User: domain.UserSummary{
					ID:              "user-2",
					Username:        "alice",
					AvatarUpdatedAt: &avatarUpdatedAt,
				},
			},
		},
	}

	store := &stubFriendshipsStore{
		t: t,
		listOverviewFunc: func(_ context.Context, userID string) (domain.FriendsOverview, error) {
			if userID != "user-1" {
				t.Fatalf("unexpected user id: %s", userID)
			}
			return overview, nil
		},
		latestFriendshipUpdateFn: func(_ context.Context, userID string) (time.Time, error) {
			if userID != "user-1" {
				t.Fatalf("unexpected user id: %s", userID)
			}
			return now, nil
		},
	}

	api := &api{
		friendsSvc: &service.FriendsService{
			Friendships: store,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/friends/connections", nil)
	req = req.WithContext(context.WithValue(req.Context(), authUserKey, domain.User{ID: "user-1"}))
	rr := httptest.NewRecorder()
	api.handleFriendsConnections(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}

	connections, err := api.friendsSvc.ListConnections(req.Context(), "user-1")
	if err != nil {
		t.Fatalf("list connections: %v", err)
	}
	expectedETag := friendsConnectionsETag("user-1", connections, now)

	if got := rr.Header().Get("ETag"); got != expectedETag {
		t.Fatalf("unexpected etag: %s", got)
	}
	if got := rr.Header().Get("Cache-Control"); got != "private, max-age=0" {
		t.Fatalf("unexpected cache-control: %s", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/friends/connections", nil)
	req.Header.Set("If-None-Match", expectedETag)
	req = req.WithContext(context.WithValue(req.Context(), authUserKey, domain.User{ID: "user-1"}))
	rr = httptest.NewRecorder()
	api.handleFriendsConnections(rr, req)

	if rr.Code != http.StatusNotModified {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	if got := rr.Header().Get("ETag"); got != expectedETag {
		t.Fatalf("unexpected etag: %s", got)
	}
}
