package service

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"gorm.io/gorm"
	"locator/models"
)

// VisitEventProcessor отвечает за обработку событий локации из RabbitMQ.
type VisitEventProcessor struct {
	CheckpointService *CheckpointService
	VisitService      *VisitService
	geofenceStates    *geofenceStateStore
}

// NewVisitEventProcessor создаёт новый экземпляр обработчика событий.
func NewVisitEventProcessor(cs *CheckpointService, vs *VisitService) *VisitEventProcessor {
	return &VisitEventProcessor{
		CheckpointService: cs,
		VisitService:      vs,
		geofenceStates:    newGeofenceStateStore(),
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

	checkpoints, err := vep.CheckpointService.GetCheckpoints()
	if err != nil {
		log.Printf("[ProcessEvent] Ошибка получения чекпоинтов: %v", err)
		return err
	}
	log.Printf("[ProcessEvent] Получено %d чекпоинтов для обработки", len(checkpoints))

	for _, cp := range checkpoints {
		if err := vep.processCheckpoint(cp, event); err != nil {
			return err
		}
	}

	log.Println("[ProcessEvent] Обработка события завершена успешно")
	return nil
}

func (vep *VisitEventProcessor) processCheckpoint(cp models.Checkpoint, event models.LocationEvent) error {
	log.Printf("[processCheckpoint] Проверка чекпоинта: ID=%d, Name=%s", cp.ID, cp.Name)

	distance := vep.CheckpointService.DistanceToCheckpoint(event.Latitude, event.Longitude, &cp)
	now := event.OccurredAt.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	activeVisit, err := vep.getActiveVisit(event.UserID, cp.ID)
	if err != nil {
		return err
	}

	hasVisit := activeVisit != nil
	inside := geofenceInside(distance, cp.Radius, hasVisit)
	state := vep.geofenceStates.get(event.UserID, cp.ID)

	log.Printf("[processCheckpoint] userID=%d checkpointID=%d distance=%.1fm radius=%.1fm inside=%v hasVisit=%v",
		event.UserID, cp.ID, distance, cp.Radius, inside, hasVisit)

	if inside {
		state.clearPendingExit()
		return vep.handleInside(event.UserID, cp.ID, activeVisit, state, now)
	}

	state.clearPendingEnter()
	return vep.handleOutside(event.UserID, cp.ID, activeVisit, state, now)
}

func (vep *VisitEventProcessor) getActiveVisit(userID, checkpointID int) (*models.Visit, error) {
	activeVisit, err := vep.VisitService.GetActiveVisit(userID, checkpointID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("[getActiveVisit] Ошибка получения активного визита для userID=%d, checkpointID=%d: %v", userID, checkpointID, err)
		return nil, err
	}
	return activeVisit, nil
}

// handleInside: при отсутствии визита ждём устойчивого нахождения в зоне, затем создаём визит.
func (vep *VisitEventProcessor) handleInside(userID, checkpointID int, activeVisit *models.Visit, state *geofencePendingState, now time.Time) error {
	if activeVisit != nil {
		log.Printf("[handleInside] Пользователь %d в зоне чекпоинта %d, визит уже активен", userID, checkpointID)
		return nil
	}

	enterGrace := geofenceEnterGraceSeconds()
	if !state.pendingEnterElapsed(now, enterGrace) {
		state.markPendingEnter(now)
		log.Printf("[handleInside] Ожидание подтверждения входа userID=%d checkpointID=%d (grace=%ds)",
			userID, checkpointID, enterGrace)
		return nil
	}

	state.clearPendingEnter()
	_, err := vep.VisitService.StartVisit(userID, checkpointID)
	if err != nil {
		log.Printf("[handleInside] Ошибка начала визита для userID=%d, checkpointID=%d: %v", userID, checkpointID, err)
		return err
	}
	log.Printf("[handleInside] Начат визит для userID=%d checkpointID=%d после %ds в зоне",
		userID, checkpointID, enterGrace)
	return nil
}

// handleOutside: при активном визите ждём устойчивого выхода из зоны, затем завершаем.
func (vep *VisitEventProcessor) handleOutside(userID, checkpointID int, activeVisit *models.Visit, state *geofencePendingState, now time.Time) error {
	if activeVisit == nil {
		return nil
	}

	exitGrace := geofenceExitGraceSeconds()
	if !state.pendingExitElapsed(now, exitGrace) {
		state.markPendingExit(now)
		log.Printf("[handleOutside] Ожидание подтверждения выхода userID=%d checkpointID=%d (grace=%ds)",
			userID, checkpointID, exitGrace)
		return nil
	}

	state.clearPendingExit()

	minVisit := geofenceMinVisitSeconds()
	elapsed := int(now.Sub(activeVisit.StartAt.UTC()).Seconds())
	if elapsed < minVisit {
		if err := vep.VisitService.AbandonVisit(activeVisit); err != nil {
			log.Printf("[handleOutside] Ошибка отмены короткого визита userID=%d checkpointID=%d: %v",
				userID, checkpointID, err)
			return err
		}
		log.Printf("[handleOutside] Короткий визит (%ds < %ds) отменён для userID=%d checkpointID=%d",
			elapsed, minVisit, userID, checkpointID)
		return nil
	}

	if err := vep.VisitService.EndVisit(activeVisit); err != nil {
		log.Printf("[handleOutside] Ошибка завершения визита userID=%d checkpointID=%d: %v",
			userID, checkpointID, err)
		return err
	}
	log.Printf("[handleOutside] Завершён визит для userID=%d checkpointID=%d", userID, checkpointID)
	return nil
}
