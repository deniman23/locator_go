package service

import (
	"testing"

	"locator/models"
)

func TestShouldSkipPoorLocation_poorAccuracy(t *testing.T) {
	acc := 160.0 // выше defaultMaxPeriodicAccuracyM (150)
	skip, reason := ShouldSkipPoorLocation(models.LocationSourcePeriodic, &acc, nil, 53.9, 27.5)
	if !skip || reason != "poor_accuracy" {
		t.Fatalf("want poor_accuracy skip, got skip=%v reason=%q", skip, reason)
	}
}

func TestShouldSkipPoorLocation_stationaryPoorAccuracy(t *testing.T) {
	acc := 65.0
	prev := &models.Location{Latitude: 53.885, Longitude: 27.509}
	skip, reason := ShouldSkipPoorLocation(models.LocationSourcePeriodic, &acc, prev, 53.8851, 27.5091)
	if !skip || reason != "stationary_poor_accuracy" {
		t.Fatalf("want stationary_poor_accuracy, got skip=%v reason=%q", skip, reason)
	}
}

func TestShouldSkipPoorLocation_goodAccuracyAtWork(t *testing.T) {
	acc := 18.0
	prev := &models.Location{Latitude: 53.885, Longitude: 27.509}
	skip, _ := ShouldSkipPoorLocation(models.LocationSourcePeriodic, &acc, prev, 53.8851, 27.5091)
	if skip {
		t.Fatal("expected good accuracy stationary point to pass")
	}
}

func TestShouldSkipPoorLocation_onDemandLenient(t *testing.T) {
	acc := 100.0
	skip, _ := ShouldSkipPoorLocation(models.LocationSourceOnDemand, &acc, nil, 53.9, 27.5)
	if skip {
		t.Fatal("on_demand should allow 100m accuracy")
	}
}
