package domain

import "time"

type UserSummary struct {
	ID              string     `json:"id"`
	Username        string     `json:"username"`
	DisplayName     string     `json:"display_name,omitempty"`
	AvatarPath      string     `json:"avatar_path,omitempty"`
	AvatarUpdatedAt *time.Time `json:"avatar_updated_at,omitempty"`
	UpdatedAt       *time.Time `json:"-"`
}

type FriendRequest struct {
	ID         string      `json:"id"`
	User       UserSummary `json:"user"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
	ResolvedAt *time.Time  `json:"resolved_at,omitempty"`
}

type FriendsOverview struct {
	Friends  []UserSummary   `json:"friends"`
	Incoming []FriendRequest `json:"incoming_requests"`
	Outgoing []FriendRequest `json:"outgoing_requests"`
}

type FriendStatus string

const (
	FriendStatusAccepted FriendStatus = "accepted"
	FriendStatusIncoming FriendStatus = "incoming"
	FriendStatusOutgoing FriendStatus = "outgoing"
)

type FriendConnection struct {
	User      UserSummary  `json:"user"`
	Status    FriendStatus `json:"status"`
	RequestID string       `json:"request_id,omitempty"`
	CreatedAt time.Time    `json:"created_at,omitempty"`
	UpdatedAt time.Time    `json:"updated_at,omitempty"`
}
