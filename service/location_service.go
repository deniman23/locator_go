package service

import (
	"locator/dao"
	"locator/models"
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
	// Здесь может быть дополнительная бизнес-логика.
	return svc.DAO.GetByUserID(userID)
}

// CreateOrUpdateLocation создаёт новую запись или обновляет существующую.
func (svc *LocationService) CreateOrUpdateLocation(userID int, lat, lon float64) (*models.Location, error) {
	loc, err := svc.DAO.GetByUserID(userID)
	if err != nil {
		// Если запись не найдена, создаём новую.
		loc = models.NewLocation(userID, lat, lon)
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
func (svc *LocationService) GetLocations() ([]models.Location, error) {
	return svc.DAO.GetAll()
}
