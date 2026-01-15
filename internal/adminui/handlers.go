package adminui

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
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
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	notice := mapUsersNotice(strings.TrimSpace(r.URL.Query().Get("notice")))
	errMsg := mapUsersError(strings.TrimSpace(r.URL.Query().Get("error")))

	var (
		users []domain.User
		err   error
	)
	if q != "" {
		users, err = a.adminSvc.SearchUsers(r.Context(), q, 50, 0)
	} else {
		users, err = a.adminSvc.ListUsers(r.Context(), 50, 0)
	}
	if err != nil {
		a.templates.renderError(w, http.StatusInternalServerError, "Error", "Failed to load users")
		return
	}
	rows := make([]userRow, 0, len(users))
	for _, u := range users {
		lastLogin := "Never"
		if u.LastLoginAt != nil {
			lastLogin = u.LastLoginAt.Format("Jan 2, 2006 15:04")
		}
		rows = append(rows, userRow{
			ID:        u.ID,
			Email:     u.Email,
			Username:  u.Username,
			Type:      userType(u.Email, a.adminEmails, a.globalAdmin),
			Status:    string(u.Status),
			JoinedAt:  u.CreatedAt.Format("Jan 2, 2006"),
			LastLogin: lastLogin,
		})
	}

	var (
		aliases []string
		hasSMTP bool
	)
	if a.emailSvc != nil {
		settings, ok, err := a.emailSvc.GetSMTPSettings(r.Context())
		if err != nil {
			a.logger.Error("adminui: load smtp settings", "err", err)
		} else if ok {
			aliases = smtpAliases(settings)
			hasSMTP = len(aliases) > 0
		}
	}

	a.templates.renderUsers(w, http.StatusOK, usersViewData{
		Title:       "Users",
		Users:       rows,
		Query:       q,
		Notice:      notice,
		Error:       errMsg,
		FromAliases: aliases,
		HasSMTP:     hasSMTP,
	})
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

func (a *app) handleEmailGet(w http.ResponseWriter, r *http.Request) {
	if a.emailSvc == nil {
		a.templates.renderError(w, http.StatusServiceUnavailable, "Email Settings", "Email settings are unavailable.")
		return
	}
	settings, ok, err := a.emailSvc.GetSMTPSettings(r.Context())
	if err != nil {
		a.logger.Error("adminui: get smtp settings", "err", err)
		a.templates.renderError(w, http.StatusInternalServerError, "Email Settings", "Failed to load SMTP settings")
		return
	}

	data := smtpViewData{
		Title:   "Email Settings",
		TLSMode: "starttls",
		Port:    587,
	}
	if ok {
		data.Host = settings.Host
		data.Port = settings.Port
		data.Username = settings.Username
		data.TLSMode = settings.TLSMode
		data.FromName = settings.FromName
		data.FromEmail = settings.FromEmail
		data.AliasEmails = strings.Join(settings.AliasEmails, ", ")
		data.HasPassword = settings.Password != ""
	}
	a.templates.renderEmail(w, http.StatusOK, data)
}

