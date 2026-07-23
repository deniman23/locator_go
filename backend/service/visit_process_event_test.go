package service

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"locator/internal/testutil"
	"locator/models"

	"gorm.io/gorm"
)

func TestProcessEvent_onDemandStartsVisitInside(t *testing.T) {
	_ = os.Setenv("GEOFENCE_ENTER_GRACE_SECONDS", "30")
	t.Cleanup(func() { _ = os.Unsetenv("GEOFENCE_ENTER_GRACE_SECONDS") })

	cp := testutil.Checkpoint(1, "office", 53.92684, 27.695144, 100)
	visitRepo := newFakeVisitRepo()
	vs := &VisitService{DAO: visitRepo}
	cs := &CheckpointService{DAO: &checkpointDAOAdapter{items: []models.Checkpoint{cp}}}
	vep := NewVisitEventProcessor(cs, vs, &fakeLocationDAO{})

	event := models.LocationEvent{
		UserID:     1,
		Latitude:   53.92684,
		Longitude:  27.695144,
		OccurredAt: testutil.FixedUTC(2026, 7, 1, 12, 0, 0),
		Source:     models.LocationSourceOnDemand,
	}
	body, _ := json.Marshal(event)
	if err := vep.ProcessEvent(body); err != nil {
		t.Fatal(err)
	}

	active, err := visitRepo.GetActiveVisit(1, 1)
	if err != nil {
		t.Fatalf("expected active visit: %v", err)
	}
	if active.CheckpointID != 1 {
		t.Fatalf("visit=%+v", active)
	}
}

func TestProcessEvent_exitAbandonsShortVisit(t *testing.T) {
	_ = os.Setenv("GEOFENCE_MIN_VISIT_SECONDS", "120")
	_ = os.Setenv("GEOFENCE_EXIT_GRACE_SECONDS", "0")
	_ = os.Setenv("GEOFENCE_FAR_EXIT_METERS", "50")
	t.Cleanup(func() {
		_ = os.Unsetenv("GEOFENCE_MIN_VISIT_SECONDS")
		_ = os.Unsetenv("GEOFENCE_EXIT_GRACE_SECONDS")
		_ = os.Unsetenv("GEOFENCE_FAR_EXIT_METERS")
	})

	cp := testutil.Checkpoint(1, "office", 53.92684, 27.695144, 100)
	visitRepo := newFakeVisitRepo()
	start := testutil.FixedUTC(2026, 7, 1, 12, 0, 0)
	if err := visitRepo.Create(&models.Visit{UserID: 1, CheckpointID: 1, StartAt: start}); err != nil {
		t.Fatal(err)
	}

	vs := &VisitService{DAO: visitRepo}
	vep := NewVisitEventProcessor(&CheckpointService{DAO: &checkpointDAOAdapter{items: []models.Checkpoint{cp}}}, vs, &fakeLocationDAO{})

	// Far outside shortly after start → abandon
	event := models.LocationEvent{
		UserID:     1,
		Latitude:   53.94,
		Longitude:  27.72,
		OccurredAt: start.Add(30 * time.Second),
		Source:     models.LocationSourcePeriodic,
	}
	body, _ := json.Marshal(event)
	if err := vep.ProcessEvent(body); err != nil {
		t.Fatal(err)
	}
	_, err := visitRepo.GetActiveVisit(1, 1)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected abandoned visit, err=%v", err)
	}
}

// checkpointDAOAdapter satisfies *dao.CheckpointDAO method set used by CheckpointService
// by embedding into a type that CheckpointService can hold — we change CheckpointService.DAO to interface.
type checkpointDAOAdapter struct {
	items []models.Checkpoint
}

func (a *checkpointDAOAdapter) Create(cp *models.Checkpoint) error {
	a.items = append(a.items, *cp)
	return nil
}
func (a *checkpointDAOAdapter) Update(cp *models.Checkpoint) error { return nil }
func (a *checkpointDAOAdapter) GetByID(id int) (*models.Checkpoint, error) {
	for _, cp := range a.items {
		if cp.ID == id {
			c := cp
			return &c, nil
		}
	}
	return nil, errors.New("not found")
}
func (a *checkpointDAOAdapter) GetAll() ([]models.Checkpoint, error) {
	return append([]models.Checkpoint(nil), a.items...), nil
}
