package models

import (
	"time"
)

type User struct {
	ID        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	ApiKey    string    `gorm:"unique;not null" json:"-"`
	IsAdmin   bool      `gorm:"default:false" json:"is_admin"`
	QRCode    string    `gorm:"type:text" json:"qr_code,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
