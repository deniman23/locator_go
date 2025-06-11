package service

import (
	"locator/dao"
	"locator/models"
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
	cp := &models.Checkpoint{
		// ID не задаём, так как оно будет автоинкрементным (устанавливается базой данных)
		Name:      name,
		Latitude:  lat,
		Longitude: lon,
		Radius:    radius,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := svc.DAO.Create(cp); err != nil {
		return nil, err
	}
	return cp, nil
}

// GetCheckpoints возвращает все чекпоинты.
func (svc *CheckpointService) GetCheckpoints() ([]models.Checkpoint, error) {
	return svc.DAO.GetAll()
}

// GetCheckpointByID возвращает чекпоинт по его ID.
func (svc *CheckpointService) GetCheckpointByID(id int) (*models.Checkpoint, error) {
	return svc.DAO.GetByID(id)
}

// IsLocationInCheckpoint проверяет, находится ли заданная локация внутри данного чекпоинта.
// От расстояния (вычисляемого по формуле Хаверсина) зависит, считается ли локация находящейся в зоне чекпоинта.
func (svc *CheckpointService) IsLocationInCheckpoint(loc *models.Location, checkpoint *models.Checkpoint) bool {
	distance := haversineDistance(loc.Latitude, loc.Longitude, checkpoint.Latitude, checkpoint.Longitude)
	return distance <= checkpoint.Radius
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
