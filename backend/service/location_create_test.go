package service

import (
	"errors"
	"testing"
	"time"

	"locator/internal/testutil"
	"locator/models"
)

type fakeLocationRepo struct {
	byUser map[int][]models.Location
	nextID int
}

func newFakeLocationRepo(locs ...models.Location) *fakeLocationRepo {
	f := &fakeLocationRepo{byUser: make(map[int][]models.Location), nextID: 1}
	for _, loc := range locs {
		if loc.ID == 0 {
			loc.ID = f.nextID
			f.nextID++
		}
		f.byUser[loc.UserID] = append(f.byUser[loc.UserID], loc)
	}
	return f
}

func (f *fakeLocationRepo) GetByUserID(userID int) (*models.Location, error) {
	locs := f.byUser[userID]
	if len(locs) == 0 {
		return nil, errors.New("not found")
	}
	best := locs[0]
	for _, loc := range locs[1:] {
		if loc.EffectiveAt().After(best.EffectiveAt()) {
			best = loc
		}
	}
	cp := best
	return &cp, nil
}

func (f *fakeLocationRepo) GetPreviousByEffectiveTime(userID int, before time.Time) (*models.Location, error) {
	var best *models.Location
	for i := range f.byUser[userID] {
		loc := f.byUser[userID][i]
		at := loc.EffectiveAt()
		if !at.Before(before) {
			continue
		}
		if best == nil || at.After(best.EffectiveAt()) {
			cp := loc
			best = &cp
		}
	}
	if best == nil {
		return nil, errors.New("not found")
	}
	return best, nil
}

func (f *fakeLocationRepo) UserExists(userID int) (bool, error) {
	_, ok := f.byUser[userID]
	return ok, nil
}

func (f *fakeLocationRepo) Create(loc *models.Location) error {
	if loc.ID == 0 {
		loc.ID = f.nextID
		f.nextID++
	}
	f.byUser[loc.UserID] = append(f.byUser[loc.UserID], *loc)
	return nil
}

func (f *fakeLocationRepo) GetAll() ([]models.Location, error) {
	var out []models.Location
	for _, locs := range f.byUser {
		out = append(out, locs...)
	}
	return out, nil
}

func (f *fakeLocationRepo) GetLocationsBetween(from, to time.Time) ([]models.Location, error) {
	all, _ := f.GetAll()
	var out []models.Location
	for _, loc := range all {
		at := loc.EffectiveAt()
		if !at.Before(from) && !at.After(to) {
			out = append(out, loc)
		}
	}
	return out, nil
}

func (f *fakeLocationRepo) ListUserIDsWithoutCapturedAt() ([]int, error) {
	return nil, nil
}

func (f *fakeLocationRepo) GetWithoutCapturedAtByUser(userID int) ([]models.Location, error) {
	return nil, nil
}

func (f *fakeLocationRepo) UpdateCapturedAt(id int, capturedAt time.Time) error {
	return nil
}

func newTestLocationService(repo *fakeLocationRepo) *LocationService {
	loc, _ := time.LoadLocation("Europe/Minsk")
	return &LocationService{DAO: repo, minskLocation: loc}
}

func TestCreateLocation_periodicAlwaysPersists(t *testing.T) {
	homeAt := testutil.FixedUTC(2026, 7, 1, 8, 0, 0)
	home := testutil.Location(1, 1, 53.88586, 27.51026, homeAt)
	repo := newFakeLocationRepo(home)
	svc := newTestLocationService(repo)

	// Far jump would be outlier for non-periodic, but periodic must persist.
	got, reason, err := svc.CreateLocation(
		1, 53.92684, 27.69516, "", models.LocationSourcePeriodic, nil, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if reason != "" || got == nil {
		t.Fatalf("reason=%q loc=%v", reason, got)
	}
	if len(repo.byUser[1]) != 2 {
		t.Fatalf("stored=%d", len(repo.byUser[1]))
	}
}

func TestCreateLocation_skipsGpsOutlier(t *testing.T) {
	homeAt := time.Now().UTC().Add(-2 * time.Minute)
	home := testutil.Location(1, 1, 53.88586, 27.51026, homeAt)
	repo := newFakeLocationRepo(home)
	svc := newTestLocationService(repo)

	jumpAt := homeAt.Add(2 * time.Second) // within batch window → jump > 250m is outlier
	acc := 10.0
	got, reason, err := svc.CreateLocation(
		1, 53.92684, 27.69516, "", models.LocationSourceOnDemand, &jumpAt, &acc,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil || reason != "gps_outlier" {
		t.Fatalf("got=%v reason=%q", got, reason)
	}
}

func TestCreateLocation_persistsNearbyPoint(t *testing.T) {
	homeAt := testutil.FixedUTC(2026, 7, 1, 8, 0, 0)
	home := testutil.Location(1, 1, 53.9, 27.5, homeAt)
	repo := newFakeLocationRepo(home)
	svc := newTestLocationService(repo)

	acc := 15.0
	got, reason, err := svc.CreateLocation(
		1, 53.9005, 27.5005, "", models.LocationSourceOnDemand, nil, &acc,
	)
	if err != nil {
		t.Fatal(err)
	}
	if reason != "" || got == nil {
		t.Fatalf("reason=%q loc=%v", reason, got)
	}
}
