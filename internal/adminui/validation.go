package adminui

import (
	"fmt"
	"net/mail"
	"strings"
)

func normalizeEmail(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

func validEmail(s string) bool {
	if s == "" {
		return false
	}
	_, err := mail.ParseAddress(s)
	return err == nil
}

func parseAliasEmails(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		email := normalizeEmail(part)
		if email == "" {
			continue
		}
		if !validEmail(email) {
			return nil, fmt.Errorf("invalid alias email: %s", email)
		}
		if seen[email] {
			continue
		}
		seen[email] = true
		out = append(out, email)
	}
	return out, nil
}
