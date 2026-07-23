package service

import (
	"time"

	"locator/models"
)

// Narrow repository interfaces so unit tests can inject in-memory fakes
// while production keeps using concrete *dao.*DAO (implicit satisfaction).

type userRepository interface {
	Create(user *models.User) error
	Update(user *models.User) error
	GetByID(id int) (*models.User, error)
	GetAll() ([]models.User, error)
}

type locationRepository interface {
	GetByUserID(userID int) (*models.Location, error)
	GetPreviousByEffectiveTime(userID int, before time.Time) (*models.Location, error)
	UserExists(userID int) (bool, error)
	Create(location *models.Location) error
	GetAll() ([]models.Location, error)
	GetLocationsBetween(from, to time.Time) ([]models.Location, error)
	ListUserIDsWithoutCapturedAt() ([]int, error)
	GetWithoutCapturedAtByUser(userID int) ([]models.Location, error)
	UpdateCapturedAt(id int, capturedAt time.Time) error
}

type visitRepository interface {
	Create(visit *models.Visit) error
	Update(visit *models.Visit) error
	Delete(id int64) error
	GetActiveVisit(userID int, checkpointID int) (*models.Visit, error)
	GetVisits(filters map[string]interface{}, activeOnly bool, rangeFrom, rangeTo *time.Time) ([]models.Visit, error)
}

type checkpointRepository interface {
	Create(cp *models.Checkpoint) error
	Update(cp *models.Checkpoint) error
	GetByID(id int) (*models.Checkpoint, error)
	GetAll() ([]models.Checkpoint, error)
}
