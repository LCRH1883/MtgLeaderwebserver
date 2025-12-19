package domain

import "time"

type MatchPlayer struct {
	User     UserSummary `json:"user"`
	IsWinner bool        `json:"is_winner"`
}

type Match struct {
	ID        string        `json:"id"`
	CreatedBy string        `json:"created_by"`
	CreatedAt time.Time     `json:"created_at"`
	PlayedAt  *time.Time    `json:"played_at,omitempty"`
	WinnerID  string        `json:"winner_id,omitempty"`
	Players   []MatchPlayer `json:"players"`
}

type StatsSummary struct {
	MatchesPlayed int `json:"matches_played"`
	Wins          int `json:"wins"`
	Losses        int `json:"losses"`
}

type HeadToHeadStats struct {
	Opponent UserSummary `json:"opponent"`
	Total    int         `json:"total"`
	Wins     int         `json:"wins"`
	Losses   int         `json:"losses"`
}
