package entity

import "time"

type URL struct {
	ID        string    `json:"id"`
	Original  string    `json:"original"`
	Short     string    `json:"short"`
	CreatedAt time.Time `json:"created_at"`
}
