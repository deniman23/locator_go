package models

import (
	"github.com/google/uuid"
	"time"
)

// User описывает пользователя.
// Связь с локациями осуществляется через внешний ключ user_id в таблице locations.
type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
