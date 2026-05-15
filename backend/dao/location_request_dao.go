package dao

import (
	"locator/models"
	"time"

	"gorm.io/gorm"
)

type LocationRequestDAO struct {
	DB *gorm.DB
}

func NewLocationRequestDAO(db *gorm.DB) *LocationRequestDAO {
	return &LocationRequestDAO{DB: db}
}

func (dao *LocationRequestDAO) Create(req *models.LocationRequest) error {
	return dao.DB.Create(req).Error
}

func (dao *LocationRequestDAO) GetByID(id string) (*models.LocationRequest, error) {
	var req models.LocationRequest
	if err := dao.DB.Where("id = ?", id).First(&req).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

func (dao *LocationRequestDAO) GetPendingByUserID(userID int) (*models.LocationRequest, error) {
	var req models.LocationRequest
	err := dao.DB.
		Where("user_id = ? AND status = ?", userID, models.LocationRequestStatusPending).
		Order("created_at DESC").
		First(&req).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (dao *LocationRequestDAO) UpdateStatus(id, status string, completedAt *time.Time) error {
	updates := map[string]interface{}{"status": status}
	if completedAt != nil {
		updates["completed_at"] = completedAt
	}
	return dao.DB.Model(&models.LocationRequest{}).Where("id = ?", id).Updates(updates).Error
}

// ExpirePendingOlderThan помечает просроченные pending-запросы как expired.
func (dao *LocationRequestDAO) ExpirePendingOlderThan(cutoff time.Time) error {
	return dao.DB.Model(&models.LocationRequest{}).
		Where("status = ? AND created_at < ?", models.LocationRequestStatusPending, cutoff).
		Update("status", models.LocationRequestStatusExpired).Error
}

// CancelPendingForUser отменяет (expire) все pending для пользователя, кроме excludeID (если не пустой).
func (dao *LocationRequestDAO) CancelPendingForUser(userID int, excludeID string) error {
	q := dao.DB.Model(&models.LocationRequest{}).
		Where("user_id = ? AND status = ?", userID, models.LocationRequestStatusPending)
	if excludeID != "" {
		q = q.Where("id <> ?", excludeID)
	}
	return q.Update("status", models.LocationRequestStatusExpired).Error
}
