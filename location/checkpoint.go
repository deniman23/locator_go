package location

import (
	"time"

	"github.com/google/uuid"
)

// Checkpoint описывает точку (чекпоинт), которую должен посещать пользователь.
// Например, если родитель хочет следить за тем, был ли ребенок в школе.
type Checkpoint struct {
	// ID — уникальный идентификатор записи (UUID).
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`

	// Name — название чекпоинта, например "Школа".
	Name string `gorm:"not null" json:"name"`

	// Latitude — широта точки.
	Latitude float64 `gorm:"not null" json:"latitude"`

	// Longitude — долгота точки.
	Longitude float64 `gorm:"not null" json:"longitude"`

	// Radius — радиус зоны (в метрах), в пределах которого считается, что пользователь находится на чекпоинте.
	Radius float64 `gorm:"not null" json:"radius"`

	// CreatedAt — время создания записи.
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt — время последнего обновления записи.
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
