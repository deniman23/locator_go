package dao

import (
	"locator/location"

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
func (dao *LocationDAO) GetByUserID(userID int) (*location.Location, error) {
	var loc location.Location
	if err := dao.DB.Where("user_id = ?", userID).First(&loc).Error; err != nil {
		return nil, err
	}
	return &loc, nil
}

// Create вставляет новую запись о местоположении.
func (dao *LocationDAO) Create(loc *location.Location) error {
	return dao.DB.Create(loc).Error
}

// Update сохраняет изменения существующей записи.
func (dao *LocationDAO) Update(loc *location.Location) error {
	return dao.DB.Save(loc).Error
}

// GetAll возвращает все записи о местоположениях.
func (dao *LocationDAO) GetAll() ([]location.Location, error) {
	var locations []location.Location
	if err := dao.DB.Find(&locations).Error; err != nil {
		return nil, err
	}
	return locations, nil
}
