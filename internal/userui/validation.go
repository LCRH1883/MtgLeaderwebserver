package userui

import (
	"net/mail"
	"strings"
)

func normalizeUsername(s string) string {
	return strings.TrimSpace(s)
}

func validUsername(s string) bool {
	if len(s) < 3 || len(s) > 24 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_':
		default:
			return false
		}
	}
	return true
}

func normalizeEmail(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

func validEmail(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	_, err := mail.ParseAddress(s)
	return err == nil
}
