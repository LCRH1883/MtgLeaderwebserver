package adminui

import (
	"fmt"
	"html/template"
	"net/http"
)

type templates struct {
	login     *template.Template
	dashboard *template.Template
	users     *template.Template
	password  *template.Template
	errorT    *template.Template
}

type viewData struct {
	Title   string
	Error   string
	Success string
}

type usersViewData struct {
	Title  string
	Users  []userRow
	Query  string
	Error  string
	Notice string
}

type passwordViewData struct {
	Title   string
	Error   string
	Success string
	Email   string
}

type userRow struct {
	ID        string
	Email     string
	Username  string
	Status    string
	JoinedAt  string
	LastLogin string
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
	dashboard, err := parse("templates/layout.html", "templates/dashboard.html")
	if err != nil {
		return nil, fmt.Errorf("parse dashboard: %w", err)
	}
	users, err := parse("templates/layout.html", "templates/users.html")
	if err != nil {
		return nil, fmt.Errorf("parse users: %w", err)
	}
	password, err := parse("templates/layout.html", "templates/password.html")
	if err != nil {
		return nil, fmt.Errorf("parse password: %w", err)
	}
	errorT, err := parse("templates/error.html")
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return &templates{login: login, dashboard: dashboard, users: users, password: password, errorT: errorT}, nil
}

func (t *templates) renderLogin(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = t.login.ExecuteTemplate(w, "login.html", data)
}

func (t *templates) renderDashboard(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = t.dashboard.ExecuteTemplate(w, "dashboard.html", data)
}

func (t *templates) renderUsers(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = t.users.ExecuteTemplate(w, "users.html", data)
}

func (t *templates) renderPassword(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = t.password.ExecuteTemplate(w, "password.html", data)
}

func (t *templates) renderErrorPage(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = t.errorT.ExecuteTemplate(w, "error.html", data)
}

func (t *templates) renderError(w http.ResponseWriter, status int, title, msg string) {
	t.renderErrorPage(w, status, viewData{Title: title, Error: msg})
}
