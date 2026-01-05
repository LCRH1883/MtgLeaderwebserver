package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const fcmScope = "https://www.googleapis.com/auth/firebase.messaging"

var ErrInvalidToken = errors.New("fcm_invalid_token")

type FCMSender struct {
	projectID   string
	tokenSource oauth2.TokenSource
	client      *http.Client
}

func NewFCMSender(ctx context.Context, projectID, credentialsPath string) (*FCMSender, error) {
	if strings.TrimSpace(credentialsPath) == "" {
		return nil, fmt.Errorf("fcm credentials path required")
	}
	raw, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("read fcm credentials: %w", err)
	}
	creds, err := google.CredentialsFromJSON(ctx, raw, fcmScope)
	if err != nil {
		return nil, fmt.Errorf("load fcm credentials: %w", err)
	}
	if projectID == "" {
		projectID = creds.ProjectID
	}
	if projectID == "" {
		return nil, fmt.Errorf("fcm project id required")
	}
	return &FCMSender{
		projectID:   projectID,
		tokenSource: creds.TokenSource,
		client:      http.DefaultClient,
	}, nil
}

func (s *FCMSender) Send(ctx context.Context, token string, data map[string]string) error {
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("fcm token required")
	}
	if s == nil {
		return fmt.Errorf("fcm sender not configured")
	}
	msg := fcmRequest{
		Message: fcmMessage{
			Token: token,
			Data:  data,
			Android: fcmAndroidConfig{
				Priority: "HIGH",
			},
		},
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal fcm payload: %w", err)
	}
	accessToken, err := s.tokenSource.Token()
	if err != nil {
		return fmt.Errorf("fcm access token: %w", err)
	}
	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", s.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build fcm request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken.AccessToken)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send fcm request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		io.Copy(io.Discard, resp.Body)
		return nil
	}

	rawBody, _ := io.ReadAll(resp.Body)
	if err := fcmErrorFromResponse(rawBody); err != nil {
		return err
	}
	return fmt.Errorf("fcm send failed: status %d: %s", resp.StatusCode, string(rawBody))
}

type fcmRequest struct {
	Message fcmMessage `json:"message"`
}

type fcmMessage struct {
	Token   string            `json:"token"`
	Data    map[string]string `json:"data,omitempty"`
	Android fcmAndroidConfig  `json:"android,omitempty"`
}

type fcmAndroidConfig struct {
	Priority string `json:"priority,omitempty"`
}

type fcmErrorResponse struct {
	Error struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Details []struct {
			Type      string `json:"@type"`
			ErrorCode string `json:"errorCode"`
		} `json:"details"`
	} `json:"error"`
}

func fcmErrorFromResponse(body []byte) error {
	if len(body) == 0 {
		return fmt.Errorf("fcm send failed: empty response")
	}
	var resp fcmErrorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("fcm send failed: %s", string(body))
	}
	for _, detail := range resp.Error.Details {
		if detail.ErrorCode == "UNREGISTERED" {
			return fmt.Errorf("%w: %s", ErrInvalidToken, resp.Error.Message)
		}
	}
	return fmt.Errorf("fcm send failed: %s", resp.Error.Message)
}
