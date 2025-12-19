package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
	"time"
)

const SessionCookieName = "mtg_session"

type CookieCodec struct {
	secret []byte
}

func NewCookieCodec(secret []byte) CookieCodec {
	secretCopy := make([]byte, len(secret))
	copy(secretCopy, secret)
	return CookieCodec{secret: secretCopy}
}

func (c CookieCodec) EncodeSessionID(sessionID string) string {
	if len(c.secret) == 0 {
		return sessionID
	}

	mac := hmac.New(sha256.New, c.secret)
	_, _ = mac.Write([]byte(sessionID))
	sig := mac.Sum(nil)

	return sessionID + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func (c CookieCodec) DecodeSessionID(cookieValue string) (string, bool) {
	if len(c.secret) == 0 {
		return cookieValue, cookieValue != ""
	}

	id, sigB64, ok := strings.Cut(cookieValue, ".")
	if !ok || id == "" || sigB64 == "" {
		return "", false
	}

	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil || len(sig) != sha256.Size {
		return "", false
	}

	mac := hmac.New(sha256.New, c.secret)
	_, _ = mac.Write([]byte(id))
	expected := mac.Sum(nil)
	if subtle.ConstantTimeCompare(sig, expected) != 1 {
		return "", false
	}

	return id, true
}

func SetSessionCookie(w http.ResponseWriter, cookieValue string, ttl time.Duration, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    cookieValue,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
		Expires:  time.Now().Add(ttl),
	})
}

func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}
