package models

import (
	"time"

	"gorm.io/datatypes"
)

const (
	DeviceCommandStatusPending   = "pending"
	DeviceCommandStatusDelivered = "delivered"
	DeviceCommandStatusAcked     = "acked"
	DeviceCommandStatusFailed    = "failed"
	DeviceCommandStatusExpired   = "expired"
)

const (
	DeviceCommandTypeLocationRequest = "location_request"
	DeviceCommandTypeHealthCheck     = "health_check"
	DeviceCommandTypeConfigUpdate    = "config_update"
	DeviceCommandTypeAppUpdate       = "app_update"
)

// DeviceCommand — очередь удалённых команд для устройства.
type DeviceCommand struct {
	ID          string         `gorm:"primaryKey;size:36" json:"id"`
	UserID      int            `gorm:"not null;index:idx_device_cmd_user_status" json:"user_id"`
	Type        string         `gorm:"not null;size:50" json:"type"`
	Payload     datatypes.JSON `gorm:"type:jsonb" json:"payload,omitempty"`
	Status      string         `gorm:"not null;size:20;index:idx_device_cmd_user_status" json:"status"`
	AckStatus   string         `gorm:"size:50" json:"ack_status,omitempty"`
	AckMessage  string         `gorm:"type:text" json:"ack_message,omitempty"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeliveredAt *time.Time     `json:"delivered_at,omitempty"`
	AckedAt     *time.Time     `json:"acked_at,omitempty"`
}
