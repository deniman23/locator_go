package service

import (
	"errors"
	"net/url"
	"testing"
	"time"

	"locator/internal/testutil"
	"locator/models"

	"gorm.io/gorm"
)

type fakeVisitRepo struct {
	visits map[int64]*models.Visit
	nextID int64
}

func newFakeVisitRepo() *fakeVisitRepo {
	return &fakeVisitRepo{visits: make(map[int64]*models.Visit), nextID: 1}
}

func (f *fakeVisitRepo) Create(visit *models.Visit) error {
	if visit.ID == 0 {
		visit.ID = f.nextID
		f.nextID++
	}
	cp := *visit
	f.visits[visit.ID] = &cp
	return nil
}

func (f *fakeVisitRepo) Update(visit *models.Visit) error {
	cp := *visit
	f.visits[visit.ID] = &cp
	return nil
}

func (f *fakeVisitRepo) Delete(id int64) error {
	delete(f.visits, id)
	return nil
}

func (f *fakeVisitRepo) GetActiveVisit(userID int, checkpointID int) (*models.Visit, error) {
	for _, v := range f.visits {
		if v.UserID == userID && v.CheckpointID == checkpointID && v.EndAt == nil {
			cp := *v
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeVisitRepo) GetVisits(filters map[string]interface{}, activeOnly bool, rangeFrom, rangeTo *time.Time) ([]models.Visit, error) {
	var out []models.Visit
	for _, v := range f.visits {
		if uid, ok := filters["user_id"].(int); ok && v.UserID != uid {
			continue
		}
		if cid, ok := filters["checkpoint_id"].(int); ok && v.CheckpointID != cid {
			continue
		}
		if activeOnly && v.EndAt != nil {
			continue
		}
		out = append(out, *v)
	}
	return out, nil
}

func TestStartVisitAt_andEndVisitAt(t *testing.T) {
	repo := newFakeVisitRepo()
	vs := &VisitService{DAO: repo}
	start := testutil.FixedUTC(2026, 7, 1, 10, 0, 0)
	visit, err := vs.StartVisitAt(1, 5, start)
	if err != nil {
		t.Fatal(err)
	}
	if visit.ID == 0 || visit.EndAt != nil {
		t.Fatalf("unexpected visit %+v", visit)
	}

	end := start.Add(15 * time.Minute)
	if err := vs.EndVisitAt(visit, end); err != nil {
		t.Fatal(err)
	}
	if visit.Duration != 15*60 {
		t.Fatalf("duration=%d", visit.Duration)
	}
	if visit.EndAt == nil || !visit.EndAt.Equal(end) {
		t.Fatalf("endAt=%v", visit.EndAt)
	}
}

func TestEndVisitAt_clampsBeforeStart(t *testing.T) {
	repo := newFakeVisitRepo()
	vs := &VisitService{DAO: repo}
	start := testutil.FixedUTC(2026, 7, 1, 10, 0, 0)
	visit, err := vs.StartVisitAt(1, 5, start)
	if err != nil {
		t.Fatal(err)
	}
	before := start.Add(-time.Minute)
	if err := vs.EndVisitAt(visit, before); err != nil {
		t.Fatal(err)
	}
	if visit.Duration != 0 {
		t.Fatalf("duration=%d want 0", visit.Duration)
	}
	if !visit.EndAt.Equal(start.UTC()) {
		t.Fatalf("endAt=%v want start", visit.EndAt)
	}
}

func TestAbandonVisit(t *testing.T) {
	repo := newFakeVisitRepo()
	vs := &VisitService{DAO: repo}
	visit, err := vs.StartVisitAt(1, 5, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := vs.AbandonVisit(visit); err != nil {
		t.Fatal(err)
	}
	_, err = repo.GetActiveVisit(1, 5)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestGetVisitsByFilters_requiresBothFromTo(t *testing.T) {
	vs := &VisitService{DAO: newFakeVisitRepo()}
	_, err := vs.GetVisitsByFilters(url.Values{"from": {"2026-07-01T10:00"}})
	if err == nil {
		t.Fatal("expected error when only from is set")
	}
}

func TestGetVisitsByFilters_userAndActive(t *testing.T) {
	repo := newFakeVisitRepo()
	vs := &VisitService{DAO: repo}
	_, _ = vs.StartVisitAt(7, 1, testutil.FixedUTC(2026, 7, 1, 9, 0, 0))

	got, err := vs.GetVisitsByFilters(url.Values{
		"user_id": {"7"},
		"active":  {"true"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d visits", len(got))
	}
}
