package userui

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"MtgLeaderwebserver/internal/auth"
	"MtgLeaderwebserver/internal/domain"
)

const (
	loginUnavailableMsg    = "Login is unavailable. Set APP_DB_DSN and restart the server."
	registerUnavailableMsg = "Registration is unavailable. Set APP_DB_DSN and restart the server."
	uiUnavailableMsg       = "User UI is unavailable. Set APP_DB_DSN and restart the server."
)

func (a *app) handleLoginGet(w http.ResponseWriter, r *http.Request) {
	if a.authSvc == nil {
		a.templates.renderLogin(w, http.StatusServiceUnavailable, loginViewData{Title: "MTG Friends", Error: loginUnavailableMsg})
		return
	}
	if _, _, ok := a.currentUser(r); ok {
		http.Redirect(w, r, "/app/", http.StatusFound)
		return
	}
	notice := mapLoginNotice(strings.TrimSpace(r.URL.Query().Get("notice")))
	a.templates.renderLogin(w, http.StatusOK, loginViewData{Title: "MTG Friends", Notice: notice})
}

func (a *app) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	if a.authSvc == nil {
		a.templates.renderLogin(w, http.StatusServiceUnavailable, loginViewData{Title: "MTG Friends", Error: loginUnavailableMsg})
		return
	}
	if _, _, ok := a.currentUser(r); ok {
		http.Redirect(w, r, "/app/", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		a.templates.renderLogin(w, http.StatusBadRequest, loginViewData{Title: "MTG Friends", Error: "Invalid form"})
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	password := strings.TrimSpace(r.FormValue("password"))
	if email == "" || password == "" {
		a.templates.renderLogin(w, http.StatusBadRequest, loginViewData{Title: "MTG Friends", Email: email, Error: "Email and password are required"})
		return
	}

	ip := clientIP(r)
	_, sessID, err := a.authSvc.Login(r.Context(), email, password, ip, r.UserAgent())
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidCredentials):
			a.templates.renderLogin(w, http.StatusUnauthorized, loginViewData{Title: "MTG Friends", Email: email, Error: "Invalid email or password"})
		case errors.Is(err, domain.ErrUserDisabled):
			a.templates.renderLogin(w, http.StatusForbidden, loginViewData{Title: "MTG Friends", Email: email, Error: "Account disabled"})
		default:
			a.logger.Error("userui: login failed", "err", err)
			a.templates.renderLogin(w, http.StatusInternalServerError, loginViewData{Title: "MTG Friends", Email: email, Error: "Login failed"})
		}
		return
	}

	cookieValue := a.cookieCodec.EncodeSessionID(sessID)
	auth.SetSessionCookie(w, cookieValue, a.sessionTTL, a.cookieSecure)
	redirectHome(w, r, "", "", "login_success", "")
}

func (a *app) handleRegisterGet(w http.ResponseWriter, r *http.Request) {
	if a.authSvc == nil {
		a.templates.renderRegister(w, http.StatusServiceUnavailable, registerViewData{Title: "Create Account", Error: registerUnavailableMsg})
		return
	}
	if _, _, ok := a.currentUser(r); ok {
		http.Redirect(w, r, "/app/", http.StatusFound)
		return
	}
	a.templates.renderRegister(w, http.StatusOK, registerViewData{Title: "Create Account"})
}

