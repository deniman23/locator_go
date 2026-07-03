package service

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"gorm.io/gorm"
	"locator/models"
)

// visitLocationReader — для тестов и DAO: история точек при закрытии визита.
type visitLocationReader interface {
	GetLocationsByUserBetween(userID int, from, to time.Time) ([]models.Location, error)
}

// VisitEventProcessor отвечает за обработку событий локации из RabbitMQ.
type VisitEventProcessor struct {
	CheckpointService *CheckpointService
	VisitService      *VisitService
	LocationDAO       visitLocationReader
	geofenceStates    *geofenceStateStore
}

// NewVisitEventProcessor создаёт новый экземпляр обработчика событий.
func NewVisitEventProcessor(cs *CheckpointService, vs *VisitService, locationDAO visitLocationReader) *VisitEventProcessor {
	return &VisitEventProcessor{
		CheckpointService: cs,
		VisitService:      vs,
		LocationDAO:       locationDAO,
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
		return vep.handleInside(event.UserID, cp.ID, activeVisit, state, now, event.Source)
	}

	state.clearPendingEnter()
	return vep.handleOutside(event.UserID, cp, activeVisit, state, now, distance)
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
// on_demand (пинг менеджера) — визит сразу, без ожидания grace.
func (vep *VisitEventProcessor) handleInside(
	userID, checkpointID int,
	activeVisit *models.Visit,
	state *geofencePendingState,
	now time.Time,
	source string,
) error {
	if activeVisit != nil {
		log.Printf("[handleInside] Пользователь %d в зоне чекпоинта %d, визит уже активен", userID, checkpointID)
		return nil
	}

	onDemand := source == models.LocationSourceOnDemand
	if !onDemand {
		enterGrace := geofenceEnterGraceSeconds()
		if !state.pendingEnterElapsed(now, enterGrace) {
			state.markPendingEnter(now)
			log.Printf("[handleInside] Ожидание подтверждения входа userID=%d checkpointID=%d (grace=%ds)",
				userID, checkpointID, enterGrace)
			return nil
		}
	}

	state.clearPendingEnter()
	_, err := vep.VisitService.StartVisitAt(userID, checkpointID, now)
	if err != nil {
		log.Printf("[handleInside] Ошибка начала визита для userID=%d, checkpointID=%d: %v", userID, checkpointID, err)
		return err
	}
	if onDemand {
		log.Printf("[handleInside] Начат визит (on_demand) userID=%d checkpointID=%d", userID, checkpointID)
	} else {
		log.Printf("[handleInside] Начат визит для userID=%d checkpointID=%d после grace", userID, checkpointID)
	}
	return nil
}

// handleOutside: при активном визите ждём устойчивого выхода из зоны, затем завершаем.
func (vep *VisitEventProcessor) handleOutside(
	userID int,
	cp models.Checkpoint,
	activeVisit *models.Visit,
	state *geofencePendingState,
	now time.Time,
	distance float64,
) error {
	if activeVisit == nil {
		return nil
	}

	checkpointID := cp.ID
	farOutside := geofenceFarOutside(distance, cp.Radius, true)
	exitGrace := geofenceExitGraceSeconds()
	if !farOutside && !state.pendingExitElapsed(now, exitGrace) {
		state.markPendingExit(now)
		log.Printf("[handleOutside] Ожидание подтверждения выхода userID=%d checkpointID=%d (grace=%ds)",
			userID, checkpointID, exitGrace)
		return nil
	}

	state.clearPendingExit()

	minVisit := geofenceMinVisitSeconds()
	endAt := vep.resolveVisitEndAt(userID, cp, activeVisit, now, farOutside)
	elapsed := int(endAt.Sub(activeVisit.StartAt.UTC()).Seconds())
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

	if err := vep.VisitService.EndVisitAt(activeVisit, endAt); err != nil {
		log.Printf("[handleOutside] Ошибка завершения визита userID=%d checkpointID=%d: %v",
			userID, checkpointID, err)
		return err
	}
	log.Printf("[handleOutside] Завершён визит для userID=%d checkpointID=%d endAt=%s farOutside=%v",
		userID, checkpointID, endAt.UTC(), farOutside)
	return nil
}

// resolveVisitEndAt не растягивает визит на период без GPS: конец = последняя точка в зоне + grace.
func (vep *VisitEventProcessor) resolveVisitEndAt(
	userID int,
	cp models.Checkpoint,
	visit *models.Visit,
	eventNow time.Time,
	farOutside bool,
) time.Time {
	eventNow = eventNow.UTC()
	exitGrace := time.Duration(geofenceExitGraceSeconds()) * time.Second
	staleGap := time.Duration(geofenceStaleGapSeconds()) * time.Second

	if vep.LocationDAO == nil {
		return eventNow
	}

	locs, err := vep.LocationDAO.GetLocationsByUserBetween(userID, visit.StartAt.UTC(), eventNow)
	if err != nil || len(locs) == 0 {
		return eventNow
	}

	insideRadius := cp.Radius + geofenceExitBufferMeters()
	var lastInside time.Time
	foundInside := false
	for _, loc := range locs {
		d := haversineDistance(loc.Latitude, loc.Longitude, cp.Latitude, cp.Longitude)
		if d <= insideRadius {
			lastInside = loc.EffectiveAt().UTC()
			foundInside = true
		}
	}

	if !foundInside {
		return eventNow
	}

	gap := eventNow.Sub(lastInside)
	if !farOutside && gap <= staleGap {
		return eventNow
	}

	endAt := lastInside.Add(exitGrace)
	if endAt.After(eventNow) {
		endAt = eventNow
	}
	if endAt.Before(visit.StartAt.UTC()) {
		endAt = visit.StartAt.UTC()
	}
	log.Printf("[resolveVisitEndAt] userID=%d checkpointID=%d lastInside=%s gap=%s endAt=%s",
		userID, cp.ID, lastInside, gap, endAt)
	return endAt
}
