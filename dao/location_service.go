package dao

import (
	"locator/location"
	"math"
)

// LocationService отвечает за бизнес-логику, связанную с операциями над местоположениями.
type LocationService struct {
	DAO *LocationDAO
}

// NewLocationService создаёт новый экземпляр сервиса.
func NewLocationService(dao *LocationDAO) *LocationService {
	return &LocationService{DAO: dao}
}

// GetLocation получает данные о местоположении для заданного пользователя.
func (svc *LocationService) GetLocation(userID int) (*location.Location, error) {
	// Здесь может быть дополнительная бизнес-логика.
	return svc.DAO.GetByUserID(userID)
}

// CreateOrUpdateLocation создаёт новую запись или обновляет существующую.
func (svc *LocationService) CreateOrUpdateLocation(userID int, lat, lon float64) (*location.Location, error) {
	loc, err := svc.DAO.GetByUserID(userID)
	if err != nil {
		// Если запись не найдена, создаём новую.
		loc = location.NewLocation(userID, lat, lon)
		if err := svc.DAO.Create(loc); err != nil {
			return nil, err
		}
	} else {
		// Если запись найдена, обновляем координаты.
		loc.UpdateCoordinates(lat, lon)
		if err := svc.DAO.Update(loc); err != nil {
			return nil, err
		}
	}
	return loc, nil
}

// GetLocations возвращает все записи о местоположениях.
func (svc *LocationService) GetLocations() ([]location.Location, error) {
	return svc.DAO.GetAll()
}

// IsUserInCheckpoint проверяет, находится ли пользователь с заданным userID в указанном чекпоинте.
// Для этого:
//  1. Получаем текущее местоположение пользователя.
//  2. Расчитываем расстояние между точкой местоположения и координатами чекпоинта с помощью формулы Хаверсина.
//  3. Если расстояние меньше или равно радиусу чекпоинта, считаем, что пользователь находится на чекпоинте.
func (svc *LocationService) IsUserInCheckpoint(userID int, checkpoint *location.Checkpoint) (bool, error) {
	loc, err := svc.GetLocation(userID)
	if err != nil {
		return false, err
	}
	if loc == nil {
		// Если данных о местоположении для пользователя нет, считаем, что он не находится в чекпоинте.
		return false, nil
	}

	distance := haversineDistance(loc.Latitude, loc.Longitude, checkpoint.Latitude, checkpoint.Longitude)
	// Если расстояние меньше или равно радиусу чекпоинта, пользователь считается находящимся в зоне.
	return distance <= checkpoint.Radius, nil
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
