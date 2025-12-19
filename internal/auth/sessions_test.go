package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCookieCodec_SignAndVerify(t *testing.T) {
	codec := NewCookieCodec([]byte(strings.Repeat("x", 32)))

	encoded := codec.EncodeSessionID("abc")
	if encoded == "abc" {
		t.Fatalf("expected signed cookie value")
	}

	id, ok := codec.DecodeSessionID(encoded)
	if !ok || id != "abc" {
		t.Fatalf("expected decode ok for signed cookie")
	}

	_, ok = codec.DecodeSessionID(encoded + "x")
	if ok {
		t.Fatalf("expected tampered cookie to fail verification")
	}
}

func TestCookieCodec_Unsigned(t *testing.T) {
	codec := NewCookieCodec(nil)
	id, ok := codec.DecodeSessionID("abc")
	if !ok || id != "abc" {
		t.Fatalf("expected unsigned cookie to decode")
	}
}

func TestSessionCookieHelpers(t *testing.T) {
	rr := httptest.NewRecorder()
	SetSessionCookie(rr, "v", 10*time.Minute, false)

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Name != SessionCookieName {
		t.Fatalf("unexpected cookie name: %s", cookies[0].Name)
	}
	if cookies[0].HttpOnly != true || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected cookie attributes")
	}

	rr = httptest.NewRecorder()
	ClearSessionCookie(rr, false)
	cookies = rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].MaxAge != -1 {
		t.Fatalf("expected MaxAge=-1 on clear")
	}
}
