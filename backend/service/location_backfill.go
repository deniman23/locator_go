package service

import (
	"fmt"
	"locator/models"
	"log"
	"sort"
	"time"
)

const (
	backfillBurstGap       = 30 * time.Second
	backfillDefaultInterval = 5 * time.Minute
)

// BackfillCapturedAtOptions — параметры восстановления captured_at для старых записей.
type BackfillCapturedAtOptions struct {
	UserID          int
	Interval        time.Duration
	DryRun          bool
	BurstGapSeconds int
}

// BackfillCapturedAtResult — итог backfill.
type BackfillCapturedAtResult struct {
	UsersProcessed int `json:"users_processed"`
	Updated        int `json:"updated"`
	Bursts         int `json:"bursts"`
	Skipped        int `json:"skipped"`
}

// BackfillCapturedAt восстанавливает captured_at для точек без него.
// Пакеты с быстрой загрузкой (офлайн-очередь) размазываются с interval назад от момента приёма.
func (svc *LocationService) BackfillCapturedAt(opts BackfillCapturedAtOptions) (*BackfillCapturedAtResult, error) {
	interval := opts.Interval
	if interval <= 0 {
		interval = backfillDefaultInterval
	}
	burstGap := backfillBurstGap
	if opts.BurstGapSeconds > 0 {
		burstGap = time.Duration(opts.BurstGapSeconds) * time.Second
	}

	var userIDs []int
	if opts.UserID > 0 {
		userIDs = []int{opts.UserID}
	} else {
		ids, err := svc.DAO.ListUserIDsWithoutCapturedAt()
		if err != nil {
			return nil, err
		}
		userIDs = ids
	}

	result := &BackfillCapturedAtResult{}
	for _, uid := range userIDs {
		n, bursts, err := svc.backfillCapturedAtForUser(uid, interval, burstGap, opts.DryRun)
		if err != nil {
			return nil, fmt.Errorf("user %d: %w", uid, err)
		}
		if n > 0 || bursts > 0 {
			result.UsersProcessed++
		}
		result.Updated += n
		result.Bursts += bursts
	}
	return result, nil
}

func (svc *LocationService) backfillCapturedAtForUser(
	userID int, interval, burstGap time.Duration, dryRun bool,
) (updated, bursts int, err error) {
	locs, err := svc.DAO.GetWithoutCapturedAtByUser(userID)
	if err != nil {
		return 0, 0, err
	}
	if len(locs) == 0 {
		return 0, 0, nil
	}
	sort.Slice(locs, func(i, j int) bool {
		if locs[i].CreatedAt.Equal(locs[j].CreatedAt) {
			return locs[i].ID < locs[j].ID
		}
		return locs[i].CreatedAt.Before(locs[j].CreatedAt)
	})

	type assignment struct {
		id int
		at time.Time
	}
	var pending []assignment

	flushBurst := func(batch []models.Location) {
		if len(batch) == 0 {
			return
		}
		bursts++
		anchor := batch[0].CreatedAt.UTC()
		for i, loc := range batch {
			var at time.Time
			if len(batch) == 1 {
				at = loc.CreatedAt.UTC()
			} else {
				offset := time.Duration(len(batch)-1-i) * interval
				at = anchor.Add(-offset)
			}
			pending = append(pending, assignment{id: loc.ID, at: at})
		}
	}

	var batch []models.Location
	for _, loc := range locs {
		if len(batch) > 0 {
			prev := batch[len(batch)-1]
			if loc.CreatedAt.Sub(prev.CreatedAt) > burstGap {
				flushBurst(batch)
				batch = batch[:0]
			}
		}
		batch = append(batch, loc)
	}
	flushBurst(batch)

	for _, item := range pending {
		if dryRun {
			log.Printf("[BackfillCapturedAt] dry-run id=%d captured_at=%s", item.id, item.at.Format(time.RFC3339))
			updated++
			continue
		}
		if err := svc.DAO.UpdateCapturedAt(item.id, item.at); err != nil {
			return updated, bursts, err
		}
		updated++
	}
	return updated, bursts, nil
}

// ResolveCapturedAt — captured_at из строки или timestamp (unix ms) с телефона.
func (svc *LocationService) ResolveCapturedAt(capturedAtStr string, timestampMs int64) (*time.Time, error) {
	if capturedAtStr != "" {
		return svc.ParseCapturedAt(capturedAtStr)
	}
	if timestampMs <= 0 {
		return nil, nil
	}
	t := time.UnixMilli(timestampMs).UTC()
	now := time.Now().UTC()
	if t.After(now.Add(capturedAtMaxFutureSkew)) {
		return nil, fmt.Errorf("timestamp не может быть в будущем")
	}
	if t.Before(now.Add(-capturedAtMaxAge)) {
		return nil, fmt.Errorf("timestamp слишком старый (более 90 дней)")
	}
	return &t, nil
}
