package models

import (
	"time"

	"github.com/google/uuid"
)

// Location описывает данные о местоположении отдельного объекта (например, сотрудника или ребенка).
type Location struct {
	// ID — уникальный идентификатор записи (UUID).
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`

	// UserID — идентификатор пользователя (целое число).
	UserID int `gorm:"not null" json:"user_id"`

	// Latitude — широта.
	Latitude float64 `gorm:"not null" json:"latitude"`

	// Longitude — долгота.
	Longitude float64 `gorm:"not null" json:"longitude"`

	// CreatedAt — время создания записи.
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt — время последнего обновления записи.
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// NewLocation создаёт новую запись о местоположении с заданными координатами для пользователя.
func NewLocation(userID int, lat, lon float64) *Location {
	now := time.Now()
	return &Location{
		ID:        uuid.New(),
		UserID:    userID,
		Latitude:  lat,
		Longitude: lon,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// UpdateCoordinates обновляет координаты записи и выставляет новое время обновления.
func (loc *Location) UpdateCoordinates(lat, lon float64) {
	loc.Latitude = lat
	loc.Longitude = lon
	loc.UpdatedAt = time.Now()
}
