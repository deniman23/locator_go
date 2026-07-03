package service

import (
	"os"
	"strconv"
	"strings"
)

const (
	defaultGeofenceExitBufferMeters  = 40.0
	defaultGeofenceExitGraceSeconds  = 90
	defaultGeofenceEnterGraceSeconds = 30
	defaultGeofenceMinVisitSeconds   = 60
	defaultGeofenceFarExitMeters     = 300.0
	defaultGeofenceStaleGapSeconds   = 600
)

func geofenceExitBufferMeters() float64 {
	s := strings.TrimSpace(os.Getenv("GEOFENCE_EXIT_BUFFER_METERS"))
	if s == "" {
		return defaultGeofenceExitBufferMeters
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < 0 {
		return defaultGeofenceExitBufferMeters
	}
	return v
}

func geofenceExitGraceSeconds() int {
	return geofenceSecondsEnv("GEOFENCE_EXIT_GRACE_SECONDS", defaultGeofenceExitGraceSeconds)
}

func geofenceEnterGraceSeconds() int {
	return geofenceSecondsEnv("GEOFENCE_ENTER_GRACE_SECONDS", defaultGeofenceEnterGraceSeconds)
}

func geofenceMinVisitSeconds() int {
	return geofenceSecondsEnv("GEOFENCE_MIN_VISIT_SECONDS", defaultGeofenceMinVisitSeconds)
}

func geofenceFarExitMeters() float64 {
	s := strings.TrimSpace(os.Getenv("GEOFENCE_FAR_EXIT_METERS"))
	if s == "" {
		return defaultGeofenceFarExitMeters
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < 0 {
		return defaultGeofenceFarExitMeters
	}
	return v
}

func geofenceStaleGapSeconds() int {
	return geofenceSecondsEnv("GEOFENCE_STALE_GAP_SECONDS", defaultGeofenceStaleGapSeconds)
}

func geofenceSecondsEnv(key string, fallback int) int {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return fallback
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return fallback
	}
	return v
}

// geofenceInside определяет, считается ли точка внутри зоны с учётом гистерезиса.
// Для активного визита зона шире (radius + buffer), чтобы кратковременный выход GPS не закрывал визит.
func geofenceInside(distance, radius float64, hasActiveVisit bool) bool {
	if hasActiveVisit {
		return distance <= radius+geofenceExitBufferMeters()
	}
	return distance <= radius
}

// geofenceFarOutside — явный уход далеко от чекпоинта (не путать с дрожанием GPS у границы).
func geofenceFarOutside(distance, radius float64, hasActiveVisit bool) bool {
	if !hasActiveVisit {
		return false
	}
	return distance > radius+geofenceExitBufferMeters()+geofenceFarExitMeters()
}
