package domain

import "time"

type GameFormat string

const (
	FormatCommander GameFormat = "commander"
	FormatBrawl     GameFormat = "brawl"
	FormatStandard  GameFormat = "standard"
	FormatModern    GameFormat = "modern"
)

type MatchPlayer struct {
	User             UserSummary `json:"user"`
	IsWinner         bool        `json:"is_winner"`
	Rank             *int        `json:"rank,omitempty"`
	EliminationTurn  *int        `json:"elimination_turn,omitempty"`
	EliminationBatch *int        `json:"elimination_batch,omitempty"`
}

type MatchResultInput struct {
	ID               string `json:"id"`
	Rank             int    `json:"rank"`
	EliminationTurn  *int   `json:"elimination_turn,omitempty"`
	EliminationBatch *int   `json:"elimination_batch,omitempty"`
}

type Match struct {
	ID                   string        `json:"id"`
	CreatedBy            string        `json:"created_by"`
	CreatedAt            time.Time     `json:"created_at"`
	PlayedAt             *time.Time    `json:"played_at,omitempty"`
	WinnerID             string        `json:"winner_id,omitempty"`
	Format               GameFormat    `json:"format"`
	TotalDurationSeconds int           `json:"total_duration_seconds"`
	TurnCount            int           `json:"turn_count"`
	Players              []MatchPlayer `json:"players"`
}

type StatsSummary struct {
	MatchesPlayed  int                    `json:"matches_played"`
	Wins           int                    `json:"wins"`
	Losses         int                    `json:"losses"`
	AvgTurnSeconds int                    `json:"avg_turn_seconds"`
	ByFormat       map[string]StatsSummary `json:"by_format,omitempty"`
	MostOftenBeat     *OpponentStat `json:"most_often_beat,omitempty"`
	MostOftenBeatsYou *OpponentStat `json:"most_often_beats_you,omitempty"`
}

type OpponentStat struct {
	Opponent UserSummary `json:"opponent"`
	Count    int         `json:"count"`
}

type HeadToHeadStats struct {
	Opponent UserSummary              `json:"opponent"`
	Total    int                      `json:"total"`
	Wins     int                      `json:"wins"`
	Losses   int                      `json:"losses"`
	CoLosses int                      `json:"co_losses"`
	ByFormat map[string]HeadToHeadStats `json:"by_format,omitempty"`
}

type FriendStatsListItem struct {
	Friend   UserSummary `json:"friend"`
	Total    int         `json:"total"`
	Wins     int         `json:"wins"`
	Losses   int         `json:"losses"`
	CoLosses int         `json:"co_losses"`
}
