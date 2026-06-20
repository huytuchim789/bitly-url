package entity

import (
	"time"

	"github.com/google/uuid"
)

type Click struct {
	ID        uuid.UUID `json:"id"`
	ShortID   uuid.UUID `json:"short_id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Referer   string    `json:"referer"`
	Country   string    `json:"country,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