func (a *app) handleEmailPost(w http.ResponseWriter, r *http.Request) {
	if a.emailSvc == nil {
		a.templates.renderError(w, http.StatusServiceUnavailable, "Email Settings", "Email settings are unavailable.")
		return
	}
	if err := r.ParseForm(); err != nil {
		a.templates.renderEmail(w, http.StatusBadRequest, smtpViewData{Title: "Email Settings", Error: "Invalid form"})
		return
	}

	action := strings.TrimSpace(r.FormValue("action"))
	if action == "" {
		action = "save"
	}

	host := strings.TrimSpace(r.FormValue("host"))
	portRaw := strings.TrimSpace(r.FormValue("port"))
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	tlsMode := strings.TrimSpace(strings.ToLower(r.FormValue("tls_mode")))
	fromName := strings.TrimSpace(r.FormValue("from_name"))
	fromEmail := normalizeEmail(r.FormValue("from_email"))
	aliasRaw := strings.TrimSpace(r.FormValue("alias_emails"))
	testEmail := normalizeEmail(r.FormValue("test_email"))

	view := smtpViewData{
		Title:       "Email Settings",
		Host:        host,
		Username:    username,
		TLSMode:     tlsMode,
		FromName:    fromName,
		FromEmail:   fromEmail,
		AliasEmails: aliasRaw,
		HasPassword: password != "",
		TestEmail:   testEmail,
	}

	port, err := strconv.Atoi(portRaw)
	if err != nil || port < 1 || port > 65535 {
		view.Error = "SMTP port must be between 1 and 65535"
		a.templates.renderEmail(w, http.StatusBadRequest, view)
		return
	}
	view.Port = port
	if host == "" || username == "" || fromName == "" || !validEmail(fromEmail) {
		view.Error = "Host, username, from name, and from email are required"
		a.templates.renderEmail(w, http.StatusBadRequest, view)
		return
	}
	switch tlsMode {
	case "starttls", "tls", "none":
	default:
		view.Error = "TLS mode must be starttls, tls, or none"
		a.templates.renderEmail(w, http.StatusBadRequest, view)
		return
	}

	aliases, aliasErr := parseAliasEmails(aliasRaw)
	if aliasErr != nil {
		view.Error = aliasErr.Error()
		a.templates.renderEmail(w, http.StatusBadRequest, view)
		return
	}

	existing, ok, err := a.emailSvc.GetSMTPSettings(r.Context())
	if err != nil {
		a.logger.Error("adminui: get smtp settings", "err", err)
		view.Error = "Failed to load SMTP settings"
		a.templates.renderEmail(w, http.StatusInternalServerError, view)
		return
	}
	if password == "" && ok {
		password = existing.Password
	}
	if ok && existing.Password != "" {
		view.HasPassword = true
	}
	if password == "" {
		view.Error = "SMTP password is required"
		a.templates.renderEmail(w, http.StatusBadRequest, view)
		return
	}

	settings := domain.SMTPSettings{
		Host:        host,
		Port:        port,
		Username:    username,
		Password:    password,
		TLSMode:     tlsMode,
		FromName:    fromName,
		FromEmail:   fromEmail,
		AliasEmails: aliases,
	}
	if err := a.emailSvc.SaveSMTPSettings(r.Context(), settings); err != nil {
		a.logger.Error("adminui: save smtp settings", "err", err)
		view.Error = "Failed to save SMTP settings"
		a.templates.renderEmail(w, http.StatusInternalServerError, view)
		return
	}

	if action == "test" {
		if !validEmail(testEmail) {
			view.Error = "Enter a valid test email address"
			a.templates.renderEmail(w, http.StatusBadRequest, view)
			return
		}
		if err := a.emailSvc.SendTestEmail(r.Context(), settings, testEmail); err != nil {
			a.logger.Error("adminui: send smtp test", "err", err)
			view.Error = fmt.Sprintf("Failed to send test email: %v", err)
			view.HasPassword = true
			a.templates.renderEmail(w, http.StatusInternalServerError, view)
			return
		}
		view.Success = fmt.Sprintf("Test email sent to %s", testEmail)
	} else {
		view.Success = "SMTP settings saved"
	}

	view.AliasEmails = strings.Join(aliases, ", ")
	view.HasPassword = true
	a.templates.renderEmail(w, http.StatusOK, view)
}

