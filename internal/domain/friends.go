package domain

import "time"

type UserSummary struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type FriendRequest struct {
	ID        string      `json:"id"`
	User      UserSummary `json:"user"`
	CreatedAt time.Time   `json:"created_at"`
}

type FriendsOverview struct {
	Friends  []UserSummary   `json:"friends"`
	Incoming []FriendRequest `json:"incoming_requests"`
	Outgoing []FriendRequest `json:"outgoing_requests"`
}
