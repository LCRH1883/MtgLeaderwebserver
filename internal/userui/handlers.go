package userui

import (
	"context"
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
	"MtgLeaderwebserver/internal/service"
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
	redirectHome(w, r, "login_success", "")
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
	redirectHome(w, r, "registered", "")
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
			Error:       "Invalid form submission.",
		})
		return
	}

	displayName := strings.TrimSpace(r.FormValue("display_name"))
	updatedAt := time.Now().UTC().Truncate(time.Millisecond)
	updated, result, err := a.profileSvc.UpdateDisplayName(r.Context(), u.ID, displayName, updatedAt)
	if err != nil {
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
			Error:       msg,
		})
		return
	}
	if result == service.ProfileUpdateConflict {
		a.templates.renderProfile(w, http.StatusConflict, profileViewData{
			Title:       "Profile",
			User:        updated,
			DisplayName: updated.DisplayName,
			AvatarURL:   avatarURL(updated),
			Error:       "Profile updated elsewhere. Refresh and try again.",
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

	const (
		maxAvatarSize = 8 << 20
		avatarSize    = 96
	)
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
	if bounds.Dx() != avatarSize || bounds.Dy() != avatarSize {
		http.Error(w, "Avatar must be 96x96. Use the crop tool.", http.StatusBadRequest)
		return
	}

	if err := os.MkdirAll(a.avatarDir, 0o755); err != nil {
		a.logger.Error("userui: create avatar dir failed", "err", err)
		http.Error(w, "Failed to store avatar.", http.StatusInternalServerError)
		return
	}

	updatedAt := time.Now().UTC().Truncate(time.Millisecond)
	filename := fmt.Sprintf("%s-%d.jpg", u.ID, updatedAt.UnixNano())
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

	dst := image.NewRGBA(image.Rect(0, 0, avatarSize, avatarSize))
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

	_, result, err := a.profileSvc.UpdateAvatar(r.Context(), u.ID, filename, updatedAt)
	if err != nil {
		_ = os.Remove(targetPath)
		a.logger.Error("userui: update avatar failed", "err", err)
		http.Error(w, "Failed to update avatar.", http.StatusInternalServerError)
		return
	}
	if result != service.ProfileUpdateApplied {
		_ = os.Remove(targetPath)
		if result == service.ProfileUpdateConflict {
			http.Error(w, "Profile updated elsewhere. Refresh and try again.", http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if oldPath := strings.TrimSpace(u.AvatarPath); oldPath != "" && oldPath != filename {
		_ = os.Remove(filepath.Join(a.avatarDir, oldPath))
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleProfileDeletePost(w http.ResponseWriter, r *http.Request) {
	if a.authSvc == nil {
		a.templates.renderError(w, http.StatusServiceUnavailable, "Unavailable", "Account deletion is unavailable.")
		return
	}
	u, _, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/app/login", http.StatusFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/app/profile?error=invalid_form", http.StatusFound)
		return
	}

	confirm := strings.TrimSpace(r.FormValue("confirm"))
	if confirm != "DELETE" {
		http.Redirect(w, r, "/app/profile?error=delete_confirm", http.StatusFound)
		return
	}

	if err := a.authSvc.DeleteUser(r.Context(), u.ID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			auth.ClearSessionCookie(w, a.cookieSecure)
			http.Redirect(w, r, "/app/login?notice=account_deleted", http.StatusFound)
			return
		}
		a.logger.Error("userui: delete account failed", "err", err, "user_id", u.ID)
		http.Redirect(w, r, "/app/profile?error=delete_failed", http.StatusFound)
		return
	}

	auth.ClearSessionCookie(w, a.cookieSecure)
	http.Redirect(w, r, "/app/login?notice=account_deleted", http.StatusFound)
}

func (a *app) handleWikiRedirect(w http.ResponseWriter, r *http.Request) {
	target := "/wiki"
	if strings.Contains(r.URL.Path, "/delete-account") {
		target = "/wiki/delete-account"
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func (a *app) handleHome(w http.ResponseWriter, r *http.Request) {
	u, _, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/app/login", http.StatusFound)
		return
	}

	data := homeViewData{
		Title:  "Home",
		User:   u,
		Notice: mapNoticeCode(strings.TrimSpace(r.URL.Query().Get("notice"))),
		Error:  mapErrorCode(strings.TrimSpace(r.URL.Query().Get("error"))),
	}

	a.templates.renderHome(w, http.StatusOK, data)
}

func (a *app) handleFriends(w http.ResponseWriter, r *http.Request) {
	if a.friendsSvc == nil || a.usersSvc == nil {
		a.templates.renderError(w, http.StatusServiceUnavailable, "Unavailable", uiUnavailableMsg)
		return
	}
	u, _, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/app/login", http.StatusFound)
		return
	}

	data := friendsViewData{
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
	data.Friends = make([]friendCard, 0, len(overview.Friends))
	for _, f := range overview.Friends {
		display := strings.TrimSpace(f.DisplayName)
		if display == "" {
			display = f.Username
		}
		data.Friends = append(data.Friends, friendCard{
			Username:    f.Username,
			DisplayName: display,
			AvatarURL:   avatarURLForSummary(f),
		})
	}
	if a.matchSvc != nil && len(overview.Friends) > 0 {
		stats, err := a.friendStats(r.Context(), u.ID, overview.Friends)
		if err != nil {
			a.logger.Error("userui: friend stats failed", "err", err)
			if data.Error == "" {
				data.Error = "Friend stats unavailable."
			}
		} else {
			data.Stats = stats
		}
	}
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

	a.templates.renderFriends(w, http.StatusOK, data)
}

func (a *app) handleStats(w http.ResponseWriter, r *http.Request) {
	if a.matchSvc == nil {
		a.templates.renderError(w, http.StatusServiceUnavailable, "Unavailable", "Stats are unavailable.")
		return
	}
	u, _, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/app/login", http.StatusFound)
		return
	}

	summary, err := a.matchSvc.Summary(r.Context(), u.ID)
	if err != nil {
		a.logger.Error("userui: stats summary failed", "err", err)
		a.templates.renderError(w, http.StatusInternalServerError, "Error", "Failed to load stats")
		return
	}

	data := statsViewData{
		Title:   "Stats",
		User:    u,
		Summary: summary,
		Formats: formatStats(summary.ByFormat),
		Error:   mapErrorCode(strings.TrimSpace(r.URL.Query().Get("error"))),
		Notice:  mapNoticeCode(strings.TrimSpace(r.URL.Query().Get("notice"))),
	}
	if summary.MostOftenBeat != nil {
		data.MostOftenBeat = &opponentStatRow{
			Username: summary.MostOftenBeat.Opponent.Username,
			Count:    summary.MostOftenBeat.Count,
		}
	}
	if summary.MostOftenBeatsYou != nil {
		data.MostOftenBeatsYou = &opponentStatRow{
			Username: summary.MostOftenBeatsYou.Opponent.Username,
			Count:    summary.MostOftenBeatsYou.Count,
		}
	}

	a.templates.renderStats(w, http.StatusOK, data)
}

func (a *app) handleMatchesList(w http.ResponseWriter, r *http.Request) {
	if a.matchSvc == nil {
		a.templates.renderError(w, http.StatusServiceUnavailable, "Unavailable", "Matches are unavailable.")
		return
	}
	u, _, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/app/login", http.StatusFound)
		return
	}

	matches, err := a.matchSvc.ListMatches(r.Context(), u.ID, 25)
	if err != nil {
		a.logger.Error("userui: list matches failed", "err", err)
		a.templates.renderError(w, http.StatusInternalServerError, "Error", "Failed to load matches")
		return
	}

	rows := make([]matchListItem, 0, len(matches))
	for _, m := range matches {
		rows = append(rows, matchListItem{
			ID:        m.ID,
			PlayedAt:  formatPlayedAt(m.PlayedAt, m.CreatedAt),
			Format:    string(m.Format),
			Duration:  formatDuration(m.TotalDurationSeconds),
			TurnCount: m.TurnCount,
			Winner:    matchWinner(m),
			Players:   len(m.Players),
		})
	}

	data := matchesViewData{
		Title:   "Matches",
		User:    u,
		Matches: rows,
		Error:   mapErrorCode(strings.TrimSpace(r.URL.Query().Get("error"))),
		Notice:  mapNoticeCode(strings.TrimSpace(r.URL.Query().Get("notice"))),
	}

	a.templates.renderMatches(w, http.StatusOK, data)
}

func (a *app) handleMatchesDetail(w http.ResponseWriter, r *http.Request) {
	if a.matchSvc == nil {
		a.templates.renderError(w, http.StatusServiceUnavailable, "Unavailable", "Match details are unavailable.")
		return
	}
	u, _, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/app/login", http.StatusFound)
		return
	}

	matchID := strings.TrimSpace(r.PathValue("id"))
	if matchID == "" {
		a.templates.renderError(w, http.StatusBadRequest, "Invalid", "Match id is required.")
		return
	}

	m, err := a.matchSvc.GetMatch(r.Context(), u.ID, matchID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			a.templates.renderError(w, http.StatusNotFound, "Not found", "Match not found.")
			return
		}
		a.logger.Error("userui: match detail failed", "err", err)
		a.templates.renderError(w, http.StatusInternalServerError, "Error", "Failed to load match")
		return
	}

	avgTurn := "—"
	if m.TurnCount > 0 && m.TotalDurationSeconds > 0 {
		avgTurn = formatDuration(m.TotalDurationSeconds / m.TurnCount)
	}

	data := matchDetailViewData{
		Title:    "Match detail",
		User:     u,
		Match:    m,
		PlayedAt: formatPlayedAt(m.PlayedAt, m.CreatedAt),
		Duration: formatDuration(m.TotalDurationSeconds),
		AvgTurn:  avgTurn,
	}

	a.templates.renderMatch(w, http.StatusOK, data)
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
		redirectFriends(w, r, "", "", "", "invalid_form")
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	q := strings.TrimSpace(r.FormValue("q"))
	view := strings.TrimSpace(r.FormValue("view"))
	if username == "" {
		redirectFriends(w, r, q, view, "", "username_required")
		return
	}

	_, err := a.friendsSvc.CreateRequest(r.Context(), u.ID, username)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			redirectFriends(w, r, q, view, "", "user_not_found")
		case errors.Is(err, domain.ErrFriendshipExists):
			redirectFriends(w, r, q, view, "", "already_requested")
		case errors.Is(err, domain.ErrForbidden):
			redirectFriends(w, r, q, view, "", "user_unavailable")
		case errors.Is(err, domain.ErrValidation):
			redirectFriends(w, r, q, view, "", "invalid_request")
		default:
			a.logger.Error("userui: create request", "err", err)
			a.templates.renderError(w, http.StatusInternalServerError, "Error", "Failed to send request")
		}
		return
	}

	redirectFriends(w, r, q, view, "request_sent", "")
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
		redirectFriends(w, r, "", "", "", "invalid_form")
		return
	}

	id := strings.TrimSpace(r.FormValue("id"))
	q := strings.TrimSpace(r.FormValue("q"))
	view := strings.TrimSpace(r.FormValue("view"))
	if id == "" {
		redirectFriends(w, r, q, view, "", "invalid_request")
		return
	}

	if _, err := a.friendsSvc.Accept(r.Context(), u.ID, id, nil); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			redirectFriends(w, r, q, view, "", "request_not_found")
			return
		}
		a.logger.Error("userui: accept request", "err", err)
		a.templates.renderError(w, http.StatusInternalServerError, "Error", "Failed to accept request")
		return
	}

	redirectFriends(w, r, q, view, "request_accepted", "")
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
		redirectFriends(w, r, "", "", "", "invalid_form")
		return
	}

	id := strings.TrimSpace(r.FormValue("id"))
	q := strings.TrimSpace(r.FormValue("q"))
	view := strings.TrimSpace(r.FormValue("view"))
	if id == "" {
		redirectFriends(w, r, q, view, "", "invalid_request")
		return
	}

	if _, err := a.friendsSvc.Decline(r.Context(), u.ID, id, nil); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			redirectFriends(w, r, q, view, "", "request_not_found")
			return
		}
		a.logger.Error("userui: decline request", "err", err)
		a.templates.renderError(w, http.StatusInternalServerError, "Error", "Failed to decline request")
		return
	}

	redirectFriends(w, r, q, view, "request_declined", "")
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
		redirectFriends(w, r, "", "", "", "invalid_form")
		return
	}

	id := strings.TrimSpace(r.FormValue("id"))
	q := strings.TrimSpace(r.FormValue("q"))
	view := strings.TrimSpace(r.FormValue("view"))
	if id == "" {
		redirectFriends(w, r, q, view, "", "invalid_request")
		return
	}

	if _, err := a.friendsSvc.Cancel(r.Context(), u.ID, id, nil); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			redirectFriends(w, r, q, view, "", "request_not_found")
			return
		}
		a.logger.Error("userui: cancel request", "err", err)
		a.templates.renderError(w, http.StatusInternalServerError, "Error", "Failed to cancel request")
		return
	}

	redirectFriends(w, r, q, view, "request_cancelled", "")
}

