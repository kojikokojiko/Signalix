package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                uuid.UUID
	Email             string
	PasswordHash      string
	DisplayName       string
	AvatarURL         *string
	PreferredLanguage string
	IsAdmin           bool
	IsActive          bool
	LastLoginAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
