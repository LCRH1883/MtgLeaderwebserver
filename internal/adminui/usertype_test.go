package adminui

import "testing"

func TestUserType(t *testing.T) {
	admins := map[string]bool{
		"admin@example.com": true,
	}

	tests := []struct {
		name        string
		email       string
		global      string
		adminEmails map[string]bool
		want        string
	}{
		{name: "blank email", email: "", global: "root@example.com", adminEmails: admins, want: "User"},
		{name: "regular user", email: "user@example.com", global: "root@example.com", adminEmails: admins, want: "User"},
		{name: "admin", email: "admin@example.com", global: "root@example.com", adminEmails: admins, want: "Admin"},
		{name: "admin casing/whitespace", email: "  ADMIN@EXAMPLE.COM ", global: "root@example.com", adminEmails: admins, want: "Admin"},
		{name: "global admin", email: "root@example.com", global: "root@example.com", adminEmails: admins, want: "Global admin"},
		{name: "global admin wins", email: "admin@example.com", global: "admin@example.com", adminEmails: admins, want: "Global admin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := userType(tt.email, tt.adminEmails, tt.global); got != tt.want {
				t.Fatalf("userType() = %q, want %q", got, tt.want)
			}
		})
	}
}
