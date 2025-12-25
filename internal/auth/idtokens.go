package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/HendrickPhan/go-verify-apple-id-token/appleid"
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

	return &ExternalTokenClaims{
		Issuer:  payload.Issuer,
		Subject: payload.Subject,
		Email:   strings.TrimSpace(strings.ToLower(payload.Email)),
	}, nil
}

func VerifyAppleIDToken(ctx context.Context, tokenString, expectedAud string) (*ExternalTokenClaims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, errors.New("missing id token")
	}
	if strings.TrimSpace(expectedAud) == "" {
		return nil, errors.New("missing apple service id")
	}

	client, err := appleid.New()
	if err != nil {
		return nil, fmt.Errorf("apple id token client: %w", err)
	}

	idToken, err := client.VerifyIdToken(expectedAud, tokenString)
	if err != nil {
		return nil, err
	}
	if idToken.Issuer != "https://appleid.apple.com" {
		return nil, fmt.Errorf("unexpected issuer: %s", idToken.Issuer)
	}

	_ = ctx
	return &ExternalTokenClaims{
		Issuer:  idToken.Issuer,
		Subject: idToken.Subject,
		Email:   strings.TrimSpace(strings.ToLower(idToken.Email)),
	}, nil
}
