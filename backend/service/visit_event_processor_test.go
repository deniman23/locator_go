package service

import (
	"testing"
	"time"

	"locator/models"
)

func TestGeofenceFarOutside(t *testing.T) {
	radius := 100.0
	buffer := geofenceExitBufferMeters()
	far := geofenceFarExitMeters()
	threshold := radius + buffer + far

	if geofenceFarOutside(threshold, radius, true) {
		t.Fatal("at threshold should not be far outside")
	}
	if !geofenceFarOutside(threshold+1, radius, true) {
		t.Fatal("beyond threshold should be far outside")
	}
	if geofenceFarOutside(threshold+1000, radius, false) {
		t.Fatal("no active visit — never far outside")
	}
}

func TestResolveVisitEndAt_capsAfterStaleGap(t *testing.T) {
	cp := models.Checkpoint{
		ID:        1,
		Latitude:  53.92684,
		Longitude: 27.695144,
		Radius:    100,
	}
	start := time.Date(2026, 7, 2, 10, 58, 13, 0, time.UTC)
	lastInside := time.Date(2026, 7, 2, 13, 10, 26, 0, time.UTC)
	eventNow := time.Date(2026, 7, 2, 16, 15, 13, 0, time.UTC)

	vep := &VisitEventProcessor{
		LocationDAO: &fakeLocationDAO{locations: []models.Location{
			{UserID: 1, Latitude: 53.92684, Longitude: 27.695144, CapturedAt: &lastInside},
		}},
	}
	visit := &models.Visit{ID: 10, UserID: 1, CheckpointID: 1, StartAt: start}

	endAt := vep.resolveVisitEndAt(1, cp, visit, eventNow, true)
	want := lastInside.Add(time.Duration(geofenceExitGraceSeconds()) * time.Second)
	if !endAt.Equal(want) {
		t.Fatalf("endAt=%s want=%s", endAt, want)
	}
}

func TestResolveVisitEndAt_recentBoundaryUsesEventNow(t *testing.T) {
	cp := models.Checkpoint{
		ID:        1,
		Latitude:  53.92684,
		Longitude: 27.695144,
		Radius:    100,
	}
	start := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	lastInside := time.Date(2026, 7, 2, 10, 5, 0, 0, time.UTC)
	eventNow := time.Date(2026, 7, 2, 10, 6, 0, 0, time.UTC)

	vep := &VisitEventProcessor{
		LocationDAO: &fakeLocationDAO{locations: []models.Location{
			{UserID: 1, Latitude: 53.92684, Longitude: 27.695144, CapturedAt: &lastInside},
		}},
	}
	visit := &models.Visit{ID: 1, UserID: 1, CheckpointID: 1, StartAt: start}

	endAt := vep.resolveVisitEndAt(1, cp, visit, eventNow, false)
	if !endAt.Equal(eventNow) {
		t.Fatalf("endAt=%s want eventNow=%s", endAt, eventNow)
	}
}

type fakeLocationDAO struct {
	locations []models.Location
}

func (f *fakeLocationDAO) GetLocationsByUserBetween(userID int, from, to time.Time) ([]models.Location, error) {
	return f.locations, nil
}
