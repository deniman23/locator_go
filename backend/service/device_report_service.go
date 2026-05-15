package service

import (
	"encoding/json"
	"errors"
	"locator/dao"
	"locator/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var ErrDeviceReportNotFound = errors.New("device report not found")

// DeviceReportService — диагностические отчёты с устройств.
type DeviceReportService struct {
	DAO *dao.DeviceReportDAO
}

func NewDeviceReportService(dao *dao.DeviceReportDAO) *DeviceReportService {
	return &DeviceReportService{DAO: dao}
}

// SaveReport сохраняет отчёт с устройства.
func (svc *DeviceReportService) SaveReport(userID int, body map[string]interface{}) (*models.DeviceReport, error) {
	reportJSON, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	report := &models.DeviceReport{
		UserID: userID,
		Report: datatypes.JSON(reportJSON),
	}

	if v, ok := body["app_version"].(string); ok {
		report.AppVersion = v
	}
	if v, ok := body["platform"].(string); ok {
		report.Platform = v
	}
	if issuesRaw, ok := body["issues"]; ok {
		issuesBytes, err := json.Marshal(issuesRaw)
		if err != nil {
			return nil, err
		}
		report.Issues = datatypes.JSON(issuesBytes)
	}

	if err := svc.DAO.Create(report); err != nil {
		return nil, err
	}
	return report, nil
}

// GetLatestByUserID возвращает последний отчёт пользователя.
func (svc *DeviceReportService) GetLatestByUserID(userID int) (*models.DeviceReport, error) {
	report, err := svc.DAO.GetLatestByUserID(userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrDeviceReportNotFound
	}
	if err != nil {
		return nil, err
	}
	return report, nil
}

// ReportAsMap разбирает JSON отчёта в map.
func ReportAsMap(report *models.DeviceReport) (map[string]interface{}, error) {
	if report == nil || len(report.Report) == 0 {
		return map[string]interface{}{}, nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal(report.Report, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]interface{}{}
	}
	return out, nil
}

// IssuesSlice возвращает список проблем из отчёта.
func IssuesSlice(report *models.DeviceReport) ([]string, error) {
	if report == nil || len(report.Issues) == 0 {
		return nil, nil
	}

	var issues []string
	if err := json.Unmarshal(report.Issues, &issues); err == nil {
		return issues, nil
	}

	var generic []interface{}
	if err := json.Unmarshal(report.Issues, &generic); err != nil {
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
