package domain

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	ErrUnauthorized          = errors.New("unauthorized")
	ErrForbidden             = errors.New("forbidden")
	ErrNotFound              = errors.New("not_found")
	ErrUsernameTaken         = errors.New("username_taken")
	ErrEmailTaken            = errors.New("email_taken")
	ErrInvalidCredentials    = errors.New("invalid_credentials")
	ErrUserDisabled          = errors.New("user_disabled")
	ErrFriendshipExists      = errors.New("friendship_exists")
	ErrExternalAccountExists = errors.New("external_account_exists")
	ErrResetTokenInvalid     = errors.New("reset_token_invalid")
	ErrResetTokenExpired     = errors.New("reset_token_expired")
	ErrValidation            = errors.New("validation")
)

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	if len(e.Fields) == 0 {
		return "validation failed"
	}
	keys := make([]string, 0, len(e.Fields))
	for k := range e.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %s", k, e.Fields[k]))
	}
	return "validation failed: " + strings.Join(parts, ", ")
}

func (e *ValidationError) Unwrap() error { return ErrValidation }

func NewValidationError(fields map[string]string) error {
	return &ValidationError{Fields: fields}
}
