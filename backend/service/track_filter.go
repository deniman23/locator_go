package service

import (
	"locator/models"
	"math"
	"time"
)

const (
	trackMaxSpeedMPS      = 25.0 // ~90 км/ч
	trackBatchWindow      = 10 * time.Second
	trackBatchMaxJumpM    = 250.0
	trackShortGap         = 2 * time.Minute
	trackShortGapMaxJumpM = 500.0
	// После backfill офлайн-очереди точки разнесены по 5 мин — иначе 13 км за 10 мин проходят как «пешком».
	trackAbsoluteMaxJumpM      = 1500.0
	trackAbsoluteMaxJumpWindow = 45 * time.Minute
)

// IsTrackOutlierFromPrev — точка невозможна относительно предыдущей принятой.
func IsTrackOutlierFromPrev(prev, curr models.Location) bool {
	dist := haversineDistanceM(prev.Latitude, prev.Longitude, curr.Latitude, curr.Longitude)
	prevAt := prev.EffectiveAt()
	currAt := curr.EffectiveAt()
	if currAt.IsZero() || prevAt.IsZero() {
		return false
	}
	dt := currAt.Sub(prevAt)
	if dt < 0 {
		return true
	}
	if dt <= trackBatchWindow {
		return dist > trackBatchMaxJumpM
	}
	if dt > 0 && dt <= trackAbsoluteMaxJumpWindow && dist > trackAbsoluteMaxJumpM {
		return true
	}
	if dt <= trackShortGap {
		return dist > trackShortGapMaxJumpM
	}
	if dt > 0 {
		speed := dist / dt.Seconds()
		return speed > trackMaxSpeedMPS
	}
	return dist > trackBatchMaxJumpM
}

// FilterTrackOutliers оставляет точки, образующие физически возможный трек.
func FilterTrackOutliers(locs []models.Location) []models.Location {
	if len(locs) <= 1 {
		return locs
	}
	out := make([]models.Location, 0, len(locs))
	out = append(out, locs[0])
	for i := 1; i < len(locs); i++ {
		curr := locs[i]
		prev := out[len(out)-1]
		baseline := trackOutlierBaseline(out)
		if IsTrackOutlierFromPrev(prev, curr) {
			if baseline == nil ||
				haversineDistanceM(baseline.Latitude, baseline.Longitude, curr.Latitude, curr.Longitude) > trackBatchMaxJumpM {
				continue
			}
		} else if baseline != nil && IsTrackOutlierFromPrev(*baseline, curr) {
			continue
		}
		out = append(out, curr)
	}
	return out
}

func trackOutlierBaseline(kept []models.Location) *models.Location {
	if len(kept) == 0 {
		return nil
	}
	idx := len(kept) - 1
	prev := kept[idx]
	for n := 0; n < 8; n++ {
		if idx <= 0 {
			break
		}
		grand := kept[idx-1]
		if IsTrackOutlierFromPrev(grand, prev) {
			prev = grand
			idx--
			continue
		}
		break
	}
	return &prev
}

// SplitTrackForRoadMatch делит трек на непрерывные отрезки без GPS-телепортов.
func SplitTrackForRoadMatch(locs []models.Location) [][]models.Location {
	if len(locs) < 2 {
		return nil
	}
	var segments [][]models.Location
	current := []models.Location{locs[0]}
	for i := 1; i < len(locs); i++ {
		prev := current[len(current)-1]
		curr := locs[i]
		if IsTrackOutlierFromPrev(prev, curr) {
			if len(current) >= 2 {
				segments = append(segments, current)
			}
			current = []models.Location{curr}
			continue
		}
		current = append(current, curr)
	}
	if len(current) >= 2 {
		segments = append(segments, current)
	}
	return segments
}

func haversineDistanceM(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000
	rLat1 := lat1 * math.Pi / 180
	rLat2 := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(rLat1)*math.Cos(rLat2)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
