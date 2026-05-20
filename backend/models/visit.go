package models

import "time"

// Visit описывает визит пользователя в чекпоинт или участок перемещения вне чекпоинтов.
type Visit struct {
	ID           int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int        `json:"user_id"`                 // Идентификатор пользователя.
	CheckpointID int        `json:"checkpoint_id"`           // Идентификатор чекпоинта; 0 — вне чекпоинтов.
	StartAt      time.Time  `json:"start_at"`                // Время начала визита (входа).
	EndAt        *time.Time `json:"end_at,omitempty"`        // Время окончания визита; если визит активен, это поле будет nil.
	Duration     int        `json:"duration"`                // Длительность визита в секундах (вычисляется при завершении визита).
	Kind         string     `gorm:"-" json:"kind,omitempty"` // "checkpoint" (по умолчанию) или "outside".
}
