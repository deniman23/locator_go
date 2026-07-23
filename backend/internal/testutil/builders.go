// Package testutil provides shared fixtures for unit and integration tests.
package testutil

import (
	"time"

	"locator/models"

	"golang.org/x/crypto/bcrypt"
)

// FixedUTC returns a deterministic UTC timestamp for fixtures.
func FixedUTC(year int, month time.Month, day, hour, min, sec int) time.Time {
	return time.Date(year, month, day, hour, min, sec, 0, time.UTC)
}

// Location builds a location fixture.
func Location(id, userID int, lat, lon float64, at time.Time) models.Location {
	cap := at
	return models.Location{
		ID:         id,
		UserID:     userID,
		Latitude:   lat,
		Longitude:  lon,
		CapturedAt: &cap,
		CreatedAt:  at,
		UpdatedAt:  at,
		Source:     models.LocationSourcePeriodic,
	}
}

// Checkpoint builds a checkpoint fixture.
func Checkpoint(id int, name string, lat, lon, radius float64) models.Checkpoint {
	now := FixedUTC(2026, 7, 1, 12, 0, 0)
	return models.Checkpoint{
		ID:        id,
		Name:      name,
		Latitude:  lat,
		Longitude: lon,
		Radius:    radius,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// UserWithAPIKey hashes plainKey with bcrypt and returns a user ready for auth tests.
func UserWithAPIKey(id int, name, plainKey string, isAdmin bool) (models.User, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plainKey), bcrypt.MinCost)
	if err != nil {
		return models.User{}, err
	}
	return models.User{
		ID:      id,
		Name:    name,
		ApiKey:  string(hashed),
		IsAdmin: isAdmin,
	}, nil
}

// VisitActive builds an open visit.
func VisitActive(id int64, userID, checkpointID int, start time.Time) models.Visit {
	return models.Visit{
		ID:           id,
		UserID:       userID,
		CheckpointID: checkpointID,
		StartAt:      start,
	}
}
