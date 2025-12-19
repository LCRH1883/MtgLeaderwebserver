package httpapi

import "strings"

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
