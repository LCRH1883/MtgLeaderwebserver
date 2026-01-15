package adminui

import "strings"

func userType(email string, adminEmails map[string]bool, globalAdminEmail string) string {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return "User"
	}
	globalAdminEmail = strings.TrimSpace(strings.ToLower(globalAdminEmail))
	if globalAdminEmail != "" && email == globalAdminEmail {
		return "Global admin"
	}
	if adminEmails != nil && adminEmails[email] {
		return "Admin"
	}
	return "User"
}
