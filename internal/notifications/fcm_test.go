package notifications

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

type captureTransport struct {
	req  *http.Request
	body []byte
}

func (t *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.req = req
	t.body, _ = io.ReadAll(req.Body)
	_ = req.Body.Close()
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{}`)),
		Header:     make(http.Header),
	}, nil
}

func TestFCMSenderSend_NotificationIncludesAPNSAlert(t *testing.T) {
	rt := &captureTransport{}
	sender := &FCMSender{
		projectID:   "pid",
		tokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}),
		client:      &http.Client{Transport: rt},
	}

	err := sender.Send(context.Background(), "fcm-token-1", Message{
		Data: map[string]string{"type": "friend_request"},
		Notification: &Notification{
			Title: "Friend request",
			Body:  "You received a friend request.",
		},
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(rt.body, &payload); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	message, _ := payload["message"].(map[string]any)
	if message == nil {
		t.Fatalf("missing message payload")
	}

	notification, _ := message["notification"].(map[string]any)
	if notification == nil {
		t.Fatalf("missing notification payload")
	}
	if notification["title"] != "Friend request" {
		t.Fatalf("unexpected notification title: %v", notification["title"])
	}

	apns, _ := message["apns"].(map[string]any)
	if apns == nil {
		t.Fatalf("missing apns payload")
	}
	headers, _ := apns["headers"].(map[string]any)
	if headers == nil {
		t.Fatalf("missing apns headers")
	}
	if headers["apns-push-type"] != "alert" {
		t.Fatalf("unexpected apns-push-type: %v", headers["apns-push-type"])
	}
	if headers["apns-priority"] != "10" {
		t.Fatalf("unexpected apns-priority: %v", headers["apns-priority"])
	}
}

func TestFCMSenderSend_DataOnlyOmitsNotificationAndAPNS(t *testing.T) {
	rt := &captureTransport{}
	sender := &FCMSender{
		projectID:   "pid",
		tokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}),
		client:      &http.Client{Transport: rt},
	}

	err := sender.Send(context.Background(), "fcm-token-1", Message{
		Data: map[string]string{"type": "friend_request"},
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(rt.body, &payload); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	message, _ := payload["message"].(map[string]any)
	if message == nil {
		t.Fatalf("missing message payload")
	}
	if _, ok := message["notification"]; ok {
		t.Fatalf("expected notification to be omitted for data-only")
	}
	if _, ok := message["apns"]; ok {
		t.Fatalf("expected apns to be omitted for data-only")
	}
}
