package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const fcmScope = "https://www.googleapis.com/auth/firebase.messaging"
const cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

var ErrInvalidToken = errors.New("fcm_invalid_token")

type Notification struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
}

type Message struct {
	Data         map[string]string
	Notification *Notification
}

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

	scopes := []string{fcmScope, cloudPlatformScope}

	var meta struct {
		Type      string `json:"type"`
		ProjectID string `json:"project_id"`
	}
	_ = json.Unmarshal(raw, &meta)

	var tokenSource oauth2.TokenSource
	switch meta.Type {
	case "service_account":
		conf, err := google.JWTConfigFromJSON(raw, scopes...)
		if err != nil {
			return nil, fmt.Errorf("load fcm credentials (service account): %w", err)
		}
		tokenSource = conf.TokenSource(ctx)
		if projectID == "" {
			projectID = meta.ProjectID
		}
	default:
		creds, err := google.CredentialsFromJSON(ctx, raw, scopes...)
		if err != nil {
			return nil, fmt.Errorf("load fcm credentials: %w", err)
		}
		tokenSource = creds.TokenSource
		if projectID == "" {
			projectID = creds.ProjectID
		}
	}

	if projectID == "" {
		return nil, fmt.Errorf("fcm project id required")
	}
	return &FCMSender{
		projectID:   projectID,
		tokenSource: tokenSource,
		client:      http.DefaultClient,
	}, nil
}

func (s *FCMSender) Send(ctx context.Context, token string, msg Message) error {
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("fcm token required")
	}
	if s == nil {
		return fmt.Errorf("fcm sender not configured")
	}
	client := s.client
	if client == nil {
		client = http.DefaultClient
	}

	fcmMsg := fcmMessage{
		Token:        token,
		Data:         msg.Data,
		Notification: msg.Notification,
		Android: fcmAndroidConfig{
			Priority: "HIGH",
		},
	}
	if msg.Notification != nil {
		fcmMsg.APNS = &fcmAPNSConfig{
			Headers: map[string]string{
				"apns-push-type": "alert",
				"apns-priority":  "10",
			},
			Payload: fcmAPNSPayload{
				APS: fcmAPS{
					Alert: &fcmAPSAlert{
						Title: msg.Notification.Title,
						Body:  msg.Notification.Body,
					},
					Sound: "default",
				},
			},
		}
	}

	reqMsg := fcmRequest{Message: fcmMsg}
	body, err := json.Marshal(reqMsg)
	if err != nil {
		return fmt.Errorf("marshal fcm payload: %w", err)
	}
	accessToken, err := s.tokenSource.Token()
	if err != nil {
		return fmt.Errorf("fcm access token: %w", err)
	}
	if strings.TrimSpace(accessToken.AccessToken) == "" {
		return fmt.Errorf("fcm access token: empty")
	}
	accessTokenRaw := accessToken.AccessToken
	accessTokenLen := len(accessTokenRaw)
	accessTokenHasYa29 := strings.HasPrefix(accessTokenRaw, "ya29.")
	accessTokenLooksJWT := strings.Count(accessTokenRaw, ".") >= 2
	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", s.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build fcm request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessTokenRaw)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send fcm request: %w", err)
	}
	defer resp.Body.Close()
	finalURL := ""
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		io.Copy(io.Discard, resp.Body)
		return nil
	}

	rawBody, _ := io.ReadAll(resp.Body)
	wwwAuth := resp.Header.Get("Www-Authenticate")
	if err := fcmErrorFromResponse(rawBody); err != nil {
		if strings.Contains(err.Error(), "fcm send failed: UNAUTHENTICATED:") {
			scope, expiresIn, tokenInfoErr := tokenInfo(ctx, client, accessTokenRaw)
			if tokenInfoErr != nil {
				return fmt.Errorf("%w (http_status=%d final_url=%q access_token_len=%d ya29=%t jwt=%t www_auth=%q tokeninfo_err=%q)", err, resp.StatusCode, finalURL, accessTokenLen, accessTokenHasYa29, accessTokenLooksJWT, wwwAuth, tokenInfoErr.Error())
			}
			return fmt.Errorf("%w (http_status=%d final_url=%q access_token_len=%d ya29=%t jwt=%t www_auth=%q tokeninfo_scope=%q tokeninfo_expires_in=%s)", err, resp.StatusCode, finalURL, accessTokenLen, accessTokenHasYa29, accessTokenLooksJWT, wwwAuth, scope, expiresIn)
		}
		return err
	}
	return fmt.Errorf("fcm send failed: status %d: %s", resp.StatusCode, string(rawBody))
}

func tokenInfo(ctx context.Context, client *http.Client, accessToken string) (scope string, expiresIn string, err error) {
	reqURL := "https://oauth2.googleapis.com/tokeninfo?access_token=" + url.QueryEscape(accessToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out struct {
		Scope     string `json:"scope"`
		ExpiresIn string `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", "", err
	}
	return out.Scope, out.ExpiresIn, nil
}

type fcmRequest struct {
	Message fcmMessage `json:"message"`
}

type fcmMessage struct {
	Token        string            `json:"token"`
	Data         map[string]string `json:"data,omitempty"`
	Notification *Notification     `json:"notification,omitempty"`
	Android      fcmAndroidConfig  `json:"android,omitempty"`
	APNS         *fcmAPNSConfig    `json:"apns,omitempty"`
}

type fcmAndroidConfig struct {
	Priority string `json:"priority,omitempty"`
}

type fcmAPNSConfig struct {
	Headers map[string]string `json:"headers,omitempty"`
	Payload fcmAPNSPayload    `json:"payload,omitempty"`
}

type fcmAPNSPayload struct {
	APS fcmAPS `json:"aps"`
}

type fcmAPS struct {
	Alert *fcmAPSAlert `json:"alert,omitempty"`
	Sound string       `json:"sound,omitempty"`
}

type fcmAPSAlert struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
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
			return fmt.Errorf("%w: %s: %s", ErrInvalidToken, resp.Error.Status, resp.Error.Message)
		}
	}
	return fmt.Errorf("fcm send failed: %s: %s", resp.Error.Status, resp.Error.Message)
}
