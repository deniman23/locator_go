package models

import (
	"time"

	"gorm.io/datatypes"
)

// DeviceReport — диагностический отчёт с устройства.
type DeviceReport struct {
	ID         int            `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int            `gorm:"not null;index" json:"user_id"`
	Report     datatypes.JSON `gorm:"type:jsonb;not null" json:"report"`
	Issues     datatypes.JSON `gorm:"type:jsonb" json:"issues,omitempty"`
	AppVersion string         `gorm:"size:50" json:"app_version,omitempty"`
	Platform   string         `gorm:"size:20" json:"platform,omitempty"`
	CreatedAt  time.Time      `gorm:"autoCreateTime;index" json:"created_at"`
}
