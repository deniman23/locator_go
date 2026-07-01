package service

import (
	"testing"
	"time"

	"locator/models"
)

func TestParseCapturedAt_rfc3339(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Minsk")
	if err != nil {
		t.Fatal(err)
	}
	svc := &LocationService{minskLocation: loc}
	got, err := svc.ParseCapturedAt("2026-07-01T08:30:00+03:00")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected time")
	}
	if got.In(loc).Hour() != 8 || got.In(loc).Minute() != 30 {
		t.Fatalf("unexpected %v", got.In(loc))
	}
}

func TestParseCapturedAt_minskLocal(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Minsk")
	if err != nil {
		t.Fatal(err)
	}
	svc := &LocationService{minskLocation: loc}
	got, err := svc.ParseCapturedAt("2026-07-01T08:30")
	if err != nil {
		t.Fatal(err)
	}
	if got.In(loc).Hour() != 8 {
		t.Fatalf("got %v", got.In(loc))
	}
}

func TestLocationEffectiveAt_prefersCaptured(t *testing.T) {
	captured := time.Date(2026, 7, 1, 8, 0, 0, 0, time.UTC)
	created := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	loc := &models.Location{CapturedAt: &captured, CreatedAt: created}
	if !loc.EffectiveAt().Equal(captured) {
		t.Fatalf("want captured, got %v", loc.EffectiveAt())
	}
}
