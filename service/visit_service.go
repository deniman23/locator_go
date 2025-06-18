package service

import (
	"locator/dao"
	"locator/models"
	"log"
	"time"
)

// VisitService отвечает за работу с визитами (замер времени посещений).
type VisitService struct {
	DAO *dao.VisitDAO
}

// NewVisitService создаёт новый экземпляр сервиса для работы с визитами.
func NewVisitService(dao *dao.VisitDAO) *VisitService {
	log.Printf("[NewVisitService] Инициализация сервиса визитов")
	return &VisitService{DAO: dao}
}

// GetActiveVisit возвращает активный визит пользователя в указанный чекпоинт.
func (vs *VisitService) GetActiveVisit(userID int, checkpointID int) (*models.Visit, error) {
	log.Printf("[GetActiveVisit] Запрос активного визита для userID=%d, checkpointID=%d", userID, checkpointID)
	visit, err := vs.DAO.GetActiveVisit(userID, checkpointID)
	if err != nil {
		log.Printf("[GetActiveVisit] Ошибка получения активного визита для userID=%d, checkpointID=%d: %v", userID, checkpointID, err)
	} else {
		log.Printf("[GetActiveVisit] Получен активный визит для userID=%d, checkpointID=%d: Visit=%+v", userID, checkpointID, visit)
	}
	return visit, err
}

// StartVisit начинается новый визит.
func (vs *VisitService) StartVisit(userID int, checkpointID int) (*models.Visit, error) {
	log.Printf("[StartVisit] Начало визита для userID=%d, checkpointID=%d", userID, checkpointID)
	visit := &models.Visit{
		UserID:       userID,
		CheckpointID: checkpointID,
		StartAt:      time.Now(),
	}
	err := vs.DAO.Create(visit)
	if err != nil {
		log.Printf("[StartVisit] Ошибка создания визита для userID=%d, checkpointID=%d: %v", userID, checkpointID, err)
		return nil, err
	}
	log.Printf("[StartVisit] Визит успешно начат для userID=%d, checkpointID=%d, VisitID=%d", userID, checkpointID, visit.ID)
	return visit, nil
}

// EndVisit завершает активный визит, фиксируя время окончания и вычисляя длительность.
func (vs *VisitService) EndVisit(visit *models.Visit) error {
	log.Printf("[EndVisit] Завершение визита: userID=%d, checkpointID=%d, VisitID=%d", visit.UserID, visit.CheckpointID, visit.ID)
	now := time.Now()
	visit.EndAt = &now
	visit.Duration = now.Sub(visit.StartAt).Seconds()
	err := vs.DAO.Update(visit)
	if err != nil {
		log.Printf("[EndVisit] Ошибка завершения визита для userID=%d, checkpointID=%d, VisitID=%d: %v", visit.UserID, visit.CheckpointID, visit.ID, err)
		return err
	}
	log.Printf("[EndVisit] Визит успешно завершен: userID=%d, checkpointID=%d, VisitID=%d, Длительность=%.2f секунд", visit.UserID, visit.CheckpointID, visit.ID, visit.Duration)
	return nil
}

// GetVisitsByUser возвращает все визиты для указанного пользователя.
func (vs *VisitService) GetVisitsByUser(userID int) ([]models.Visit, error) {
	log.Printf("[GetVisitsByUser] Запрос всех визитов для userID=%d", userID)
	visits, err := vs.DAO.GetVisitsByUser(userID)
	if err != nil {
		log.Printf("[GetVisitsByUser] Ошибка получения визитов для userID=%d: %v", userID, err)
		return nil, err
	}
	log.Printf("[GetVisitsByUser] Найдено %d визитов для userID=%d", len(visits), userID)
	return visits, nil
}
