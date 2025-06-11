package service

import (
	"locator/dao"
	"locator/models"
	"time"
)

// VisitService отвечает за работу с визитами (замер времени посещений).
type VisitService struct {
	DAO *dao.VisitDAO
}

func NewVisitService(dao *dao.VisitDAO) *VisitService {
	return &VisitService{DAO: dao}
}

// GetActiveVisit возвращает активный визит пользователя в указанный чекпоинт.
func (vs *VisitService) GetActiveVisit(userID int, checkpointID int) (*models.Visit, error) {
	return vs.DAO.GetActiveVisit(userID, checkpointID)
}

// StartVisit начинается новый визит.
func (vs *VisitService) StartVisit(userID int, checkpointID int) (*models.Visit, error) {
	visit := &models.Visit{
		UserID:       userID,
		CheckpointID: checkpointID,
		StartAt:      time.Now(),
	}
	err := vs.DAO.Create(visit)
	if err != nil {
		return nil, err
	}
	return visit, nil
}

// EndVisit завершает активный визит, фиксируя время окончания и вычисляя длительность.
func (vs *VisitService) EndVisit(visit *models.Visit) error {
	now := time.Now()
	visit.EndAt = &now
	visit.Duration = now.Sub(visit.StartAt).Seconds()
	return vs.DAO.Update(visit)
}

// GetVisitsByUser возвращает все визиты для указанного пользователя.
func (vs *VisitService) GetVisitsByUser(userID int) ([]models.Visit, error) {
	return vs.DAO.GetVisitsByUser(userID)
}
