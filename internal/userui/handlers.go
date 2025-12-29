package userui

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

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
	a.templates.renderLogin(w, http.StatusOK, loginViewData{Title: "MTG Friends"})
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
