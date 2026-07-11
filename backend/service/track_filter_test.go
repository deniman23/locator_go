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

func TestIsTrackOutlierFromPrev_periodicFiveMinJump(t *testing.T) {
	base := time.Date(2026, 7, 3, 3, 3, 52, 0, time.UTC)
	prev := models.Location{
		Latitude: 53.8850848, Longitude: 27.5088216,
		CapturedAt: &base, CreatedAt: base,
	}
	jumpAt := base.Add(5 * time.Minute)
	curr := models.Location{
		Latitude: 53.87069, Longitude: 27.529505,
		CapturedAt: &jumpAt, CreatedAt: jumpAt,
	}
	if IsTrackOutlierFromPrev(prev, curr) {
		t.Fatal("expected ~2km in 5min drive to be kept (not outlier)")
	}
}

func TestFilterTrackOutliers_gpsIslandReturnHome(t *testing.T) {
	base := time.Date(2026, 7, 3, 3, 3, 52, 0, time.UTC)
	home := func(id int, at time.Time) models.Location {
		return models.Location{
			ID: id, UserID: 1, Latitude: 53.8850848, Longitude: 27.5088216,
			CapturedAt: &at, CreatedAt: at,
		}
	}
	wrong := func(id int, at time.Time) models.Location {
		return models.Location{
			ID: id, UserID: 1, Latitude: 53.88269, Longitude: 27.529505,
			CapturedAt: &at, CreatedAt: at,
		}
	}
	locs := []models.Location{
		home(6001, base),
		wrong(6002, base.Add(5*time.Minute)),
		wrong(6003, base.Add(10*time.Minute)),
		home(6004, base.Add(16*time.Minute)),
	}
	out := FilterTrackOutliers(locs)
	if len(out) != 2 {
		t.Fatalf("expected 2 points (home before/after island), got %d: %#v", len(out), out)
	}
	if out[0].ID != 6001 || out[1].ID != 6004 {
		t.Fatalf("unexpected ids: %#v", out)
	}
}

func TestFilterTrackOutliers_singleSandwichPoint(t *testing.T) {
	base := time.Date(2026, 7, 3, 4, 9, 46, 0, time.UTC)
	home := func(id int, at time.Time, lat, lon float64) models.Location {
		return models.Location{
			ID: id, UserID: 1, Latitude: lat, Longitude: lon,
			CapturedAt: &at, CreatedAt: at,
		}
	}
	locs := []models.Location{
		home(6011, base, 53.885109, 27.5089028),
		home(6012, base.Add(27*time.Minute+24*time.Second), 53.88393, 27.5168083),
		home(6013, base.Add(32*time.Minute+49*time.Second), 53.885083, 27.508913),
	}
	out := FilterTrackOutliers(locs)
	if len(out) != 2 {
		t.Fatalf("expected sandwich #6012 removed, got %d: %#v", len(out), out)
	}
	if out[1].ID != 6013 {
		t.Fatalf("expected 6013 kept, got %#v", out)
	}
}

func TestFilterTrackOutliers_driveKeepsMinutePoints(t *testing.T) {
	base := time.Date(2026, 7, 5, 8, 0, 0, 0, time.UTC)
	locs := []models.Location{
		{ID: 9000, Latitude: 53.885, Longitude: 27.510, CapturedAt: &base, CreatedAt: base},
	}
	lat, lon := 53.885, 27.510
	for i := 1; i <= 20; i++ {
		at := base.Add(time.Duration(i) * time.Minute)
		lat += 0.003
		lon += 0.004
		tCopy := at
		locs = append(locs, models.Location{
			ID: 9000 + i, Latitude: lat, Longitude: lon, CapturedAt: &tCopy, CreatedAt: at,
		})
	}
	out := FilterTrackOutliers(locs)
	if len(out) != len(locs) {
		t.Fatalf("expected all %d drive points kept, got %d", len(locs), len(out))
	}
}

func TestFilterTrackOutliers_staleCapturedAfterOffline(t *testing.T) {
	homeAt := time.Date(2026, 7, 3, 9, 47, 29, 0, time.UTC)
	home := models.Location{
		ID: 6069, UserID: 1, Latitude: 53.885872, Longitude: 27.510088,
		CapturedAt: &homeAt, CreatedAt: homeAt,
	}
	zamokCap := time.Date(2026, 7, 3, 11, 31, 5, 0, time.UTC)
	received := time.Date(2026, 7, 3, 11, 31, 43, 0, time.UTC)
	zamok := models.Location{
		ID: 6070, UserID: 1, Latitude: 53.926598, Longitude: 27.517794,
		CapturedAt: &zamokCap, CreatedAt: received,
	}
	wrongCap := time.Date(2026, 7, 3, 11, 28, 24, 0, time.UTC)
	wrong := models.Location{
		ID: 6071, UserID: 1, Latitude: 53.866451, Longitude: 27.484868,
		Source: models.LocationSourceOnDemand, CapturedAt: &wrongCap, CreatedAt: received,
	}
	okCap := time.Date(2026, 7, 3, 11, 32, 0, 0, time.UTC)
	ok := models.Location{
		ID: 6072, UserID: 1, Latitude: 53.926578, Longitude: 27.517909,
		Source: models.LocationSourceOnDemand, CapturedAt: &okCap, CreatedAt: okCap,
	}

	out := FilterTrackOutliers([]models.Location{home, zamok, wrong, ok})
	if len(out) != 3 {
		t.Fatalf("expected 3 points (wrong #6071 removed), got %d: %#v", len(out), out)
	}
	if out[len(out)-1].ID != 6072 {
		t.Fatalf("expected last point at zamok, got %#v", out[len(out)-1])
	}
	for _, loc := range out {
		if loc.ID == 6071 {
			t.Fatal("stale wrong point #6071 must be filtered")
		}
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
