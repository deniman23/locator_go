package service

import (
	"fmt"
	"locator/dao"
	"locator/models"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// VisitService отвечает за работу с визитами (замер времени посещений).
type VisitService struct {
	DAO            *dao.VisitDAO
	TravelSegments *TravelSegmentService
}

// NewVisitService создаёт новый экземпляр сервиса для работы с визитами.
func NewVisitService(dao *dao.VisitDAO, travelSegments *TravelSegmentService) *VisitService {
	log.Printf("[NewVisitService] Инициализация сервиса визитов")
	return &VisitService{DAO: dao, TravelSegments: travelSegments}
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

// AbandonVisit удаляет активный визит без записи в историю (короткий ложный визит).
func (vs *VisitService) AbandonVisit(visit *models.Visit) error {
	log.Printf("[AbandonVisit] Удаление короткого визита: userID=%d, checkpointID=%d, VisitID=%d",
		visit.UserID, visit.CheckpointID, visit.ID)
	return vs.DAO.Delete(visit.ID)
}

// GetVisits возвращает список визитов с применением переданных фильтров.
func (vs *VisitService) GetVisits(
	filters map[string]interface{},
	activeOnly bool,
	rangeFrom, rangeTo *time.Time,
) ([]models.Visit, error) {
	return vs.DAO.GetVisits(filters, activeOnly, rangeFrom, rangeTo)
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

	activeOnly := false
	if a := params.Get("active"); a == "true" || a == "1" {
		activeOnly = true
	}

	var rangeFrom, rangeTo *time.Time
	fromStr := params.Get("from")
	toStr := params.Get("to")
	if fromStr != "" || toStr != "" {
		if fromStr == "" || toStr == "" {
			return nil, fmt.Errorf("укажите оба параметра from и to")
		}
		from, to, err := parseVisitQueryRange(fromStr, toStr)
		if err != nil {
			return nil, err
		}
		rangeFrom = &from
		rangeTo = &to
	}

	visits, err := vs.GetVisits(filters, activeOnly, rangeFrom, rangeTo)
	if err != nil {
		return nil, err
	}

	includeOutside := params.Get("include_outside") == "true" || params.Get("include_outside") == "1"
	if !includeOutside || vs.TravelSegments == nil || activeOnly {
		return visits, nil
	}

	userID, hasUser := filters["user_id"].(int)
	if !hasUser || rangeFrom == nil || rangeTo == nil {
		return nil, fmt.Errorf("для участков вне чекпоинтов укажите user_id, from и to")
	}

	outside, err := vs.TravelSegments.GetOutsideSegments(userID, *rangeFrom, *rangeTo)
	if err != nil {
		return nil, err
	}

	return mergeVisitsSorted(visits, outside), nil
}

func parseVisitQueryRange(fromStr, toStr string) (time.Time, time.Time, error) {
	loc, err := time.LoadLocation("Europe/Minsk")
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("часовой пояс Europe/Minsk: %w", err)
	}
	from, err := parseVisitQueryTime(fromStr, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid from: %w", err)
	}
	to, err := parseVisitQueryTime(toStr, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid to: %w", err)
	}
	to = normalizeVisitRangeEnd(to, toStr)
	if from.After(to) {
		return time.Time{}, time.Time{}, fmt.Errorf("начало интервала не может быть позже окончания")
	}
	return from.UTC(), to.UTC(), nil
}

func parseVisitQueryTime(s string, loc *time.Location) (time.Time, error) {
	if strings.ContainsAny(s, "Z+") {
		return time.Parse(time.RFC3339, s)
	}
	return time.ParseInLocation("2006-01-02T15:04", s, loc)
}

func normalizeVisitRangeEnd(to time.Time, toStr string) time.Time {
	if !strings.Contains(toStr, "T") {
		return to
	}
	timePart := strings.SplitN(toStr, "T", 2)[1]
	if strings.Count(timePart, ":") == 1 {
		return to.Add(59*time.Second + 999*time.Millisecond)
	}
	return to
}
