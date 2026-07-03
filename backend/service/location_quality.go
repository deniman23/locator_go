package service

import (
	"os"
	"strconv"
	"strings"

	"locator/models"
)

const (
	defaultMaxPeriodicAccuracyM = 80.0
	defaultMaxOnDemandAccuracyM   = 150.0
	defaultStationaryPoorAccM     = 50.0
	defaultStationaryRadiusM      = 25.0
)

func maxPeriodicAccuracyM() float64 {
	return envFloat("LOCATION_MAX_PERIODIC_ACCURACY_M", defaultMaxPeriodicAccuracyM)
}

func maxOnDemandAccuracyM() float64 {
	return envFloat("LOCATION_MAX_ON_DEMAND_ACCURACY_M", defaultMaxOnDemandAccuracyM)
}

func envFloat(key string, fallback float64) float64 {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

// ShouldSkipPoorLocation отбрасывает periodic/on_demand с плохой точностью и «залипшие» координаты.
// accuracy == nil — не отклоняем только по точности (старые клиенты).
func ShouldSkipPoorLocation(
	source string,
	accuracy *float64,
	prev *models.Location,
	lat, lon float64,
) (bool, string) {
	if accuracy != nil {
		limit := maxPeriodicAccuracyM()
		if source == models.LocationSourceOnDemand {
			limit = maxOnDemandAccuracyM()
		}
		if *accuracy > limit {
			return true, "poor_accuracy"
		}
	}

	if source != models.LocationSourcePeriodic || prev == nil || accuracy == nil {
		return false, ""
	}

	dist := haversineDistanceM(prev.Latitude, prev.Longitude, lat, lon)
	if dist < defaultStationaryRadiusM && *accuracy > defaultStationaryPoorAccM {
		return true, "stationary_poor_accuracy"
	}

	return false, ""
}
