package service

import (
	"locator/dao"
	"locator/models"
	"log"
	"net/url"
	"strconv"
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
		StartAt:      time.Now().UTC(),
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
	log.Printf("[EndVisit] Завершение визита: userID=%d, checkpointID=%d, VisitID=%d",
		visit.UserID, visit.CheckpointID, visit.ID)

	// Приводим обе временные метки к UTC, чтобы избежать проблем с часовыми поясами.
	nowUTC := time.Now().UTC()
	startUTC := visit.StartAt.UTC()

	visit.EndAt = &nowUTC
	// Расчет длительности в секундах без дробной части.
	visit.Duration = int(nowUTC.Sub(startUTC).Seconds())

	err := vs.DAO.Update(visit)
	if err != nil {
		log.Printf("[EndVisit] Ошибка завершения визита для userID=%d, checkpointID=%d, VisitID=%d: %v",
			visit.UserID, visit.CheckpointID, visit.ID, err)
		return err
	}
	log.Printf("[EndVisit] Визит успешно завершен: userID=%d, checkpointID=%d, VisitID=%d, Длительность=%d секунд",
		visit.UserID, visit.CheckpointID, visit.ID, visit.Duration)
	return nil
}

// GetVisits возвращает список визитов с применением переданных фильтров.
func (vs *VisitService) GetVisits(filters map[string]interface{}) ([]models.Visit, error) {
	return vs.DAO.GetVisits(filters)
}

// GetVisitsByFilters анализирует query-параметры, формирует фильтры и возвращает список визитов.
func (vs *VisitService) GetVisitsByFilters(params url.Values) ([]models.Visit, error) {
	filters := make(map[string]interface{})

	if idStr := params.Get("id"); idStr != "" {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Printf("[GetVisitsByFilters] неверный формат параметра id: %v", err)
		} else {
			filters["id"] = id
		}
	}

	if userIDStr := params.Get("user_id"); userIDStr != "" {
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			log.Printf("[GetVisitsByFilters] неверный формат параметра user_id: %v", err)
		} else {
			filters["user_id"] = userID
		}
	}

	if checkpointIDStr := params.Get("checkpoint_id"); checkpointIDStr != "" {
		checkpointID, err := strconv.Atoi(checkpointIDStr)
		if err != nil {
			log.Printf("[GetVisitsByFilters] неверный формат параметра checkpoint_id: %v", err)
		} else {
			filters["checkpoint_id"] = checkpointID
		}
	}

	return vs.GetVisits(filters)
}
