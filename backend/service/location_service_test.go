package service

import (
	"testing"
	"time"

	"locator/models"
)

func TestSortLocationsByCreatedAt(t *testing.T) {
	t1 := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	t0 := time.Date(2026, 1, 2, 9, 0, 0, 0, time.UTC)
	locs := []models.Location{{ID: 2, CreatedAt: t1}, {ID: 1, CreatedAt: t0}}
	sortLocationsByCreatedAt(locs)
	if locs[0].ID != 1 || locs[1].ID != 2 {
		t.Fatalf("unexpected order: %#v", locs)
	}
}

func TestParseLocationRange_minskWallClock(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Minsk")
	if err != nil {
		t.Fatal(err)
	}
	svc := &LocationService{minskLocation: loc}
	from, to, err := svc.parseLocationRange("2026-04-11T10:00", "2026-04-11T11:00")
	if err != nil {
		t.Fatal(err)
	}
	if from.In(loc).Hour() != 10 || to.In(loc).Hour() != 11 || to.In(loc).Second() != 59 {
		t.Fatalf("expected 10:00–11:00:59 in Minsk, got %v %v", from.In(loc), to.In(loc))
	}
}

func TestLocationRangeForDB_minskEveningInUTC(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Minsk")
	if err != nil {
		t.Fatal(err)
	}
	svc := &LocationService{minskLocation: loc}
	from, to, err := svc.locationRangeForDB("2026-05-15T22:00", "2026-05-15T23:59")
	if err != nil {
		t.Fatal(err)
	}
	// 22:00 Minsk = 19:00 UTC; 23:59:59.999 Minsk = 20:59:59.999 UTC
	wantFrom := time.Date(2026, 5, 15, 19, 0, 0, 0, time.UTC)
	wantTo := time.Date(2026, 5, 15, 20, 59, 59, 999000000, time.UTC)
	if !from.Equal(wantFrom) || !to.Equal(wantTo) {
		t.Fatalf("got %v — %v, want %v — %v", from, to, wantFrom, wantTo)
	}
	// 23:53 UTC = 02:53 Minsk 16-го — вне вечернего окна 15-го
	point := time.Date(2026, 5, 15, 23, 53, 42, 0, time.UTC)
	if !point.After(to) {
		t.Fatalf("23:53 UTC should be after %v", to)
	}
}

func TestFilterSignificantLocations_clusterWithin100m(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Minsk")
	if err != nil {
		t.Fatal(err)
	}
	svc := &LocationService{minskLocation: loc}
	base := time.Date(2026, 4, 11, 10, 0, 0, 0, loc)
	locs := []models.Location{
		{UserID: 1, Latitude: 53.9, Longitude: 27.57, CreatedAt: base},
		{UserID: 1, Latitude: 53.90001, Longitude: 27.57001, CreatedAt: base.Add(5 * time.Minute)},
		{UserID: 1, Latitude: 53.90002, Longitude: 27.57002, CreatedAt: base.Add(16 * time.Minute)},
	}
	out := svc.filterSignificantLocations(locs)
	if len(out) != 1 {
		t.Fatalf("expected 1 representative cluster point, got %d", len(out))
	}
}
