package service

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrDeviceConfigUpdateEmpty   = errors.New("укажите хотя бы одно поле конфигурации")
	ErrDeviceConfigUpdateInvalid = errors.New("некорректные параметры конфигурации")
)

var adminPinPattern = regexp.MustCompile(`^\d{4,12}$`)

// DeviceConfigUpdateInput — поля config_update, допустимые с админки.
type DeviceConfigUpdateInput struct {
	APIKey                      *string `json:"api_key"`
	APIBaseURL                  *string `json:"api_base_url"`
	TrackingPaused              *bool   `json:"tracking_paused"`
	LocationIntervalSeconds     *int64  `json:"location_interval_seconds"`
	PollIntervalSeconds         *int64  `json:"poll_interval_seconds"`
	HealthReportIntervalSeconds *int64  `json:"health_report_interval_seconds"`
	AdminPin                    *string `json:"admin_pin"`
	HiddenFromLauncher          *bool   `json:"hidden_from_launcher"`
	WakeDevice                  *bool   `json:"wake_device"`
}

// BuildConfigUpdatePayload валидирует и собирает payload для команды config_update.
func BuildConfigUpdatePayload(userID int, in DeviceConfigUpdateInput) (map[string]interface{}, error) {
	payload := make(map[string]interface{})

	if in.APIKey != nil {
		key := strings.TrimSpace(*in.APIKey)
		if len(key) < 8 {
			return nil, ErrDeviceConfigUpdateInvalid
		}
		payload["api_key"] = key
		payload["user_id"] = userID
	}
	if in.APIBaseURL != nil {
		url := strings.TrimSpace(strings.TrimRight(*in.APIBaseURL, "/"))
		if url == "" || !strings.HasPrefix(url, "http") {
			return nil, ErrDeviceConfigUpdateInvalid
		}
		payload["api_base_url"] = url
	}
	if in.TrackingPaused != nil {
		payload["tracking_paused"] = *in.TrackingPaused
	}
	if in.LocationIntervalSeconds != nil {
		if *in.LocationIntervalSeconds < 30 {
			return nil, ErrDeviceConfigUpdateInvalid
		}
		payload["location_interval_seconds"] = *in.LocationIntervalSeconds
	}
	if in.PollIntervalSeconds != nil {
		if *in.PollIntervalSeconds < 5 {
			return nil, ErrDeviceConfigUpdateInvalid
		}
		payload["poll_interval_seconds"] = *in.PollIntervalSeconds
	}
	if in.HealthReportIntervalSeconds != nil {
		if *in.HealthReportIntervalSeconds < 60 {
			return nil, ErrDeviceConfigUpdateInvalid
		}
		payload["health_report_interval_seconds"] = *in.HealthReportIntervalSeconds
	}
	if in.AdminPin != nil {
		pin := strings.TrimSpace(*in.AdminPin)
		if !adminPinPattern.MatchString(pin) {
			return nil, ErrDeviceConfigUpdateInvalid
		}
		payload["admin_pin"] = pin
	}
	if in.HiddenFromLauncher != nil {
		payload["hidden_from_launcher"] = *in.HiddenFromLauncher
	}
	if in.WakeDevice != nil && *in.WakeDevice {
		payload["wake_device"] = true
	}

	if len(payload) == 0 {
		return nil, ErrDeviceConfigUpdateEmpty
	}
	return payload, nil
}
