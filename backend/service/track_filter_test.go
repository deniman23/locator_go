package service

import (
	"testing"
	"time"

	"locator/models"
)

func TestFilterTrackOutliers_offlineBatchJumps(t *testing.T) {
	base := time.Date(2026, 7, 1, 8, 0, 0, 0, time.UTC)
	received := time.Date(2026, 7, 1, 9, 1, 7, 0, time.UTC)
	home := models.Location{
		ID: 1, UserID: 1, Latitude: 53.88586, Longitude: 27.51026,
		CapturedAt: &base, CreatedAt: received,
	}
	jumpAt := base.Add(5 * time.Minute)
	jump := models.Location{
		ID: 2, UserID: 1, Latitude: 53.92684, Longitude: 27.69516,
		CapturedAt: &jumpAt, CreatedAt: received.Add(130 * time.Millisecond),
	}
	backAt := base.Add(10 * time.Minute)
	back := models.Location{
		ID: 3, UserID: 1, Latitude: 53.88585, Longitude: 27.51024,
		CapturedAt: &backAt, CreatedAt: received.Add(260 * time.Millisecond),
	}

	out := FilterTrackOutliers([]models.Location{home, jump, back})
	if len(out) != 2 {
		t.Fatalf("expected 2 points, got %d: %#v", len(out), out)
	}
	if out[1].ID != 3 {
		t.Fatalf("expected point 3 kept, got %#v", out[1])
	}
}

func TestFilterTrackOutliers_normalDrive(t *testing.T) {
	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	locs := []models.Location{
		{ID: 1, Latitude: 53.9, Longitude: 27.5, CreatedAt: base},
		{ID: 2, Latitude: 53.905, Longitude: 27.51, CreatedAt: base.Add(5 * time.Minute)},
	}
	out := FilterTrackOutliers(locs)
	if len(out) != 2 {
		t.Fatalf("expected both points, got %d", len(out))
	}
}

func TestIsTrackOutlierFromPrev_batchFlush(t *testing.T) {
	prev := models.Location{Latitude: 53.885, Longitude: 27.510, CreatedAt: time.Now()}
	curr := models.Location{Latitude: 53.926, Longitude: 27.695, CreatedAt: prev.CreatedAt.Add(200 * time.Millisecond)}
	if !IsTrackOutlierFromPrev(prev, curr) {
		t.Fatal("expected 13km in 200ms to be outlier")
	}
}
