// Package dao dao/visit_dao.go
package dao

import (
	"locator/models"

	"gorm.io/gorm"
)

type VisitDAO struct {
	DB *gorm.DB
}

func NewVisitDAO(db *gorm.DB) *VisitDAO {
	return &VisitDAO{DB: db}
}

// Create сохраняет новый визит.
func (dao *VisitDAO) Create(visit *models.Visit) error {
	return dao.DB.Create(visit).Error
}

// Update обновляет существующий визит.
func (dao *VisitDAO) Update(visit *models.Visit) error {
	return dao.DB.Save(visit).Error
}

// GetActiveVisit возвращает активный визит (без заданного EndAt) для данного пользователя и чекпоинта.
func (dao *VisitDAO) GetActiveVisit(userID int, checkpointID int) (*models.Visit, error) {
	var visit models.Visit
	err := dao.DB.
		Where("user_id = ? AND checkpoint_id = ? AND end_at IS NULL", userID, checkpointID).
		First(&visit).Error
	if err != nil {
		return nil, err
	}
	return &visit, nil
}

// GetVisitsByUser возвращает все визиты для указанного пользователя.
func (dao *VisitDAO) GetVisitsByUser(userID int) ([]models.Visit, error) {
	var visits []models.Visit
	err := dao.DB.Where("user_id = ?", userID).Find(&visits).Error
	if err != nil {
		return nil, err
	}
	return visits, nil
}

// GetVisits возвращает список визитов с применением фильтров.
// Параметр filters может содержать ключи: "id", "user_id", "checkpoint_id" и т.д.
func (dao *VisitDAO) GetVisits(filters map[string]interface{}) ([]models.Visit, error) {
	var visits []models.Visit
	query := dao.DB
	for key, value := range filters {
		query = query.Where(key+" = ?", value)
	}
	if err := query.Find(&visits).Error; err != nil {
		return nil, err
	}
	return visits, nil
}
