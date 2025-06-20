package dao

import (
	"locator/models"

	"gorm.io/gorm"
)

// CheckpointDAO предоставляет методы для работы с данными чекпоинтов.
type CheckpointDAO struct {
	DB *gorm.DB
}

// NewCheckpointDAO создаёт новый экземпляр CheckpointDAO.
func NewCheckpointDAO(db *gorm.DB) *CheckpointDAO {
	return &CheckpointDAO{DB: db}
}

// Create вставляет новый чекпоинт в базу данных.
func (dao *CheckpointDAO) Create(cp *models.Checkpoint) error {
	return dao.DB.Create(cp).Error
}

// GetAll возвращает список всех чекпоинтов.
func (dao *CheckpointDAO) GetAll() ([]models.Checkpoint, error) {
	var checkpoints []models.Checkpoint
	if err := dao.DB.Find(&checkpoints).Error; err != nil {
		return nil, err
	}
	return checkpoints, nil
}

// GetByID возвращает чекпоинт по его ID.
func (dao *CheckpointDAO) GetByID(id int) (*models.Checkpoint, error) {
	var cp models.Checkpoint
	if err := dao.DB.First(&cp, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &cp, nil
}