func (a *app) handleRegisterPost(w http.ResponseWriter, r *http.Request) {
	if a.authSvc == nil {
		a.templates.renderRegister(w, http.StatusServiceUnavailable, registerViewData{Title: "Create Account", Error: registerUnavailableMsg})
		return
	}
	if _, _, ok := a.currentUser(r); ok {
		http.Redirect(w, r, "/app/", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		a.templates.renderRegister(w, http.StatusBadRequest, registerViewData{Title: "Create Account", Error: "Invalid form"})
		return
	}

	email := normalizeEmail(r.FormValue("email"))
	username := normalizeUsername(r.FormValue("username"))
	password := strings.TrimSpace(r.FormValue("password"))

	var errs []string
	if !validEmail(email) {
		errs = append(errs, "Email must be valid.")
	}
	if username == "" || !validUsername(username) {
		errs = append(errs, "Username must be 3-24 characters with letters, numbers, or underscore.")
	}
	if len(password) < 12 {
		errs = append(errs, "Password must be at least 12 characters.")
	}
	if len(errs) > 0 {
		a.templates.renderRegister(w, http.StatusBadRequest, registerViewData{
			Title:    "Create Account",
			Email:    email,
			Username: username,
			Error:    strings.Join(errs, " "),
		})
		return
	}

	ip := clientIP(r)
	_, sessID, err := a.authSvc.Register(r.Context(), email, username, password, ip, r.UserAgent())
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEmailTaken):
			a.templates.renderRegister(w, http.StatusBadRequest, registerViewData{
				Title:    "Create Account",
				Email:    email,
				Username: username,
				Error:    "That email is already in use.",
			})
		case errors.Is(err, domain.ErrUsernameTaken):
			a.templates.renderRegister(w, http.StatusBadRequest, registerViewData{
				Title:    "Create Account",
				Email:    email,
				Username: username,
				Error:    "That username is taken.",
			})
		default:
			a.logger.Error("userui: register failed", "err", err)
			a.templates.renderRegister(w, http.StatusInternalServerError, registerViewData{
				Title:    "Create Account",
				Email:    email,
				Username: username,
				Error:    "Registration failed.",
			})
		}
		return
	}

	cookieValue := a.cookieCodec.EncodeSessionID(sessID)
	auth.SetSessionCookie(w, cookieValue, a.sessionTTL, a.cookieSecure)
	redirectHome(w, r, "", "", "registered", "")
}

func (a *app) handleLogoutPost(w http.ResponseWriter, r *http.Request) {
	if a.authSvc == nil {
		auth.ClearSessionCookie(w, a.cookieSecure)
		http.Redirect(w, r, "/app/login", http.StatusFound)
		return
	}
	_, sessID, ok := a.currentUser(r)
	if ok && sessID != "" {
		_ = a.authSvc.Logout(r.Context(), sessID)
	}
	auth.ClearSessionCookie(w, a.cookieSecure)
	http.Redirect(w, r, "/app/login", http.StatusFound)
}

func (a *app) handleResetGet(w http.ResponseWriter, r *http.Request) {
	if a.resetSvc == nil {
		a.templates.renderError(w, http.StatusServiceUnavailable, "Reset Unavailable", "Password reset is unavailable.")
		return
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		a.templates.renderReset(w, http.StatusBadRequest, resetViewData{Title: "Reset Password", Error: "Reset token is required."})
		return
	}
	a.templates.renderReset(w, http.StatusOK, resetViewData{Title: "Reset Password", Token: token})
}

func (a *app) handleResetPost(w http.ResponseWriter, r *http.Request) {
	if a.resetSvc == nil {
		a.templates.renderError(w, http.StatusServiceUnavailable, "Reset Unavailable", "Password reset is unavailable.")
		return
	}
	if err := r.ParseForm(); err != nil {
		a.templates.renderReset(w, http.StatusBadRequest, resetViewData{Title: "Reset Password", Error: "Invalid form submission."})
		return
	}

	token := strings.TrimSpace(r.FormValue("token"))
	password := r.FormValue("password")
	confirm := r.FormValue("confirm")

	switch {
	case token == "":
		a.templates.renderReset(w, http.StatusBadRequest, resetViewData{Title: "Reset Password", Error: "Reset token is required."})
		return
	case password == "" || confirm == "":
		a.templates.renderReset(w, http.StatusBadRequest, resetViewData{Title: "Reset Password", Token: token, Error: "All fields are required."})
		return
	case password != confirm:
		a.templates.renderReset(w, http.StatusBadRequest, resetViewData{Title: "Reset Password", Token: token, Error: "Passwords do not match."})
		return
	case len(password) < 12:
		a.templates.renderReset(w, http.StatusBadRequest, resetViewData{Title: "Reset Password", Token: token, Error: "Password must be at least 12 characters."})
		return
	}

	if err := a.resetSvc.ResetPassword(r.Context(), token, password); err != nil {
		switch {
		case errors.Is(err, domain.ErrResetTokenInvalid):
			a.templates.renderReset(w, http.StatusBadRequest, resetViewData{Title: "Reset Password", Token: token, Error: "Reset link is invalid or already used."})
		case errors.Is(err, domain.ErrResetTokenExpired):
			a.templates.renderReset(w, http.StatusBadRequest, resetViewData{Title: "Reset Password", Token: token, Error: "Reset link has expired."})
		default:
			a.logger.Error("userui: reset password failed", "err", err)
			a.templates.renderReset(w, http.StatusInternalServerError, resetViewData{Title: "Reset Password", Token: token, Error: "Failed to reset password."})
		}
		return
	}

	http.Redirect(w, r, "/app/login?notice=password_reset", http.StatusFound)
}

