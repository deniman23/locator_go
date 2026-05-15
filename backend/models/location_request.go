package models

import "time"

const (
	LocationRequestStatusPending   = "pending"
	LocationRequestStatusCompleted = "completed"
	LocationRequestStatusExpired   = "expired"
)

const (
	LocationSourcePeriodic = "periodic"
	LocationSourceOnDemand = "on_demand"
)

// LocationRequest — запрос на срочную отправку координат с устройства.
type LocationRequest struct {
	ID          string     `gorm:"primaryKey;size:36" json:"request_id"`
	UserID      int        `gorm:"not null;index" json:"user_id"`
	Status      string     `gorm:"not null;size:20;index" json:"status"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
