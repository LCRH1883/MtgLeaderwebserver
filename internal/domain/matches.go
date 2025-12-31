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
	SeatIndex        *int        `json:"seat_index,omitempty"`
	GuestName        string      `json:"guest_name,omitempty"`
	DisplayName      string      `json:"display_name,omitempty"`
	Place            *int        `json:"place,omitempty"`
	EliminatedTurn   *int        `json:"eliminated_turn_number,omitempty"`
	EliminatedDuring *int        `json:"eliminated_during_seat_index,omitempty"`
	TotalTurnTimeMs  *int64      `json:"total_turn_time_ms,omitempty"`
	TurnsTaken       *int        `json:"turns_taken,omitempty"`
}

type MatchResultInput struct {
	ID               string `json:"id"`
	Rank             int    `json:"rank"`
	EliminationTurn  *int   `json:"elimination_turn,omitempty"`
	EliminationBatch *int   `json:"elimination_batch,omitempty"`
}

type MatchParticipantInput struct {
	SeatIndex        int
	UserID           string
	GuestName        string
	DisplayName      string
	Place            int
	EliminatedTurn   *int
	EliminatedDuring *int
	TotalTurnTimeMs  *int64
	TurnsTaken       *int
}

type Match struct {
	ID                   string        `json:"id"`
	CreatedBy            string        `json:"created_by"`
	CreatedAt            time.Time     `json:"created_at"`
	UpdatedAt            time.Time     `json:"updated_at"`
	ClientMatchID        string        `json:"client_match_id,omitempty"`
	StartedAt            *time.Time    `json:"started_at,omitempty"`
	EndedAt              *time.Time    `json:"ended_at,omitempty"`
	PlayedAt             *time.Time    `json:"played_at,omitempty"`
	WinnerID             string        `json:"winner_id,omitempty"`
	Format               GameFormat    `json:"format"`
	TotalDurationSeconds int           `json:"total_duration_seconds"`
	TurnCount            int           `json:"turn_count"`
	Players              []MatchPlayer `json:"players"`
}

type StatsSummary struct {
	MatchesPlayed     int                     `json:"matches_played"`
	Wins              int                     `json:"wins"`
	Losses            int                     `json:"losses"`
	WinPct            float64                 `json:"win_pct,omitempty"`
	AvgTurnSeconds    int                     `json:"avg_turn_seconds"`
	ByFormat          map[string]StatsSummary `json:"by_format,omitempty"`
	MostOftenBeat     *OpponentStat           `json:"most_often_beat,omitempty"`
	MostOftenBeatsYou *OpponentStat           `json:"most_often_beats_you,omitempty"`
	GuestHeadToHead   []GuestHeadToHeadStat   `json:"guest_head_to_head,omitempty"`
}

type OpponentStat struct {
	Opponent UserSummary `json:"opponent"`
	Count    int         `json:"count"`
}

type GuestHeadToHeadStat struct {
	GuestName string `json:"guest_name"`
	Wins      int    `json:"wins"`
	Losses    int    `json:"losses"`
}

type HeadToHeadStats struct {
	Opponent UserSummary                `json:"opponent"`
	Total    int                        `json:"total"`
	Wins     int                        `json:"wins"`
	Losses   int                        `json:"losses"`
	CoLosses int                        `json:"co_losses"`
	ByFormat map[string]HeadToHeadStats `json:"by_format,omitempty"`
}

type FriendStatsListItem struct {
	Friend   UserSummary `json:"friend"`
	Total    int         `json:"total"`
	Wins     int         `json:"wins"`
	Losses   int         `json:"losses"`
	CoLosses int         `json:"co_losses"`
}