func redirectFriends(w http.ResponseWriter, r *http.Request, q, view, notice, errCode string) {
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

	target := "/app/friends"
	if len(values) > 0 {
		target = target + "?" + values.Encode()
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func redirectHome(w http.ResponseWriter, r *http.Request, notice, errCode string) {
	values := url.Values{}
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
	case "account_deleted":
		return "Account deleted."
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
	case "invalid_form":
		return "Invalid form submission."
	case "delete_confirm":
		return "Type DELETE to confirm account deletion."
	case "delete_failed":
		return "Account deletion failed."
	case "avatar_failed":
		return "Avatar update failed."
	default:
		return ""
	}
}

const defaultAvatarURL = "/app/static/skull.svg"

func (a *app) friendStats(ctx context.Context, userID string, friends []domain.UserSummary) ([]domain.FriendStatsListItem, error) {
	items := make([]domain.FriendStatsListItem, 0, len(friends))
	for _, friend := range friends {
		stats, err := a.matchSvc.HeadToHead(ctx, userID, friend.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, domain.FriendStatsListItem{
			Friend:   friend,
			Total:    stats.Total,
			Wins:     stats.Wins,
			Losses:   stats.Losses,
			CoLosses: stats.CoLosses,
		})
	}
	return items, nil
}

func formatStats(input map[string]domain.StatsSummary) []formatStatRow {
	if len(input) == 0 {
		return nil
	}
	order := []string{
		string(domain.FormatCommander),
		string(domain.FormatBrawl),
		string(domain.FormatStandard),
		string(domain.FormatModern),
	}
	rows := make([]formatStatRow, 0, len(input))
	seen := make(map[string]bool, len(input))
	for _, key := range order {
		stats, ok := input[key]
		if !ok {
			continue
		}
		seen[key] = true
		rows = append(rows, formatStatRow{
			Format:         key,
			MatchesPlayed:  stats.MatchesPlayed,
			Wins:           stats.Wins,
			Losses:         stats.Losses,
			AvgTurnSeconds: stats.AvgTurnSeconds,
		})
	}
	for key, stats := range input {
		if seen[key] {
			continue
		}
		rows = append(rows, formatStatRow{
			Format:         key,
			MatchesPlayed:  stats.MatchesPlayed,
			Wins:           stats.Wins,
			Losses:         stats.Losses,
			AvgTurnSeconds: stats.AvgTurnSeconds,
		})
	}
	return rows
}

func formatDuration(seconds int) string {
	if seconds <= 0 {
		return "—"
	}
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	mins := seconds / 60
	if mins < 60 {
		return fmt.Sprintf("%dm", mins)
	}
	hours := mins / 60
	mins = mins % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}

func formatPlayedAt(playedAt *time.Time, createdAt time.Time) string {
	if playedAt != nil {
		return playedAt.Format("Jan 2, 2006")
	}
	return createdAt.Format("Jan 2, 2006")
}

func matchWinner(match domain.Match) string {
	if match.WinnerID == "" {
		return "—"
	}
	for _, p := range match.Players {
		if p.User.ID == match.WinnerID {
			return "@" + p.User.Username
		}
	}
	return "—"
}

func avatarURL(u domain.User) string {
	updatedAt := u.AvatarUpdatedAt
	if updatedAt == nil {
		updatedAt = &u.UpdatedAt
	}
	return avatarURLWithUpdated(u.AvatarPath, updatedAt)
}

func avatarURLForSummary(u domain.UserSummary) string {
	return avatarURLWithUpdated(u.AvatarPath, u.AvatarUpdatedAt)
}

func avatarURLWithUpdated(path string, updatedAt *time.Time) string {
	if path == "" {
		return defaultAvatarURL
	}
	escaped := url.PathEscape(path)
	if updatedAt == nil {
		return "/app/avatars/" + escaped
	}
	return "/app/avatars/" + escaped + "?v=" + strconv.FormatInt(updatedAt.Unix(), 10)
}
