package service

import (
	"testing"
	"time"

	"locator/models"
)

func TestBackfillBurstSpread(t *testing.T) {
	svc := &LocationService{}
	interval := 5 * time.Minute
	burstGap := 30 * time.Second
	base := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)

	locs := []models.Location{
		{ID: 1, UserID: 1, CreatedAt: base},
		{ID: 2, UserID: 1, CreatedAt: base.Add(200 * time.Millisecond)},
		{ID: 3, UserID: 1, CreatedAt: base.Add(400 * time.Millisecond)},
	}

	type assignment struct {
		id int
		at time.Time
	}
	var pending []assignment
	flushBurst := func(batch []models.Location) {
		anchor := batch[0].CreatedAt.UTC()
		for i, loc := range batch {
			offset := time.Duration(len(batch)-1-i) * interval)
			pending = append(pending, assignment{id: loc.ID, at: anchor.Add(-offset)})
		}
	}
	var batch []models.Location
	for _, loc := range locs {
		if len(batch) > 0 && loc.CreatedAt.Sub(batch[len(batch)-1].CreatedAt) > burstGap {
			flushBurst(batch)
			batch = batch[:0]
		}
		batch = append(batch, loc)
	}
	flushBurst(batch)

	if len(pending) != 3 {
		t.Fatalf("expected 3 assignments, got %d", len(pending))
	}
	if !pending[0].at.Equal(base.Add(-10 * time.Minute)) {
		t.Fatalf("first captured_at: %v", pending[0].at)
	}
	if !pending[2].at.Equal(base) {
		t.Fatalf("last captured_at: %v", pending[2].at)
	}
	_ = svc
}

func TestResolveCapturedAt_timestampMs(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Minsk")
	if err != nil {
		t.Fatal(err)
	}
	svc := &LocationService{minskLocation: loc}
	ms := time.Date(2026, 7, 1, 8, 30, 0, 0, loc).UnixMilli()
	got, err := svc.ResolveCapturedAt("", ms)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.In(loc).Hour() != 8 {
		t.Fatalf("got %v", got)
	}
}
