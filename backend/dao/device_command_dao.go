package dao

import (
	"errors"
	"locator/models"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// ClaimNext берёт самую старую pending-команду или delivered с истёкшей арендой
// и продлевает аренду. Два параллельных poll не получают одну и ту же строку.
func (dao *DeviceCommandDAO) ClaimNext(userID int, now time.Time, lease time.Duration) (*models.DeviceCommand, error) {
	var claimed models.DeviceCommand
	err := dao.DB.Transaction(func(tx *gorm.DB) error {
		var cmd models.DeviceCommand
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("user_id = ? AND (status = ? OR (status = ? AND (lease_until IS NULL OR lease_until < ?)))",
				userID,
				models.DeviceCommandStatusPending,
				models.DeviceCommandStatusDelivered,
				now,
			).
			Order("created_at ASC").
			First(&cmd).Error
		if err != nil {
			return err
		}
		until := now.Add(lease)
		if err := tx.Model(&models.DeviceCommand{}).Where("id = ?", cmd.ID).Updates(map[string]interface{}{
			"status":       models.DeviceCommandStatusDelivered,
			"delivered_at": now,
			"lease_until":  until,
		}).Error; err != nil {
			return err
		}
		cmd.Status = models.DeviceCommandStatusDelivered
		cmd.DeliveredAt = &now
		cmd.LeaseUntil = &until
		claimed = cmd
		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, gorm.ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	return &claimed, nil
}

func (dao *DeviceCommandDAO) SetCreatedBy(id string, adminID int) error {
	return dao.DB.Model(&models.DeviceCommand{}).Where("id = ?", id).Update("created_by", adminID).Error
}

func (dao *DeviceCommandDAO) RedactAPIKey(id string) error {
	return dao.DB.Exec(
		`UPDATE device_commands SET payload = payload - 'api_key' WHERE id = ? AND payload ? 'api_key'`,
		id,
	).Error
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

// MarkProgress сохраняет промежуточный ack (accepted/downloaded/installing) без закрытия команды.
func (dao *DeviceCommandDAO) MarkProgress(id, ackStatus, ackMessage string, at time.Time) error {
	return dao.DB.Model(&models.DeviceCommand{}).Where("id = ?", id).Updates(map[string]interface{}{
		"ack_status":  ackStatus,
		"ack_message": ackMessage,
		"acked_at":    at,
	}).Error
}

func (dao *DeviceCommandDAO) ExpirePendingOlderThan(cutoff time.Time) error {
	return dao.ExpirePendingOlderThanExceptType(cutoff, "")
}

func (dao *DeviceCommandDAO) ExpirePendingOlderThanExceptType(cutoff time.Time, exceptType string) error {
	q := dao.DB.Model(&models.DeviceCommand{}).
		Where("status IN ? AND created_at < ?", []string{
			models.DeviceCommandStatusPending,
			models.DeviceCommandStatusDelivered,
		}, cutoff)
	if exceptType != "" {
		q = q.Where("type <> ?", exceptType)
	}
	return q.Update("status", models.DeviceCommandStatusExpired).Error
}

func (dao *DeviceCommandDAO) ExpirePendingOlderThanType(cutoff time.Time, cmdType string) error {
	return dao.DB.Model(&models.DeviceCommand{}).
		Where("status IN ? AND created_at < ? AND type = ?", []string{
			models.DeviceCommandStatusPending,
			models.DeviceCommandStatusDelivered,
		}, cutoff, cmdType).
		Update("status", models.DeviceCommandStatusExpired).Error
}

func (dao *DeviceCommandDAO) CancelPendingForUser(userID int, cmdType, excludeID string) error {
	q := dao.DB.Model(&models.DeviceCommand{}).
		Where("user_id = ? AND status = ? AND type = ?", userID, models.DeviceCommandStatusPending, cmdType)
	if excludeID != "" {
		q = q.Where("id <> ?", excludeID)
	}
	return q.Update("status", models.DeviceCommandStatusExpired).Error
}