func (a *app) handleProfileGet(w http.ResponseWriter, r *http.Request) {
	if a.profileSvc == nil {
		a.templates.renderError(w, http.StatusServiceUnavailable, "Unavailable", "Profile is unavailable.")
		return
	}
	u, _, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/app/login", http.StatusFound)
		return
	}

	data := profileViewData{
		Title:       "Profile",
		User:        u,
		DisplayName: u.DisplayName,
		AvatarURL:   avatarURL(u),
		Initials:    initialsForUser(u),
		Notice:      mapProfileNotice(strings.TrimSpace(r.URL.Query().Get("notice"))),
		Error:       mapProfileError(strings.TrimSpace(r.URL.Query().Get("error"))),
	}
	a.templates.renderProfile(w, http.StatusOK, data)
}

func (a *app) handleProfilePost(w http.ResponseWriter, r *http.Request) {
	if a.profileSvc == nil {
		a.templates.renderError(w, http.StatusServiceUnavailable, "Unavailable", "Profile is unavailable.")
		return
	}
	u, _, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/app/login", http.StatusFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		a.templates.renderProfile(w, http.StatusBadRequest, profileViewData{
			Title:       "Profile",
			User:        u,
			DisplayName: u.DisplayName,
			AvatarURL:   avatarURL(u),
			Initials:    initialsForUser(u),
			Error:       "Invalid form submission.",
		})
		return
	}

	displayName := strings.TrimSpace(r.FormValue("display_name"))
	if err := a.profileSvc.UpdateDisplayName(r.Context(), u.ID, displayName); err != nil {
		u.DisplayName = displayName
		msg := "Failed to update profile."
		var vErr *domain.ValidationError
		if errors.As(err, &vErr) {
			if fieldMsg, ok := vErr.Fields["display_name"]; ok {
				msg = fieldMsg
			}
		}
		a.templates.renderProfile(w, http.StatusBadRequest, profileViewData{
			Title:       "Profile",
			User:        u,
			DisplayName: displayName,
			AvatarURL:   avatarURL(u),
			Initials:    initialsForUser(u),
			Error:       msg,
		})
		return
	}

	http.Redirect(w, r, "/app/profile?notice=profile_saved", http.StatusFound)
}

