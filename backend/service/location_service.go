package service

import (
	"fmt"
	"locator/dao"
	"locator/models"
	"log"
	"time"
)

// LocationService отвечает за бизнес-логику, связанную с операциями над местоположениями.
type LocationService struct {
	DAO *dao.LocationDAO
}

// NewLocationService создаёт новый экземпляр сервиса.
func NewLocationService(dao *dao.LocationDAO) *LocationService {
	return &LocationService{DAO: dao}
}

// GetLocation получает данные о местоположении для заданного пользователя.
func (svc *LocationService) GetLocation(userID int) (*models.Location, error) {
	log.Printf("[GetLocation] Запрос на получение местоположения для userID=%d", userID)
	location, err := svc.DAO.GetByUserID(userID)
	if err != nil {
		log.Printf("[GetLocation] Ошибка при получении местоположения для userID=%d: %v", userID, err)
		return nil, err
	}
	log.Printf("[GetLocation] Запись о местоположении получена для userID=%d: Latitude=%.6f, Longitude=%.6f", userID, location.Latitude, location.Longitude)
	return location, nil
}

// CreateLocation создаёт новую запись о местоположении без обновления существующей.
func (svc *LocationService) CreateLocation(userID int, lat, lon float64) (*models.Location, error) {
	log.Printf("[CreateLocation] Создание записи местоположения: userID=%d, Latitude=%.6f, Longitude=%.6f", userID, lat, lon)
	newLocation := models.NewLocation(userID, lat, lon)
	if err := svc.DAO.Create(newLocation); err != nil {
		log.Printf("[CreateLocation] Ошибка при создании записи для userID=%d: %v", userID, err)
		return nil, err
	}
	log.Printf("[CreateLocation] Запись успешно создана для userID=%d: Latitude=%.6f, Longitude=%.6f", userID, lat, lon)
	return newLocation, nil
}

// CreateOrUpdateLocation создаёт новую запись или обновляет существующую.
func (svc *LocationService) CreateOrUpdateLocation(userID int, lat, lon float64) (*models.Location, error) {
	log.Printf("[CreateOrUpdateLocation] Инициирование создания или обновления записи: userID=%d, Latitude=%.6f, Longitude=%.6f", userID, lat, lon)
	loc, err := svc.DAO.GetByUserID(userID)
	if err != nil {
		log.Printf("[CreateOrUpdateLocation] Запись для userID=%d не найдена, создаём новую: %v", userID, err)
		loc = models.NewLocation(userID, lat, lon)
		if err := svc.DAO.Create(loc); err != nil {
			log.Printf("[CreateOrUpdateLocation] Ошибка при создании записи для userID=%d: %v", userID, err)
			return nil, err
		}
		log.Printf("[CreateOrUpdateLocation] Запись успешно создана для userID=%d: Latitude=%.6f, Longitude=%.6f", userID, lat, lon)
	} else {
		log.Printf("[CreateOrUpdateLocation] Запись найдена для userID=%d. Выполняется обновление координат.", userID)
		loc.UpdateCoordinates(lat, lon)
		if err := svc.DAO.Update(loc); err != nil {
			log.Printf("[CreateOrUpdateLocation] Ошибка при обновлении записи для userID=%d: %v", userID, err)
			return nil, err
		}
		log.Printf("[CreateOrUpdateLocation] Запись успешно обновлена для userID=%d: Latitude=%.6f, Longitude=%.6f", userID, lat, lon)
	}
	return loc, nil
}

// GetLocations возвращает все записи о местоположениях.
func (svc *LocationService) GetLocations() ([]models.Location, error) {
	log.Println("[GetLocations] Запрос на получение всех записей местоположений")
	locations, err := svc.DAO.GetAll()
	if err != nil {
		log.Printf("[GetLocations] Ошибка при получении записей местоположений: %v", err)
		return nil, err
	}
	log.Printf("[GetLocations] Найдено %d записей местоположений", len(locations))
	return locations, nil
}

// GetLocationsBetween возвращает все записи, созданные между заданными временными метками.
// Параметры from и to должны быть в формате RFC3339, например: "2025-09-19T10:00:00Z".
func (svc *LocationService) GetLocationsBetween(fromStr, toStr string) ([]models.Location, error) {
	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		return nil, fmt.Errorf("неверный формат параметра 'from': %v", err)
	}
	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		return nil, fmt.Errorf("неверный формат параметра 'to': %v", err)
	}
	if from.After(to) {
		return nil, fmt.Errorf("начало интервала не может быть позже окончания")
	}
	return svc.DAO.GetLocationsBetween(from, to)
}
