package dao

import (
	"locator/models"
	"time"

	"gorm.io/gorm"
)

type DeviceCommandDAO struct {
	DB *gorm.DB
}

func NewDeviceCommandDAO(db *gorm.DB) *DeviceCommandDAO {
	return &DeviceCommandDAO{DB: db}
}

func (dao *DeviceCommandDAO) Create(cmd *models.DeviceCommand) error {
	return dao.DB.Create(cmd).Error
}

func (dao *DeviceCommandDAO) GetByID(id string) (*models.DeviceCommand, error) {
	var cmd models.DeviceCommand
	if err := dao.DB.Where("id = ?", id).First(&cmd).Error; err != nil {
		return nil, err
	}
	return &cmd, nil
}

func (dao *DeviceCommandDAO) GetNextPending(userID int) (*models.DeviceCommand, error) {
	var cmd models.DeviceCommand
	err := dao.DB.
		Where("user_id = ? AND status = ?", userID, models.DeviceCommandStatusPending).
		Order("created_at ASC").
		First(&cmd).Error
	if err != nil {
		return nil, err
	}
	return &cmd, nil
}

func (dao *DeviceCommandDAO) MarkDelivered(id string, at time.Time) error {
	return dao.DB.Model(&models.DeviceCommand{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       models.DeviceCommandStatusDelivered,
		"delivered_at": at,
	}).Error
}

func (dao *DeviceCommandDAO) MarkAcked(id, ackStatus, ackMessage string, at time.Time) error {
	return dao.DB.Model(&models.DeviceCommand{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      models.DeviceCommandStatusAcked,
		"ack_status":  ackStatus,
		"ack_message": ackMessage,
		"acked_at":    at,
	}).Error
}

func (dao *DeviceCommandDAO) MarkFailed(id, ackStatus, ackMessage string, at time.Time) error {
	return dao.DB.Model(&models.DeviceCommand{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      models.DeviceCommandStatusFailed,
		"ack_status":  ackStatus,
		"ack_message": ackMessage,
		"acked_at":    at,
	}).Error
}

func (dao *DeviceCommandDAO) ExpirePendingOlderThan(cutoff time.Time) error {
	return dao.DB.Model(&models.DeviceCommand{}).
		Where("status IN ? AND created_at < ?", []string{
			models.DeviceCommandStatusPending,
			models.DeviceCommandStatusDelivered,
		}, cutoff).
		Update("status", models.DeviceCommandStatusExpired).Error
}

func (dao *DeviceCommandDAO) CancelPendingForUser(userID int, excludeID string) error {
	q := dao.DB.Model(&models.DeviceCommand{}).
		Where("user_id = ? AND status = ?", userID, models.DeviceCommandStatusPending)
	if excludeID != "" {
		q = q.Where("id <> ?", excludeID)
	}
	return q.Update("status", models.DeviceCommandStatusExpired).Error
}
