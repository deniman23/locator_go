package models

import "time"

// Location описывает данные о местоположении отдельного объекта (например, сотрудника или ребенка).
type Location struct {
	// ID — уникальный идентификатор записи (числовой, автоинкремент).
	ID int `gorm:"primaryKey;autoIncrement" json:"id"`

	// UserID — идентификатор пользователя (целое число).
	UserID int `gorm:"not null" json:"user_id"`

	// Latitude — широта.
	Latitude float64 `gorm:"not null" json:"latitude"`

	// Longitude — долгота.
	Longitude float64 `gorm:"not null" json:"longitude"`

	// RequestID — ответ на on-demand запрос (опционально).
	RequestID string `gorm:"size:36;index" json:"request_id,omitempty"`

	// Source — periodic | on_demand.
	Source string `gorm:"size:20" json:"source,omitempty"`

	// CapturedAt — момент фиксации GPS на устройстве (офлайн-очередь); если пусто — created_at.
	CapturedAt *time.Time `json:"captured_at,omitempty"`

	// CreatedAt — время приёма записи сервером.
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt — время последнего обновления записи.
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// NewLocation создаёт новую запись о местоположении с заданными координатами для пользователя.
func NewLocation(userID int, lat, lon float64) *Location {
	now := time.Now()
	return &Location{
		ID:        0, // GORM установит значение автоматически при вставке записи.
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

// trackCapturedSkew — если captured_at заметно раньше приёма сервером, для порядка трека берём created_at.
const trackCapturedSkew = 90 * time.Second

// EffectiveAt — время для визитов и геозоны: captured_at или created_at.
func (loc *Location) EffectiveAt() time.Time {
	if loc != nil && loc.CapturedAt != nil && !loc.CapturedAt.IsZero() {
		return loc.CapturedAt.UTC()
	}
	if loc != nil && !loc.CreatedAt.IsZero() {
		return loc.CreatedAt.UTC()
	}
	return time.Time{}
}

// TrackSortAt — время для сортировки трека на карте (офлайн-очередь с устаревшим captured_at).
func (loc *Location) TrackSortAt() time.Time {
	if loc == nil {
		return time.Time{}
	}
	if loc.CapturedAt != nil && !loc.CapturedAt.IsZero() && !loc.CreatedAt.IsZero() {
		if loc.CreatedAt.UTC().Sub(loc.CapturedAt.UTC()) > trackCapturedSkew {
			return loc.CreatedAt.UTC()
		}
	}
	return loc.EffectiveAt()
}

// HasStaleCapturedAt — GPS-fix с устройства заметно старше момента отправки на сервер.
func (loc *Location) HasStaleCapturedAt() bool {
	if loc == nil || loc.CapturedAt == nil || loc.CapturedAt.IsZero() || loc.CreatedAt.IsZero() {
		return false
	}
	return loc.CreatedAt.UTC().Sub(loc.CapturedAt.UTC()) > trackCapturedSkew
}
