package dao

import (
	"locator/models"

	"gorm.io/gorm"
)

type DeviceReportDAO struct {
	DB *gorm.DB
}

func NewDeviceReportDAO(db *gorm.DB) *DeviceReportDAO {
	return &DeviceReportDAO{DB: db}
}

func (dao *DeviceReportDAO) Create(report *models.DeviceReport) error {
	return dao.DB.Create(report).Error
}

func (dao *DeviceReportDAO) GetLatestByUserID(userID int) (*models.DeviceReport, error) {
	var report models.DeviceReport
	err := dao.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		First(&report).Error
	if err != nil {
		return nil, err
	}
	return &report, nil
}
