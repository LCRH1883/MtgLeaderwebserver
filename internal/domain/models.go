package domain

import "time"

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

type User struct {
	ID              string
	Email           string
	Username        string
	DisplayName     string
	AvatarPath      string
	Status          UserStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
	LastLoginAt     *time.Time
	AvatarUpdatedAt *time.Time
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

type SMTPSettings struct {
	Host        string
	Port        int
	Username    string
	Password    string
	TLSMode     string
	FromName    string
	FromEmail   string
	AliasEmails []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PasswordResetToken struct {
	ID          string
	UserID      string
	TokenHash   string
	SentToEmail string
	CreatedBy   string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	UsedAt      *time.Time
}
