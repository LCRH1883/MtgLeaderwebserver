package domain

import "time"

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

type User struct {
	ID          string
	Email       string
	Username    string
	Status      UserStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastLoginAt *time.Time
}

type UserWithPassword struct {
	User
	PasswordHash string
}

type ExternalAccount struct {
	ID         string
	UserID     string
	Provider   string
	ProviderID string
	Email      string
	CreatedAt  time.Time
}

type Session struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}
