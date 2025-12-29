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
	friends  *template.Template
	stats    *template.Template
	matches  *template.Template
	match    *template.Template
	profile  *template.Template
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
	Title  string
	User   domain.User
	Error  string
	Notice string
}

type friendsViewData struct {
	Title    string
	User     domain.User
	View     string
	Query    string
	Results  []searchResult
	Friends  []friendCard
	Stats    []domain.FriendStatsListItem
	Incoming []domain.FriendRequest
	Outgoing []domain.FriendRequest
	Error    string
	Notice   string
}

type profileViewData struct {
	Title       string
	User        domain.User
	DisplayName string
	AvatarURL   string
	Error       string
	Notice      string
}

type statsViewData struct {
	Title             string
	User              domain.User
	Summary           domain.StatsSummary
	Formats           []formatStatRow
	MostOftenBeat     *opponentStatRow
	MostOftenBeatsYou *opponentStatRow
	Error             string
	Notice            string
}

type formatStatRow struct {
	Format          string
	MatchesPlayed   int
	Wins            int
	Losses          int
	AvgTurnSeconds  int
}

type opponentStatRow struct {
	Username string
	Count    int
}

type matchesViewData struct {
	Title   string
	User    domain.User
	Matches []matchListItem
	Error   string
	Notice  string
}

type matchListItem struct {
	ID        string
	PlayedAt  string
	Format    string
	Duration  string
	TurnCount int
	Winner    string
	Players   int
}

type matchDetailViewData struct {
	Title      string
	User       domain.User
	Match      domain.Match
	PlayedAt   string
	Duration   string
	AvgTurn    string
	Error      string
}

type searchResult struct {
	ID         string
	Username   string
	IsFriend   bool
	IsOutgoing bool
	IsIncoming bool
	RequestID  string
}

type friendCard struct {
	Username    string
	DisplayName string
	AvatarURL   string
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
	friends, err := parse("templates/layout.html", "templates/friends.html")
	if err != nil {
		return nil, fmt.Errorf("parse friends: %w", err)
	}
	statsT, err := parse("templates/layout.html", "templates/stats.html")
	if err != nil {
		return nil, fmt.Errorf("parse stats: %w", err)
	}
	matchesT, err := parse("templates/layout.html", "templates/matches.html")
	if err != nil {
		return nil, fmt.Errorf("parse matches: %w", err)
	}
	matchT, err := parse("templates/layout.html", "templates/match.html")
	if err != nil {
		return nil, fmt.Errorf("parse match: %w", err)
	}
	profile, err := parse("templates/layout.html", "templates/profile.html")
	if err != nil {
		return nil, fmt.Errorf("parse profile: %w", err)
	}
	resetT, err := parse("templates/reset.html")
	if err != nil {
		return nil, fmt.Errorf("parse reset: %w", err)
	}
	errorT, err := parse("templates/error.html")
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return &templates{
		login:   login,
		register: register,
		home:    home,
		friends: friends,
		stats:   statsT,
		matches: matchesT,
		match:   matchT,
		profile: profile,
		reset:   resetT,
		errorT:  errorT,
	}, nil
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

func (t *templates) renderFriends(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = t.friends.ExecuteTemplate(w, "friends.html", data)
}

func (t *templates) renderStats(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = t.stats.ExecuteTemplate(w, "stats.html", data)
}

func (t *templates) renderMatches(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = t.matches.ExecuteTemplate(w, "matches.html", data)
}

func (t *templates) renderMatch(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = t.match.ExecuteTemplate(w, "match.html", data)
}

func (t *templates) renderProfile(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = t.profile.ExecuteTemplate(w, "profile.html", data)
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
