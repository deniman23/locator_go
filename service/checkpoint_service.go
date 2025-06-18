package service

import (
	"locator/dao"
	"locator/models"
	"log"
	"math"
	"time"
)

// CheckpointService отвечает за бизнес-логику, связанную с операциями над чекпоинтами.
type CheckpointService struct {
	DAO *dao.CheckpointDAO
}

// NewCheckpointService создаёт новый экземпляр CheckpointService.
func NewCheckpointService(dao *dao.CheckpointDAO) *CheckpointService {
	return &CheckpointService{DAO: dao}
}

// CreateCheckpoint создаёт новый чекпоинт с заданными параметрами.
func (svc *CheckpointService) CreateCheckpoint(name string, lat, lon, radius float64) (*models.Checkpoint, error) {
	log.Printf("[CreateCheckpoint] Создание чекпоинта: Name=%s, Latitude=%.6f, Longitude=%.6f, Radius=%.2f м",
		name, lat, lon, radius)
	cp := &models.Checkpoint{
		Name:      name,
		Latitude:  lat,
		Longitude: lon,
		Radius:    radius,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := svc.DAO.Create(cp); err != nil {
		log.Printf("[CreateCheckpoint] Ошибка при создании чекпоинта (Name=%s): %v", name, err)
		return nil, err
	}
	log.Printf("[CreateCheckpoint] Чекпоинт успешно создан: ID=%d, Name=%s, Latitude=%.6f, Longitude=%.6f, Radius=%.2f м",
		cp.ID, cp.Name, cp.Latitude, cp.Longitude, cp.Radius)
	return cp, nil
}

// GetCheckpoints возвращает все чекпоинты.
func (svc *CheckpointService) GetCheckpoints() ([]models.Checkpoint, error) {
	log.Println("[GetCheckpoints] Запрос на получение всех чекпоинтов")
	checkpoints, err := svc.DAO.GetAll()
	if err != nil {
		log.Printf("[GetCheckpoints] Ошибка получения чекпоинтов: %v", err)
		return nil, err
	}
	log.Printf("[GetCheckpoints] Найдено %d чекпоинтов", len(checkpoints))
	return checkpoints, nil
}

// GetCheckpointByID возвращает чекпоинт по его ID.
func (svc *CheckpointService) GetCheckpointByID(id int) (*models.Checkpoint, error) {
	log.Printf("[GetCheckpointByID] Запрос на получение чекпоинта с ID: %d", id)
	cp, err := svc.DAO.GetByID(id)
	if err != nil {
		log.Printf("[GetCheckpointByID] Ошибка получения чекпоинта с ID=%d: %v", id, err)
		return nil, err
	}
	log.Printf("[GetCheckpointByID] Чекпоинт найден: ID=%d, Name=%s, Latitude=%.6f, Longitude=%.6f, Radius=%.2f м",
		cp.ID, cp.Name, cp.Latitude, cp.Longitude, cp.Radius)
	return cp, nil
}

// IsLocationInCheckpoint проверяет, находится ли заданная локация внутри данного чекпоинта.
// Расстояние вычисляется с использованием формулы Хаверсина.
func (svc *CheckpointService) IsLocationInCheckpoint(loc *models.Location, checkpoint *models.Checkpoint) bool {
	distance := haversineDistance(loc.Latitude, loc.Longitude, checkpoint.Latitude, checkpoint.Longitude)
	log.Printf("[IsLocationInCheckpoint] Проверка попадания локации для Чекпоинта ID=%d: Локация (%.6f, %.6f), Чекпоинт (%.6f, %.6f); Вычисленная дистанция: %.2f м, Радиус: %.2f м",
		checkpoint.ID, loc.Latitude, loc.Longitude, checkpoint.Latitude, checkpoint.Longitude, distance, checkpoint.Radius)
	inZone := distance <= checkpoint.Radius
	if inZone {
		log.Printf("[IsLocationInCheckpoint] Локация находится внутри зоны Чекпоинта ID=%d", checkpoint.ID)
	} else {
		log.Printf("[IsLocationInCheckpoint] Локация находится вне зоны Чекпоинта ID=%d", checkpoint.ID)
	}
	return inZone
}

// haversineDistance вычисляет расстояние (в метрах) между двумя точками,
// заданными широтой и долготой, с использованием формулы Хаверсина.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000 // Радиус Земли в метрах

	// Перевод градусов в радианы.
	rLat1 := lat1 * math.Pi / 180
	rLat2 := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(rLat1)*math.Cos(rLat2)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
