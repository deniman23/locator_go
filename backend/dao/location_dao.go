package dao

import (
	"locator/models"
	"time"

	"gorm.io/gorm"
)

// LocationDAO предоставляет методы для работы с данными о местоположении.
type LocationDAO struct {
	DB *gorm.DB
}

// NewLocationDAO создаёт новый экземпляр DAO.
func NewLocationDAO(db *gorm.DB) *LocationDAO {
	return &LocationDAO{DB: db}
}

// GetByUserID возвращает последнюю запись по времени фиксации (captured_at или created_at).
func (dao *LocationDAO) GetByUserID(userID int) (*models.Location, error) {
	var loc models.Location
	if err := dao.DB.Where("user_id = ?", userID).
		Order("COALESCE(captured_at, created_at) DESC").
		First(&loc).Error; err != nil {
		return nil, err
	}
	return &loc, nil
}

// GetPreviousByEffectiveTime — последняя точка пользователя строго раньше before (для офлайн-очереди).
func (dao *LocationDAO) GetPreviousByEffectiveTime(userID int, before time.Time) (*models.Location, error) {
	var loc models.Location
	err := dao.DB.Where(
		"user_id = ? AND COALESCE(captured_at, created_at) < ?",
		userID, before,
	).Order("COALESCE(captured_at, created_at) DESC").First(&loc).Error
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

// UserExists проверяет наличие пользователя (офлайн-очередь может слать старый user_id).
func (dao *LocationDAO) UserExists(userID int) (bool, error) {
	var n int64
	err := dao.DB.Table("users").Where("id = ?", userID).Count(&n).Error
	return n > 0, err
}

// Create вставляет новую запись о местоположении.
func (dao *LocationDAO) Create(loc *models.Location) error {
	return dao.DB.Create(loc).Error
}

// Update сохраняет изменения существующей записи.
func (dao *LocationDAO) Update(loc *models.Location) error {
	return dao.DB.Save(loc).Error
}

// GetAll возвращает все записи о местоположениях.
func (dao *LocationDAO) GetAll() ([]models.Location, error) {
	var locations []models.Location
	if err := dao.DB.Find(&locations).Error; err != nil {
		return nil, err
	}
	return locations, nil
}

// ListUserIDsWithoutCapturedAt — пользователи с точками без captured_at.
func (dao *LocationDAO) ListUserIDsWithoutCapturedAt() ([]int, error) {
	var ids []int
	err := dao.DB.Model(&models.Location{}).
		Where("captured_at IS NULL").
		Distinct("user_id").
		Pluck("user_id", &ids).Error
	return ids, err
}

// GetWithoutCapturedAtByUser возвращает точки пользователя без captured_at.
func (dao *LocationDAO) GetWithoutCapturedAtByUser(userID int) ([]models.Location, error) {
	var locations []models.Location
	err := dao.DB.Where("user_id = ? AND captured_at IS NULL", userID).
		Order("created_at ASC, id ASC").
		Find(&locations).Error
	return locations, err
}

// UpdateCapturedAt задаёт captured_at для записи.
func (dao *LocationDAO) UpdateCapturedAt(id int, capturedAt time.Time) error {
	return dao.DB.Model(&models.Location{}).
		Where("id = ?", id).
		Update("captured_at", capturedAt.UTC()).Error
}

// GetLocationsBetween возвращает все записи, у которых время фиксации попадает в интервал.
func (dao *LocationDAO) GetLocationsBetween(from, to time.Time) ([]models.Location, error) {
	var locations []models.Location
	err := dao.DB.
		Where("COALESCE(captured_at, created_at) BETWEEN ? AND ?", from, to).
		Order("COALESCE(captured_at, created_at) ASC").
		Find(&locations).Error
	if err != nil {
		return nil, err
	}
	return locations, nil
}

// GetLocationsByUserBetween возвращает точки пользователя за период по времени фиксации.
func (dao *LocationDAO) GetLocationsByUserBetween(userID int, from, to time.Time) ([]models.Location, error) {
	var locations []models.Location
	err := dao.DB.
		Where(
			"user_id = ? AND COALESCE(captured_at, created_at) BETWEEN ? AND ?",
			userID, from, to,
		).
		Order("COALESCE(captured_at, created_at) ASC").
		Find(&locations).Error
	if err != nil {
		return nil, err
	}
	return locations, nil
}
