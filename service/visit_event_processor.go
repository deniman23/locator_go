package service

import (
	"encoding/json"
	"errors"
	"log"

	"gorm.io/gorm"
	"locator/models"
)

// VisitEventProcessor отвечает за обработку событий локации из RabbitMQ.
type VisitEventProcessor struct {
	CheckpointService *CheckpointService
	VisitService      *VisitService
	// Можно добавить и другие зависимости (например, LocationService) при необходимости
}

// NewVisitEventProcessor создаёт новый экземпляр обработчика событий.
func NewVisitEventProcessor(cs *CheckpointService, vs *VisitService) *VisitEventProcessor {
	return &VisitEventProcessor{
		CheckpointService: cs,
		VisitService:      vs,
	}
}

// ProcessEvent обрабатывает событие из RabbitMQ:
// - Получает и десериализует входящее сообщение.
// - Получает все чекпоинты для обработки.
// - Для каждого чекпоинта определяет, находится ли пользователь в зоне, и запускает или завершает визит.
func (vep *VisitEventProcessor) ProcessEvent(message []byte) error {
	log.Println("[ProcessEvent] Получение и обработка события из RabbitMQ")
	var event models.LocationEvent
	if err := json.Unmarshal(message, &event); err != nil {
		log.Printf("[ProcessEvent] Ошибка десериализации события: %v", err)
		return err
	}
	log.Printf("[ProcessEvent] Событие успешно десериализовано: userID=%d, Latitude=%.6f, Longitude=%.6f",
		event.UserID, event.Latitude, event.Longitude)

	// Получаем список всех чекпоинтов.
	checkpoints, err := vep.CheckpointService.GetCheckpoints()
	if err != nil {
		log.Printf("[ProcessEvent] Ошибка получения чекпоинтов: %v", err)
		return err
	}
	log.Printf("[ProcessEvent] Получено %d чекпоинтов для обработки", len(checkpoints))

	// Обрабатываем каждый чекпоинт отдельно.
	for _, cp := range checkpoints {
		if err := vep.processCheckpoint(cp, event); err != nil {
			return err
		}
	}

	log.Println("[ProcessEvent] Обработка события завершена успешно")
	return nil
}

// processCheckpoint разделяет логику обработки для каждого чекпоинта.
func (vep *VisitEventProcessor) processCheckpoint(cp models.Checkpoint, event models.LocationEvent) error {
	log.Printf("[processCheckpoint] Проверка чекпоинта: ID=%d, Name=%s", cp.ID, cp.Name)

	// Определяем, находится ли локация пользователя внутри данного чекпоинта.
	inZone := vep.CheckpointService.IsLocationInCheckpoint(
		&models.Location{
			Latitude:  event.Latitude,
			Longitude: event.Longitude,
		},
		&cp,
	)

	// Получаем активный визит (если он есть).
	activeVisit, err := vep.getActiveVisit(event.UserID, cp.ID)
	if err != nil {
		return err
	}

	// Обрабатываем в зависимости от того, находится пользователь в зоне или нет.
	if inZone {
		return vep.handleInZone(event.UserID, cp.ID, activeVisit)
	} else {
		return vep.handleOutOfZone(event.UserID, cp.ID, activeVisit)
	}
}

// getActiveVisit получает активный визит для указанного пользователя и чекпоинта.
// Если возникает ошибка, не связанная с отсутствием записи, она логируется и передается дальше.
func (vep *VisitEventProcessor) getActiveVisit(userID, checkpointID int) (*models.Visit, error) {
	activeVisit, err := vep.VisitService.GetActiveVisit(userID, checkpointID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("[getActiveVisit] Ошибка получения активного визита для userID=%d, checkpointID=%d: %v", userID, checkpointID, err)
		return nil, err
	}
	return activeVisit, nil
}

// handleInZone обрабатывает ситуацию, когда пользователь находится в зоне чекпоинта.
// Если активного визита нет — запускает новый, иначе логирует, что визит уже активен.
func (vep *VisitEventProcessor) handleInZone(userID, checkpointID int, activeVisit *models.Visit) error {
	log.Printf("[handleInZone] Пользователь %d находится в зоне чекпоинта ID=%d", userID, checkpointID)
	if activeVisit == nil {
		_, err := vep.VisitService.StartVisit(userID, checkpointID)
		if err != nil {
			log.Printf("[handleInZone] Ошибка начала визита для userID=%d, checkpointID=%d: %v", userID, checkpointID, err)
			return err
		}
		log.Printf("[handleInZone] Начат новый визит для пользователя %d на чекпоинте %d", userID, checkpointID)
	} else {
		log.Printf("[handleInZone] Для пользователя %d уже существует активный визит на чекпоинте %d", userID, checkpointID)
	}
	return nil
}

// handleOutOfZone обрабатывает ситуацию, когда пользователь не находится в зоне чекпоинта.
// Если активный визит существует — завершаем его, иначе логируем отсутствие активного визита.
func (vep *VisitEventProcessor) handleOutOfZone(userID, checkpointID int, activeVisit *models.Visit) error {
	log.Printf("[handleOutOfZone] Пользователь %d вне зоны чекпоинта ID=%d", userID, checkpointID)
	if activeVisit != nil {
		if err := vep.VisitService.EndVisit(activeVisit); err != nil {
			log.Printf("[handleOutOfZone] Ошибка завершения визита для userID=%d, checkpointID=%d: %v", userID, checkpointID, err)
			return err
		}
		log.Printf("[handleOutOfZone] Завершён визит для пользователя %d на чекпоинте %d", userID, checkpointID)
	} else {
		log.Printf("[handleOutOfZone] Нет активного визита для завершения для пользователя %d на чекпоинте %d", userID, checkpointID)
	}
	return nil
}
