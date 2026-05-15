package service

import (
	"errors"
	"locator/dao"
	"locator/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrLocationRequestNotFound   = errors.New("location request not found")
	ErrLocationRequestWrongUser  = errors.New("location request belongs to another user")
	ErrLocationRequestNotPending = errors.New("location request is not pending")
)

const locationRequestPendingTTL = 15 * time.Minute

// LocationRequestService — on-demand запросы координат с устройства.
type LocationRequestService struct {
	DAO *dao.LocationRequestDAO
}

func NewLocationRequestService(dao *dao.LocationRequestDAO) *LocationRequestService {
	return &LocationRequestService{DAO: dao}
}

func (svc *LocationRequestService) expireStale() error {
	cutoff := time.Now().Add(-locationRequestPendingTTL)
	return svc.DAO.ExpirePendingOlderThan(cutoff)
}

// CreatePending создаёт новый pending-запрос, отменяя предыдущие для пользователя.
func (svc *LocationRequestService) CreatePending(userID int) (*models.LocationRequest, error) {
	_ = svc.expireStale()

	id := uuid.New().String()
	if err := svc.DAO.CancelPendingForUser(userID, id); err != nil {
		return nil, err
	}

	req := &models.LocationRequest{
		ID:     id,
		UserID: userID,
		Status: models.LocationRequestStatusPending,
	}
	if err := svc.DAO.Create(req); err != nil {
		return nil, err
	}
	return req, nil
}

// PollPending возвращает активный pending-запрос или nil.
func (svc *LocationRequestService) PollPending(userID int) (*models.LocationRequest, error) {
	_ = svc.expireStale()

	req, err := svc.DAO.GetPendingByUserID(userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return req, nil
}

// GetByID возвращает запрос по ID.
func (svc *LocationRequestService) GetByID(id string) (*models.LocationRequest, error) {
	req, err := svc.DAO.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrLocationRequestNotFound
	}
	if err != nil {
		return nil, err
	}
	return req, nil
}

// Complete помечает запрос выполненным после получения координат.
func (svc *LocationRequestService) Complete(requestID string, userID int) error {
	req, err := svc.GetByID(requestID)
	if err != nil {
		return err
	}
	if req.UserID != userID {
		return ErrLocationRequestWrongUser
	}
	if req.Status != models.LocationRequestStatusPending {
		return ErrLocationRequestNotPending
	}
	now := time.Now()
	return svc.DAO.UpdateStatus(requestID, models.LocationRequestStatusCompleted, &now)
}
