package service

import (
	"locator/dao"
	"locator/models"
	"sort"
	"time"
)

// TravelSegmentService строит интервалы перемещения вне всех чекпоинтов по GPS-точкам.
type TravelSegmentService struct {
	LocationDAO       *dao.LocationDAO
	CheckpointService *CheckpointService
}

func NewTravelSegmentService(locationDAO *dao.LocationDAO, checkpointService *CheckpointService) *TravelSegmentService {
	return &TravelSegmentService{
		LocationDAO:       locationDAO,
		CheckpointService: checkpointService,
	}
}

// GetOutsideSegments возвращает участки, когда пользователь не находился ни в одном чекпоинте.
func (s *TravelSegmentService) GetOutsideSegments(userID int, from, to time.Time) ([]models.Visit, error) {
	locations, err := s.LocationDAO.GetLocationsByUserBetween(userID, from, to)
	if err != nil {
		return nil, err
	}
	if len(locations) == 0 {
		return nil, nil
	}

	checkpoints, err := s.CheckpointService.GetCheckpoints()
	if err != nil {
		return nil, err
	}

	return buildOutsideSegments(userID, locations, checkpoints, geofenceMinVisitSeconds()), nil
}

func buildOutsideSegments(
	userID int,
	locations []models.Location,
	checkpoints []models.Checkpoint,
	minDurationSec int,
) []models.Visit {
	var segments []models.Visit
	var segmentStart *time.Time
	var lastOutside time.Time
	segmentIndex := 0

	flush := func() {
		if segmentStart == nil {
			return
		}
		end := lastOutside
		duration := int(end.Sub(*segmentStart).Seconds())
		if duration < minDurationSec {
			segmentStart = nil
			return
		}
		endCopy := end
		segments = append(segments, models.Visit{
			ID:           int64(-(segmentIndex + 1)),
			UserID:       userID,
			CheckpointID: 0,
			StartAt:      segmentStart.UTC(),
			EndAt:        &endCopy,
			Duration:     duration,
			Kind:         "outside",
		})
		segmentIndex++
		segmentStart = nil
	}

	for _, loc := range locations {
		inside := false
		if len(checkpoints) > 0 {
			inside = isInsideAnyCheckpoint(loc.Latitude, loc.Longitude, checkpoints)
		}

		if !inside {
			t := loc.CreatedAt.UTC()
			if segmentStart == nil {
				segmentStart = &t
			}
			lastOutside = t
			continue
		}
		flush()
	}
	flush()

	return segments
}

func isInsideAnyCheckpoint(lat, lon float64, checkpoints []models.Checkpoint) bool {
	for i := range checkpoints {
		if haversineDistance(lat, lon, checkpoints[i].Latitude, checkpoints[i].Longitude) <= checkpoints[i].Radius {
			return true
		}
	}
	return false
}

func mergeVisitsSorted(checkpointVisits, outsideVisits []models.Visit) []models.Visit {
	merged := make([]models.Visit, 0, len(checkpointVisits)+len(outsideVisits))
	merged = append(merged, checkpointVisits...)
	merged = append(merged, outsideVisits...)
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].StartAt.After(merged[j].StartAt)
	})
	return merged
}
