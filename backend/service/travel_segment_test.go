package service

import (
	"testing"
	"time"

	"locator/models"
)

func TestBuildOutsideSegments(t *testing.T) {
	base := time.Date(2026, 5, 17, 20, 0, 0, 0, time.UTC)
	checkpoints := []models.Checkpoint{{
		ID: 1, Name: "дом",
		Latitude: 53.9, Longitude: 27.5, Radius: 100,
	}}

	locations := []models.Location{
		{UserID: 1, Latitude: 53.91, Longitude: 27.51, CreatedAt: base},
		{UserID: 1, Latitude: 53.91, Longitude: 27.51, CreatedAt: base.Add(30 * time.Second)},
		{UserID: 1, Latitude: 53.95, Longitude: 27.55, CreatedAt: base.Add(time.Minute)},
		{UserID: 1, Latitude: 53.95, Longitude: 27.55, CreatedAt: base.Add(2 * time.Minute)},
		{UserID: 1, Latitude: 53.91, Longitude: 27.51, CreatedAt: base.Add(3 * time.Minute)},
	}

	segments := buildOutsideSegments(1, locations, checkpoints, 60)
	if len(segments) != 1 {
		t.Fatalf("expected 1 outside segment, got %d", len(segments))
	}
	if segments[0].Kind != "outside" || segments[0].CheckpointID != 0 {
		t.Fatalf("unexpected segment: %+v", segments[0])
	}
	if segments[0].Duration < 60 {
		t.Fatalf("expected duration >= 60, got %d", segments[0].Duration)
	}
}

func TestBuildOutsideSegments_skipsShort(t *testing.T) {
	base := time.Date(2026, 5, 17, 20, 0, 0, 0, time.UTC)
	locations := []models.Location{
		{UserID: 1, Latitude: 54.0, Longitude: 28.0, CreatedAt: base},
		{UserID: 1, Latitude: 54.0, Longitude: 28.0, CreatedAt: base.Add(5 * time.Second)},
	}
	segments := buildOutsideSegments(1, locations, nil, 60)
	if len(segments) != 0 {
		t.Fatalf("expected no segments for short outside period, got %d", len(segments))
	}
}