func (a *app) handleProfileAvatarPost(w http.ResponseWriter, r *http.Request) {
	if a.profileSvc == nil {
		http.Error(w, "Profile is unavailable.", http.StatusServiceUnavailable)
		return
	}
	u, _, ok := a.currentUser(r)
	if !ok {
		http.Error(w, "Unauthorized.", http.StatusUnauthorized)
		return
	}

	const maxAvatarSize = 8 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarSize)
	if err := r.ParseMultipartForm(maxAvatarSize); err != nil {
		http.Error(w, "Avatar file is too large.", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("avatar")
	if err != nil {
		http.Error(w, "Avatar file is required.", http.StatusBadRequest)
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		http.Error(w, "Avatar must be a valid image file.", http.StatusBadRequest)
		return
	}
	bounds := img.Bounds()
	if bounds.Dx() != 512 || bounds.Dy() != 512 {
		http.Error(w, "Avatar must be 512x512. Use the crop tool.", http.StatusBadRequest)
		return
	}

	if err := os.MkdirAll(a.avatarDir, 0o755); err != nil {
		a.logger.Error("userui: create avatar dir failed", "err", err)
		http.Error(w, "Failed to store avatar.", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("%s.jpg", u.ID)
	targetPath := filepath.Join(a.avatarDir, filename)
	tmpFile, err := os.CreateTemp(a.avatarDir, "avatar-*")
	if err != nil {
		a.logger.Error("userui: create avatar file failed", "err", err)
		http.Error(w, "Failed to store avatar.", http.StatusInternalServerError)
		return
	}

	writeErr := func(err error) {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		a.logger.Error("userui: write avatar failed", "err", err)
		http.Error(w, "Failed to store avatar.", http.StatusInternalServerError)
	}

	dst := image.NewRGBA(image.Rect(0, 0, 512, 512))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), img, bounds.Min, draw.Over)
	if err := jpeg.Encode(tmpFile, dst, &jpeg.Options{Quality: 85}); err != nil {
		writeErr(err)
		return
	}
	if err := tmpFile.Close(); err != nil {
		writeErr(err)
		return
	}
	if err := os.Rename(tmpFile.Name(), targetPath); err != nil {
		writeErr(err)
		return
	}
	if err := os.Chmod(targetPath, 0o644); err != nil {
		a.logger.Error("userui: chmod avatar failed", "err", err)
	}

	updatedAt := time.Now()
	if err := a.profileSvc.UpdateAvatar(r.Context(), u.ID, filename, updatedAt); err != nil {
		_ = os.Remove(targetPath)
		a.logger.Error("userui: update avatar failed", "err", err)
		http.Error(w, "Failed to update avatar.", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleHome(w http.ResponseWriter, r *http.Request) {
	if a.friendsSvc == nil || a.usersSvc == nil {
		a.templates.renderError(w, http.StatusServiceUnavailable, "Unavailable", uiUnavailableMsg)
		return
	}
	u, _, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/app/login", http.StatusFound)
		return
	}

	data := homeViewData{
		Title:  "Friends",
		User:   u,
		View:   normalizeView(strings.TrimSpace(r.URL.Query().Get("view"))),
		Notice: mapNoticeCode(strings.TrimSpace(r.URL.Query().Get("notice"))),
		Error:  mapErrorCode(strings.TrimSpace(r.URL.Query().Get("error"))),
	}

	overview, err := a.friendsSvc.ListOverview(r.Context(), u.ID)
	if err != nil {
		a.logger.Error("userui: list friends", "err", err)
		a.templates.renderError(w, http.StatusInternalServerError, "Error", "Failed to load friends")
		return
	}
	data.Friends = overview.Friends
	data.Incoming = overview.Incoming
	data.Outgoing = overview.Outgoing

	friendIDs := make(map[string]bool, len(overview.Friends))
	for _, f := range overview.Friends {
		friendIDs[f.ID] = true
	}
	incomingByUser := make(map[string]string, len(overview.Incoming))
	for _, req := range overview.Incoming {
		incomingByUser[req.User.ID] = req.ID
	}
	outgoingByUser := make(map[string]string, len(overview.Outgoing))
	for _, req := range overview.Outgoing {
		outgoingByUser[req.User.ID] = req.ID
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	data.Query = q
	if q != "" {
		if len(q) < 3 {
			if data.Error == "" {
				data.Error = "Search needs at least 3 characters"
			}
		} else {
			results, err := a.usersSvc.Search(r.Context(), q, 20, u.ID)
			if err != nil {
				a.logger.Error("userui: search failed", "err", err)
				if data.Error == "" {
					data.Error = "Search failed"
				}
			} else {
				data.Results = make([]searchResult, 0, len(results))
				for _, res := range results {
					r := searchResult{
						ID:       res.ID,
						Username: res.Username,
						IsFriend: friendIDs[res.ID],
					}
					if reqID, ok := incomingByUser[res.ID]; ok {
						r.IsIncoming = true
						r.RequestID = reqID
					}
					if reqID, ok := outgoingByUser[res.ID]; ok {
						r.IsOutgoing = true
						r.RequestID = reqID
					}
					data.Results = append(data.Results, r)
				}
			}
		}
	}

	a.templates.renderHome(w, http.StatusOK, data)
}

func (a *app) handleFriendRequest(w http.ResponseWriter, r *http.Request) {
	if a.friendsSvc == nil {
		a.templates.renderError(w, http.StatusServiceUnavailable, "Unavailable", uiUnavailableMsg)
		return
	}
	u, _, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/app/login", http.StatusFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		redirectHome(w, r, "", "", "", "invalid_form")
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	q := strings.TrimSpace(r.FormValue("q"))
	view := strings.TrimSpace(r.FormValue("view"))
	if username == "" {
		redirectHome(w, r, q, view, "", "username_required")
		return
	}

	_, err := a.friendsSvc.CreateRequest(r.Context(), u.ID, username)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			redirectHome(w, r, q, view, "", "user_not_found")
		case errors.Is(err, domain.ErrFriendshipExists):
			redirectHome(w, r, q, view, "", "already_requested")
		case errors.Is(err, domain.ErrForbidden):
			redirectHome(w, r, q, view, "", "user_unavailable")
		case errors.Is(err, domain.ErrValidation):
			redirectHome(w, r, q, view, "", "invalid_request")
		default:
			a.logger.Error("userui: create request", "err", err)
			a.templates.renderError(w, http.StatusInternalServerError, "Error", "Failed to send request")
		}
		return
	}

	redirectHome(w, r, q, view, "request_sent", "")
}

func (a *app) handleFriendAccept(w http.ResponseWriter, r *http.Request) {
	if a.friendsSvc == nil {
		a.templates.renderError(w, http.StatusServiceUnavailable, "Unavailable", uiUnavailableMsg)
		return
	}
	u, _, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/app/login", http.StatusFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		redirectHome(w, r, "", "", "", "invalid_form")
		return
	}

	id := strings.TrimSpace(r.FormValue("id"))
	q := strings.TrimSpace(r.FormValue("q"))
	view := strings.TrimSpace(r.FormValue("view"))
	if id == "" {
		redirectHome(w, r, q, view, "", "invalid_request")
		return
	}

	if err := a.friendsSvc.Accept(r.Context(), u.ID, id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			redirectHome(w, r, q, view, "", "request_not_found")
			return
		}
		a.logger.Error("userui: accept request", "err", err)
		a.templates.renderError(w, http.StatusInternalServerError, "Error", "Failed to accept request")
		return
	}

	redirectHome(w, r, q, view, "request_accepted", "")
}

