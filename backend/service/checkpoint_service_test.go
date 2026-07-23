package service

import (
	"math"
	"testing"

	"locator/internal/testutil"
	"locator/models"
)

func TestDistanceToCheckpoint_samePoint(t *testing.T) {
	svc := &CheckpointService{}
	cp := testutil.Checkpoint(1, "home", 53.9, 27.5, 100)
	d := svc.DistanceToCheckpoint(53.9, 27.5, &cp)
	if d > 0.01 {
		t.Fatalf("distance=%f want ~0", d)
	}
}

func TestDistanceToCheckpoint_knownOffset(t *testing.T) {
	svc := &CheckpointService{}
	cp := testutil.Checkpoint(1, "home", 53.9, 27.5, 200)
	// ~111m north
	d := svc.DistanceToCheckpoint(53.901, 27.5, &cp)
	if d < 90 || d > 130 {
		t.Fatalf("distance=%f expected ~111m", d)
	}
}

func TestIsLocationInCheckpoint(t *testing.T) {
	svc := &CheckpointService{}
	cp := testutil.Checkpoint(1, "office", 53.92684, 27.695144, 100)
	inside := &models.Location{Latitude: 53.92684, Longitude: 27.695144}
	outside := &models.Location{Latitude: 53.93, Longitude: 27.70}

	if !svc.IsLocationInCheckpoint(inside, &cp) {
		t.Fatal("expected inside")
	}
	if svc.IsLocationInCheckpoint(outside, &cp) {
		t.Fatal("expected outside")
	}
}

func TestHaversineDistance_symmetry(t *testing.T) {
	a := haversineDistance(53.9, 27.5, 53.91, 27.51)
	b := haversineDistance(53.91, 27.51, 53.9, 27.5)
	if math.Abs(a-b) > 0.01 {
		t.Fatalf("asymmetric: %f vs %f", a, b)
	}
}
