package adminui

import (
	"net"
	"net/http"
	"strings"

	"MtgLeaderwebserver/internal/auth"
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
