package service

import (
	"encoding/json"
	"locator/dao"
	"time"
)

const gpsOnlineSeconds = 300

// UserDeviceStatusSummary — краткий статус устройства для списка пользователей.
type UserDeviceStatusSummary struct {
	GPS          string   `json:"gps"`
	AgeSeconds   *int64   `json:"age_seconds,omitempty"`
	Healthy      *bool    `json:"healthy,omitempty"`
	LastReportAt *string  `json:"last_report_at,omitempty"`
	AppVersion   string   `json:"app_version,omitempty"`
	Platform     string   `json:"platform,omitempty"`
	Issues       []string `json:"issues"`
}

// DeviceStatusService — пакетная сводка GPS и health для админки.
type DeviceStatusService struct {
	LocationDAO *dao.LocationDAO
	ReportDAO   *dao.DeviceReportDAO
}

func NewDeviceStatusService(locationDAO *dao.LocationDAO, reportDAO *dao.DeviceReportDAO) *DeviceStatusService {
	return &DeviceStatusService{LocationDAO: locationDAO, ReportDAO: reportDAO}
}

// AllUsersSummary — один запрос к БД вместо N×2 HTTP из админки.
func (svc *DeviceStatusService) AllUsersSummary() (map[int]UserDeviceStatusSummary, error) {
	out := make(map[int]UserDeviceStatusSummary)

	if svc.LocationDAO != nil {
		ages, err := svc.LocationDAO.GetLatestAgePerUser()
		if err != nil {
			return nil, err
		}
		for _, row := range ages {
			age := row.AgeSeconds
			gps := "stale"
			if age <= gpsOnlineSeconds {
				gps = "online"
			}
			out[row.UserID] = UserDeviceStatusSummary{
				GPS:        gps,
				AgeSeconds: &age,
				Issues:     []string{},
			}
		}
	}

	if svc.ReportDAO == nil {
		return out, nil
	}

	reports, err := svc.ReportDAO.GetLatestPerUser()
	if err != nil {
		return nil, err
	}
	for _, row := range reports {
		issues, _ := parseIssuesJSON(row.Issues)
		healthy := len(issues) == 0
		ts := row.CreatedAt.UTC().Format(time.RFC3339)
		prev := out[row.UserID]
		prev.Healthy = &healthy
		prev.LastReportAt = &ts
		prev.AppVersion = row.AppVersion
		prev.Platform = row.Platform
		prev.Issues = issues
		if prev.Issues == nil {
			prev.Issues = []string{}
		}
		out[row.UserID] = prev
	}

	return out, nil
}

func parseIssuesJSON(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var issues []string
	if err := json.Unmarshal(raw, &issues); err == nil {
		return issues, nil
	}
	var generic []interface{}
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(generic))
	for _, item := range generic {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out, nil
}
