// Package models models/event.go
package models

import "time"

// LocationEvent описывает событие обновления локации пользователя.
type LocationEvent struct {
	UserID       int       `json:"user_id"`
	CheckpointID int       `json:"checkpoint_id"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	OccurredAt   time.Time `json:"occurred_at"`
}
