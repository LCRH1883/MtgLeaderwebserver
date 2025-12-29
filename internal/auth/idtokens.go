package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/HendrickPhan/go-verify-apple-id-token/validator"
	"google.golang.org/api/idtoken"
)

type ExternalTokenClaims struct {
	Issuer  string
	Subject string
	Email   string
}

func VerifyGoogleIDToken(ctx context.Context, tokenString, expectedAud string) (*ExternalTokenClaims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, errors.New("missing id token")
	}
	if strings.TrimSpace(expectedAud) == "" {
		return nil, errors.New("missing google client id")
	}

	payload, err := idtoken.Validate(ctx, tokenString, expectedAud)
	if err != nil {
		return nil, err
	}
	if payload.Issuer != "accounts.google.com" && payload.Issuer != "https://accounts.google.com" {
		return nil, fmt.Errorf("unexpected issuer: %s", payload.Issuer)
	}

	email := ""
	if raw, ok := payload.Claims["email"]; ok {
		if v, ok := raw.(string); ok {
			email = v
		}
	}

	return &ExternalTokenClaims{
		Issuer:  payload.Issuer,
		Subject: payload.Subject,
		Email:   strings.TrimSpace(strings.ToLower(email)),
	}, nil
}

func VerifyAppleIDToken(ctx context.Context, tokenString, expectedAud string) (*ExternalTokenClaims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, errors.New("missing id token")
	}
	if strings.TrimSpace(expectedAud) == "" {
		return nil, errors.New("missing apple service id")
	}

	client := validator.NewClient()
	idToken, err := client.VerifyIdToken(expectedAud, tokenString)
	if err != nil {
		return nil, err
	}
	if idToken.Iss != "https://appleid.apple.com" {
		return nil, fmt.Errorf("unexpected issuer: %s", idToken.Iss)
	}

	_ = ctx
	return &ExternalTokenClaims{
		Issuer:  idToken.Iss,
		Subject: idToken.Sub,
		Email:   strings.TrimSpace(strings.ToLower(idToken.Email)),
	}, nil
}
