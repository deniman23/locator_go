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

// GetByUserID возвращает запись о местоположении по идентификатору пользователя.
func (dao *LocationDAO) GetByUserID(userID int) (*models.Location, error) {
	var loc models.Location
	if err := dao.DB.Where("user_id = ?", userID).First(&loc).Error; err != nil {
		return nil, err
	}
	return &loc, nil
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

// GetLocationsBetween возвращает все записи о местоположениях, созданные между from и to.
func (dao *LocationDAO) GetLocationsBetween(from, to time.Time) ([]models.Location, error) {
	var locations []models.Location
	err := dao.DB.Where("created_at BETWEEN ? AND ?", from, to).Find(&locations).Error
	if err != nil {
		return nil, err
	}
	return locations, nil
}