func (a *app) handleUserPasswordReset(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		redirectUsers(w, r, r.FormValue("q"), "", "invalid_form")
		return
	}

	action := strings.TrimSpace(r.FormValue("action"))
	if action == "" {
		action = strings.TrimSpace(r.FormValue("mode"))
	}
	userID := strings.TrimSpace(r.FormValue("user_id"))
	query := strings.TrimSpace(r.FormValue("q"))

	if userID == "" {
		redirectUsers(w, r, query, "", "missing_user")
		return
	}

	switch action {
	case "update_email":
		newEmail := normalizeEmail(r.FormValue("new_email"))
		if !validEmail(newEmail) {
			redirectUsers(w, r, query, "", "invalid_email")
			return
		}
		if err := a.adminSvc.UpdateUserEmail(r.Context(), userID, newEmail); err != nil {
			if errors.Is(err, domain.ErrEmailTaken) {
				redirectUsers(w, r, query, "", "email_taken")
				return
			}
			if errors.Is(err, domain.ErrNotFound) {
				redirectUsers(w, r, query, "", "user_not_found")
				return
			}
			a.logger.Error("adminui: update email failed", "err", err, "user_id", userID)
			redirectUsers(w, r, query, "", "email_update_failed")
			return
		}
		redirectUsers(w, r, query, "email_updated", "")
		return
	case "send_user", "send_other":
	default:
		redirectUsers(w, r, query, "", "invalid_action")
		return
	}

	if a.resetSvc == nil || a.emailSvc == nil {
		redirectUsers(w, r, query, "", "smtp_unavailable")
		return
	}

	u, err := a.adminSvc.GetUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			redirectUsers(w, r, query, "", "user_not_found")
			return
		}
		a.logger.Error("adminui: fetch user failed", "err", err, "user_id", userID)
		redirectUsers(w, r, query, "", "reset_failed")
		return
	}

	deliverTo := ""
	switch action {
	case "send_user":
		deliverTo = strings.TrimSpace(u.Email)
		if deliverTo == "" {
			redirectUsers(w, r, query, "", "email_missing")
			return
		}
		if !validEmail(deliverTo) {
			redirectUsers(w, r, query, "", "invalid_email")
			return
		}
	case "send_other":
		deliverTo = normalizeEmail(r.FormValue("deliver_to"))
		if !validEmail(deliverTo) {
			redirectUsers(w, r, query, "", "invalid_email")
			return
		}
	}

	settings, ok, err := a.emailSvc.GetSMTPSettings(r.Context())
	if err != nil || !ok {
		if err != nil {
			a.logger.Error("adminui: smtp settings missing", "err", err)
		}
		redirectUsers(w, r, query, "", "smtp_unavailable")
		return
	}
	aliases := smtpAliases(settings)
	fromAlias := strings.TrimSpace(r.FormValue("from_alias"))
	if fromAlias == "" && len(aliases) > 0 {
		fromAlias = aliases[0]
	}
	if !aliasAllowed(aliases, fromAlias) {
		redirectUsers(w, r, query, "", "invalid_alias")
		return
	}

	adminUser, _, _ := a.currentUser(r)
	token, err := a.resetSvc.CreateResetToken(r.Context(), u.ID, deliverTo, adminUser.ID)
	if err != nil {
		a.logger.Error("adminui: create reset token failed", "err", err, "user_id", userID)
		redirectUsers(w, r, query, "", "reset_failed")
		return
	}

	resetURL := a.resetLink(r, token)
	if err := a.emailSvc.SendPasswordReset(r.Context(), fromAlias, deliverTo, resetURL); err != nil {
		a.logger.Error("adminui: send reset email failed", "err", err, "user_id", userID)
		redirectUsers(w, r, query, "", "reset_failed")
		return
	}

	redirectUsers(w, r, query, "reset_sent", "")
}

func (a *app) handleUserDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		redirectUsers(w, r, "", "", "invalid_form")
		return
	}

	userID := strings.TrimSpace(r.FormValue("user_id"))
	query := strings.TrimSpace(r.FormValue("q"))
	confirm := strings.TrimSpace(r.FormValue("confirm"))
	if userID == "" {
		redirectUsers(w, r, query, "", "missing_user")
		return
	}
	if confirm != "DELETE" {
		redirectUsers(w, r, query, "", "delete_confirm")
		return
	}

	if err := a.adminSvc.DeleteUser(r.Context(), userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			redirectUsers(w, r, query, "", "user_not_found")
			return
		}
		a.logger.Error("adminui: delete user failed", "err", err, "user_id", userID)
		redirectUsers(w, r, query, "", "delete_failed")
		return
	}

	redirectUsers(w, r, query, "user_deleted", "")
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

func redirectUsers(w http.ResponseWriter, r *http.Request, q, notice, errCode string) {
	values := url.Values{}
	if q != "" {
		values.Set("q", q)
	}
	if notice != "" {
		values.Set("notice", notice)
	}
	if errCode != "" {
		values.Set("error", errCode)
	}
	target := "/admin/users"
	if len(values) > 0 {
		target += "?" + values.Encode()
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func mapUsersNotice(code string) string {
	switch code {
	case "reset_sent":
		return "Password reset email sent."
	case "email_updated":
		return "User email updated."
	case "user_deleted":
		return "User deleted."
	default:
		return ""
	}
}

func mapUsersError(code string) string {
	switch code {
	case "invalid_form":
		return "Invalid form submission."
	case "missing_user":
		return "User is required."
	case "delete_confirm":
		return "Type DELETE to confirm account deletion."
	case "invalid_action":
		return "Invalid action."
	case "invalid_email":
		return "Enter a valid email address."
	case "email_missing":
		return "User email is missing."
	case "email_taken":
		return "That email is already in use."
	case "email_update_failed":
		return "Failed to update user email."
	case "smtp_unavailable":
		return "SMTP is not configured."
	case "invalid_alias":
		return "Selected alias is not allowed."
	case "user_not_found":
		return "User not found."
	case "reset_failed":
		return "Failed to send reset email."
	case "delete_failed":
		return "Failed to delete user."
	default:
		return ""
	}
}