func (a *app) handleFriendDecline(w http.ResponseWriter, r *http.Request) {
	if a.friendsSvc == nil {
		a.templates.renderError(w, http.StatusServiceUnavailable, "Unavailable", uiUnavailableMsg)
		return
	}
	u, _, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/app/login", http.StatusFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		redirectHome(w, r, "", "", "", "invalid_form")
		return
	}

	id := strings.TrimSpace(r.FormValue("id"))
	q := strings.TrimSpace(r.FormValue("q"))
	view := strings.TrimSpace(r.FormValue("view"))
	if id == "" {
		redirectHome(w, r, q, view, "", "invalid_request")
		return
	}

	if err := a.friendsSvc.Decline(r.Context(), u.ID, id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			redirectHome(w, r, q, view, "", "request_not_found")
			return
		}
		a.logger.Error("userui: decline request", "err", err)
		a.templates.renderError(w, http.StatusInternalServerError, "Error", "Failed to decline request")
		return
	}

	redirectHome(w, r, q, view, "request_declined", "")
}

func (a *app) handleFriendCancel(w http.ResponseWriter, r *http.Request) {
	if a.friendsSvc == nil {
		a.templates.renderError(w, http.StatusServiceUnavailable, "Unavailable", uiUnavailableMsg)
		return
	}
	u, _, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/app/login", http.StatusFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		redirectHome(w, r, "", "", "", "invalid_form")
		return
	}

	id := strings.TrimSpace(r.FormValue("id"))
	q := strings.TrimSpace(r.FormValue("q"))
	view := strings.TrimSpace(r.FormValue("view"))
	if id == "" {
		redirectHome(w, r, q, view, "", "invalid_request")
		return
	}

	if err := a.friendsSvc.Cancel(r.Context(), u.ID, id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			redirectHome(w, r, q, view, "", "request_not_found")
			return
		}
		a.logger.Error("userui: cancel request", "err", err)
		a.templates.renderError(w, http.StatusInternalServerError, "Error", "Failed to cancel request")
		return
	}

	redirectHome(w, r, q, view, "request_cancelled", "")
}

