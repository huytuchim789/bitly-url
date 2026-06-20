package entity

import (
	"time"

	"github.com/google/uuid"
)

type URL struct {
	ID        uuid.UUID  `json:"id"`
	Short     string     `json:"short"`
	Original  string     `json:"original"`
	Clicks    int64      `json:"clicks"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (u *URL) IsExpired() bool {
	return u.ExpiresAt != nil && time.Now().After(*u.ExpiresAt)
}
