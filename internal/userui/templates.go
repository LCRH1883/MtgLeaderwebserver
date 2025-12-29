package userui

import (
	"fmt"
	"html/template"
	"net/http"

	"MtgLeaderwebserver/internal/domain"
)

type templates struct {
	login    *template.Template
	register *template.Template
	home     *template.Template
	reset    *template.Template
	errorT   *template.Template
}

type viewData struct {
	Title  string
	Error  string
	Notice string
}

type loginViewData struct {
	Title  string
	Email  string
	Error  string
	Notice string
}

type registerViewData struct {
	Title    string
	Email    string
	Username string
	Error    string
}

type resetViewData struct {
	Title  string
	Token  string
	Error  string
	Notice string
}

type homeViewData struct {
	Title    string
	User     domain.User
	View     string
	Query    string
	Results  []searchResult
	Friends  []domain.UserSummary
	Incoming []domain.FriendRequest
	Outgoing []domain.FriendRequest
	Error    string
	Notice   string
}

type searchResult struct {
	ID         string
	Username   string
	IsFriend   bool
	IsOutgoing bool
	IsIncoming bool
	RequestID  string
}

func parseTemplates() (*templates, error) {
	parse := func(files ...string) (*template.Template, error) {
		t, err := template.New("base").ParseFS(assets, files...)
		if err != nil {
			return nil, err
		}
		return t, nil
	}

	login, err := parse("templates/login.html")
	if err != nil {
		return nil, fmt.Errorf("parse login: %w", err)
	}
	register, err := parse("templates/register.html")
	if err != nil {
		return nil, fmt.Errorf("parse register: %w", err)
	}
	home, err := parse("templates/layout.html", "templates/home.html")
	if err != nil {
		return nil, fmt.Errorf("parse home: %w", err)
	}
	resetT, err := parse("templates/reset.html")
	if err != nil {
		return nil, fmt.Errorf("parse reset: %w", err)
	}
	errorT, err := parse("templates/error.html")
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return &templates{login: login, register: register, home: home, reset: resetT, errorT: errorT}, nil
}

func (t *templates) renderLogin(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = t.login.ExecuteTemplate(w, "login.html", data)
}

func (t *templates) renderRegister(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = t.register.ExecuteTemplate(w, "register.html", data)
}

func (t *templates) renderHome(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = t.home.ExecuteTemplate(w, "home.html", data)
}

func (t *templates) renderReset(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = t.reset.ExecuteTemplate(w, "reset.html", data)
}

func (t *templates) renderErrorPage(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = t.errorT.ExecuteTemplate(w, "error.html", data)
}

func (t *templates) renderError(w http.ResponseWriter, status int, title, msg string) {
	t.renderErrorPage(w, status, viewData{Title: title, Error: msg})
}
