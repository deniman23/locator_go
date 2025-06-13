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

// ProcessEvent обрабатывает событие из RabbitMQ.
// Обрабатывая данные о локации пользователя, мы проверяем, находится ли он в радиусе каждого чекпоинта.
// Если пользователь входит в зону — запускаем визит, если активный визит ещё не существует.
// Если пользователь выходит из зоны чекпоинта, завершаем активный визит.
func (vep *VisitEventProcessor) ProcessEvent(message []byte) error {
	var event models.LocationEvent
	if err := json.Unmarshal(message, &event); err != nil {
		log.Printf("Ошибка десериализации события: %v", err)
		return err
	}

	// Получаем список всех чекпоинтов.
	checkpoints, err := vep.CheckpointService.GetCheckpoints()
	if err != nil {
		log.Printf("Ошибка получения чекпоинтов: %v", err)
		return err
	}

	// Для каждого чекпоинта выполняем проверку.
	for _, cp := range checkpoints {
		// Проверяем, находится ли локация пользователя внутри зоны данного чекпоинта.
		inZone := vep.CheckpointService.IsLocationInCheckpoint(
			&models.Location{
				Latitude:  event.Latitude,
				Longitude: event.Longitude,
			},
			&cp, // Передаём указатель на переменную cp.
		)

		// Получаем активный визит (если он есть) для данного пользователя и чекпоинта.
		activeVisit, err := vep.VisitService.GetActiveVisit(event.UserID, cp.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Ошибка получения активного визита: %v", err)
			return err
		}

		if inZone {
			// Если пользователь находится в зоне и активного визита нет, создаём новый визит.
			if activeVisit == nil {
				_, err := vep.VisitService.StartVisit(event.UserID, cp.ID)
				if err != nil {
					log.Printf("Ошибка начала визита: %v", err)
					return err
				}
				log.Printf("Начат новый визит для пользователя %d на чекпоинте %d", event.UserID, cp.ID)
			}
			// Если активный визит уже существует, можно обновить его параметры (например, время последнего обновления),
			// если это требуется бизнес-логикой. Здесь этот момент оставляем по умолчанию.
		} else {
			// Если пользователь не находится в зоне и активный визит существует, завершаем его.
			if activeVisit != nil {
				if err := vep.VisitService.EndVisit(activeVisit); err != nil {
					log.Printf("Ошибка завершения визита: %v", err)
					return err
				}
				log.Printf("Завершён визит для пользователя %d на чекпоинте %d", event.UserID, cp.ID)
			}
		}
	}

	return nil
}
