package service

import (
	"locator/models"
	"math"
	"sort"
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
	// Типичный periodic (~5 мин): >1500 м — подозрительно (островки ловят «туда-обратно»).
	trackPeriodicMinGap      = 3*time.Minute + 30*time.Second
	trackPeriodicMaxGap      = 9 * time.Minute
	trackPeriodicMaxJumpM    = 1500.0
	islandReturnRadiusM      = 130.0
	islandMinJumpM           = 200.0
	maxIslandSpan            = 30 * time.Minute
)

func sortLocationsByTrackSort(locs []models.Location) []models.Location {
	out := make([]models.Location, len(locs))
	copy(out, locs)
	sort.Slice(out, func(i, j int) bool {
		ti := out[i].TrackSortAt()
		tj := out[j].TrackSortAt()
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// IsTrackOutlierFromPrev — точка невозможна относительно предыдущей принятой.
func IsTrackOutlierFromPrev(prev, curr models.Location) bool {
	dist := haversineDistanceM(prev.Latitude, prev.Longitude, curr.Latitude, curr.Longitude)
	prevAt := prev.TrackSortAt()
	currAt := curr.TrackSortAt()
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
	if dt >= trackPeriodicMinGap && dt <= trackPeriodicMaxGap && dist > trackPeriodicMaxJumpM {
		return true
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
	locs = sortLocationsByTrackSort(locs)
	locs = FilterGpsIslands(locs)
	out := make([]models.Location, 0, len(locs))
	out = append(out, locs[0])
	for i := 1; i < len(locs); i++ {
		curr := locs[i]
		prev := out[len(out)-1]
		baseline := trackOutlierBaseline(out)
		if IsTrackOutlierFromPrev(prev, curr) {
			returningToBaseline := baseline != nil &&
				haversineDistanceM(baseline.Latitude, baseline.Longitude, curr.Latitude, curr.Longitude) <= islandReturnRadiusM
			if !returningToBaseline &&
				(baseline == nil ||
					haversineDistanceM(baseline.Latitude, baseline.Longitude, curr.Latitude, curr.Longitude) > trackBatchMaxJumpM) {
				continue
			}
		} else if baseline != nil && IsTrackOutlierFromPrev(*baseline, curr) {
			continue
		}
		out = append(out, curr)
	}
	return out
}

func isSandwichOutlier(anchor, mid, next models.Location) bool {
	dAM := haversineDistanceM(anchor.Latitude, anchor.Longitude, mid.Latitude, mid.Longitude)
	dMN := haversineDistanceM(mid.Latitude, mid.Longitude, next.Latitude, next.Longitude)
	dAN := haversineDistanceM(anchor.Latitude, anchor.Longitude, next.Latitude, next.Longitude)
	if dAM < islandMinJumpM || dMN < islandMinJumpM {
		return false
	}
	return dAN <= islandReturnRadiusM
}

// FilterGpsIslands убирает короткие выбросы «туда-обратно» от якорной точки.
func FilterGpsIslands(locs []models.Location) []models.Location {
	if len(locs) < 3 {
		return locs
	}
	out := []models.Location{locs[0]}
	i := 1
	for i < len(locs) {
		anchor := out[len(out)-1]
		if i+1 < len(locs) && isSandwichOutlier(anchor, locs[i], locs[i+1]) {
			i++
			continue
		}
		j := i
		for j < len(locs) {
			d := haversineDistanceM(anchor.Latitude, anchor.Longitude, locs[j].Latitude, locs[j].Longitude)
			if d <= islandReturnRadiusM {
				break
			}
			j++
		}
		if j > i && j < len(locs) {
			span := locs[j].TrackSortAt().Sub(locs[i].TrackSortAt())
			allFar := span <= maxIslandSpan
			if allFar {
				for k := i; k < j; k++ {
					if haversineDistanceM(anchor.Latitude, anchor.Longitude, locs[k].Latitude, locs[k].Longitude) < islandMinJumpM {
						allFar = false
						break
					}
				}
			}
			if allFar {
				i = j
				continue
			}
		}
		out = append(out, locs[i])
		i++
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
