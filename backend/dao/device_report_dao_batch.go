package dao

import "time"

// LatestDeviceReportRow — краткая сводка последнего отчёта устройства.
type LatestDeviceReportRow struct {
	UserID     int
	AppVersion string
	Platform   string
	Issues     []byte
	CreatedAt  time.Time
}

// GetLatestPerUser возвращает последний отчёт для каждого пользователя (один SQL).
func (dao *DeviceReportDAO) GetLatestPerUser() ([]LatestDeviceReportRow, error) {
	var rows []LatestDeviceReportRow
	err := dao.DB.Raw(`
		SELECT DISTINCT ON (user_id) user_id, app_version, platform, issues, created_at
		FROM device_reports
		ORDER BY user_id, created_at DESC
	`).Scan(&rows).Error
	return rows, err
}
