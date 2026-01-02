package domain

import "time"

type NotificationToken struct {
	ID        string    `json:"-"`
	UserID    string    `json:"-"`
	Token     string    `json:"token"`
	Platform  string    `json:"platform"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
