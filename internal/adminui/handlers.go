package adminui

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"MtgLeaderwebserver/internal/auth"
	"MtgLeaderwebserver/internal/domain"
)

func (a *app) handleDashboard(w http.ResponseWriter, r *http.Request) {
	a.templates.renderDashboard(w, http.StatusOK, viewData{Title: "Admin"})
}

func (a *app) handleLoginGet(w http.ResponseWriter, _ *http.Request) {
	a.templates.renderLogin(w, http.StatusOK, viewData{Title: "Admin Login"})
}

func (a *app) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.templates.renderLogin(w, http.StatusBadRequest, viewData{Title: "Admin Login", Error: "Invalid form"})
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.Form.Get("email")))
	password := r.Form.Get("password")
	if email == "" || password == "" {
		a.templates.renderLogin(w, http.StatusBadRequest, viewData{Title: "Admin Login", Error: "Email and password are required"})
		return
	}

	ip := httpapiClientIP(r)
	userAgent := r.UserAgent()

	u, sessID, err := a.authSvc.Login(r.Context(), email, password, ip, userAgent)
	if err != nil {
		a.templates.renderLogin(w, http.StatusUnauthorized, viewData{Title: "Admin Login", Error: "Invalid credentials"})
		return
	}
	if !a.adminEmails[strings.ToLower(u.Email)] {
		a.templates.renderLogin(w, http.StatusForbidden, viewData{Title: "Admin Login", Error: "Not allowed"})
		return
	}

	cookieValue := a.cookieCodec.EncodeSessionID(sessID)
	auth.SetSessionCookie(w, cookieValue, a.sessionTTL, a.cookieSecure)
	http.Redirect(w, r, "/admin/", http.StatusFound)
}

func (a *app) handleLogoutPost(w http.ResponseWriter, r *http.Request) {
	_, sessID, ok := a.currentUser(r)
	if ok {
		_ = a.authSvc.Logout(r.Context(), sessID)
	}
	auth.ClearSessionCookie(w, a.cookieSecure)
	http.Redirect(w, r, "/admin/login", http.StatusFound)
}

func (a *app) handleUsersList(w http.ResponseWriter, r *http.Request) {
	users, err := a.adminSvc.ListUsers(r.Context(), 50, 0)
	if err != nil {
		a.templates.renderError(w, http.StatusInternalServerError, "Error", "Failed to load users")
		return
	}
	rows := make([]userRow, 0, len(users))
	for _, u := range users {
		rows = append(rows, userRow{
			ID:       u.ID,
			Email:    u.Email,
			Username: u.Username,
			Status:   string(u.Status),
		})
	}
	a.templates.renderUsers(w, http.StatusOK, usersViewData{Title: "Users", Users: rows})
}

func (a *app) handlePasswordGet(w http.ResponseWriter, r *http.Request) {
	u, _, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}
	a.templates.renderPassword(w, http.StatusOK, passwordViewData{Title: "Change Password", Email: u.Email})
}

func (a *app) handlePasswordPost(w http.ResponseWriter, r *http.Request) {
	u, _, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		a.templates.renderPassword(w, http.StatusBadRequest, passwordViewData{Title: "Change Password", Email: u.Email, Error: "Invalid form"})
		return
	}

	current := r.Form.Get("current_password")
	newPassword := r.Form.Get("new_password")
	confirm := r.Form.Get("confirm_password")
	data := passwordViewData{Title: "Change Password", Email: u.Email}

	switch {
	case current == "" || newPassword == "" || confirm == "":
		data.Error = "All fields are required"
		a.templates.renderPassword(w, http.StatusBadRequest, data)
		return
	case newPassword != confirm:
		data.Error = "New passwords do not match"
		a.templates.renderPassword(w, http.StatusBadRequest, data)
		return
	case len(newPassword) < 12:
		data.Error = "New password must be at least 12 characters"
		a.templates.renderPassword(w, http.StatusBadRequest, data)
		return
	}

	if err := a.authSvc.ChangePassword(r.Context(), u.Email, current, newPassword); err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidCredentials):
			data.Error = "Current password is incorrect"
			a.templates.renderPassword(w, http.StatusUnauthorized, data)
		case errors.Is(err, domain.ErrUserDisabled):
			data.Error = "Account is disabled"
			a.templates.renderPassword(w, http.StatusForbidden, data)
		default:
			data.Error = "Failed to update password"
			a.templates.renderPassword(w, http.StatusInternalServerError, data)
		}
		return
	}

	data.Success = "Password updated"
	a.templates.renderPassword(w, http.StatusOK, data)
}

// minimal duplicate of httpapi.clientIP to avoid import cycles.
func httpapiClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