func redirectHome(w http.ResponseWriter, r *http.Request, q, view, notice, errCode string) {
	values := url.Values{}
	if q != "" {
		values.Set("q", q)
	}
	if view != "" && view != "all" {
		values.Set("view", view)
	}
	if notice != "" {
		values.Set("notice", notice)
	}
	if errCode != "" {
		values.Set("error", errCode)
	}

	target := "/app/"
	if len(values) > 0 {
		target = target + "?" + values.Encode()
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func normalizeView(view string) string {
	switch view {
	case "search", "incoming", "outgoing", "friends":
		return view
	default:
		return "all"
	}
}

func mapNoticeCode(code string) string {
	switch code {
	case "login_success":
		return "Welcome back."
	case "registered":
		return "Welcome! Your account is ready."
	case "request_sent":
		return "Friend request sent."
	case "request_accepted":
		return "Friend request accepted."
	case "request_declined":
		return "Friend request declined."
	case "request_cancelled":
		return "Friend request canceled."
	default:
		return ""
	}
}

func mapLoginNotice(code string) string {
	switch code {
	case "password_reset":
		return "Password updated. Sign in with your new password."
	default:
		return ""
	}
}

func mapErrorCode(code string) string {
	switch code {
	case "invalid_form":
		return "Invalid form submission."
	case "username_required":
		return "Username is required."
	case "user_not_found":
		return "User not found."
	case "already_requested":
		return "Friend request already exists."
	case "user_unavailable":
		return "That user cannot accept requests."
	case "request_not_found":
		return "Friend request not found."
	case "invalid_request":
		return "Invalid request."
	default:
		return ""
	}
}

func mapProfileNotice(code string) string {
	switch code {
	case "profile_saved":
		return "Profile updated."
	case "avatar_saved":
		return "Avatar updated."
	default:
		return ""
	}
}

func mapProfileError(code string) string {
	switch code {
	case "avatar_failed":
		return "Avatar update failed."
	default:
		return ""
	}
}

func avatarURL(u domain.User) string {
	if u.AvatarPath == "" {
		return ""
	}
	escaped := url.PathEscape(u.AvatarPath)
	v := u.AvatarUpdatedAt
	if v == nil {
		v = &u.UpdatedAt
	}
	return "/app/avatars/" + escaped + "?v=" + strconv.FormatInt(v.Unix(), 10)
}

func initialsForUser(u domain.User) string {
	name := strings.TrimSpace(u.DisplayName)
	if name == "" {
		name = strings.TrimSpace(u.Username)
	}
	if name == "" {
		return "?"
	}
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == ' ' || r == '_' || r == '-'
	})
	if len(parts) == 0 {
		return "?"
	}
	first := []rune(parts[0])
	if len(parts) == 0 {
		return "?"
	}
	if len(parts) > 1 {
		second := []rune(parts[1])
		return strings.ToUpper(string([]rune{first[0], second[0]}))
	}
	if len(first) >= 2 {
		return strings.ToUpper(string(first[:2]))
	}
	return strings.ToUpper(string(first[0]))
}
