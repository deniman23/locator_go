package service

import (
	"encoding/json"
	"errors"
	"locator/dao"
	"locator/models"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrDeviceCommandNotFound    = errors.New("device command not found")
	ErrDeviceCommandWrongUser   = errors.New("device command belongs to another user")
	ErrDeviceCommandInvalidType = errors.New("device command type is invalid")
)

const deviceCommandPendingTTL = 15 * time.Minute
const deviceCommandAppUpdateTTL = 48 * time.Hour

var allowedDeviceCommandTypes = map[string]struct{}{
	models.DeviceCommandTypeLocationRequest: {},
	models.DeviceCommandTypeHealthCheck:     {},
	models.DeviceCommandTypeConfigUpdate:    {},
	models.DeviceCommandTypeAppUpdate:       {},
}

// DeviceCommandService — очередь команд для мобильного коннектора.
type DeviceCommandService struct {
	DAO              *dao.DeviceCommandDAO
	LocationRequests *LocationRequestService
}

func NewDeviceCommandService(dao *dao.DeviceCommandDAO, locationRequests *LocationRequestService) *DeviceCommandService {
	return &DeviceCommandService{DAO: dao, LocationRequests: locationRequests}
}

func (svc *DeviceCommandService) expireStale() error {
	now := time.Now()
	if err := svc.DAO.ExpirePendingOlderThanExceptType(
		now.Add(-deviceCommandPendingTTL),
		models.DeviceCommandTypeAppUpdate,
	); err != nil {
		return err
	}
	return svc.DAO.ExpirePendingOlderThanType(
		now.Add(-deviceCommandAppUpdateTTL),
		models.DeviceCommandTypeAppUpdate,
	)
}

// EnqueueCommand ставит команду в очередь для пользователя.
func (svc *DeviceCommandService) EnqueueCommand(userID int, cmdType string, payload map[string]interface{}) (*models.DeviceCommand, error) {
	if _, ok := allowedDeviceCommandTypes[cmdType]; !ok {
		return nil, ErrDeviceCommandInvalidType
	}

	if payload == nil {
		payload = make(map[string]interface{})
	}

	if cmdType == models.DeviceCommandTypeLocationRequest && svc.LocationRequests != nil {
		req, err := svc.LocationRequests.CreatePending(userID)
		if err != nil {
			return nil, err
		}
		payload["request_id"] = req.ID
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	id := uuid.New().String()
	if err := svc.DAO.CancelPendingForUser(userID, cmdType, id); err != nil {
		return nil, err
	}

	cmd := &models.DeviceCommand{
		ID:      id,
		UserID:  userID,
		Type:    cmdType,
		Payload: datatypes.JSON(payloadBytes),
		Status:  models.DeviceCommandStatusPending,
	}
	if err := svc.DAO.Create(cmd); err != nil {
		return nil, err
	}
	return cmd, nil
}

// Poll возвращает следующую команду и помечает её доставленной.
func (svc *DeviceCommandService) Poll(userID int) (*models.DeviceCommand, error) {
	_ = svc.expireStale()

	cmd, err := svc.DAO.GetNextPending(userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if err := svc.DAO.MarkDelivered(cmd.ID, now); err != nil {
		return nil, err
	}
	cmd.Status = models.DeviceCommandStatusDelivered
	cmd.DeliveredAt = &now
	return cmd, nil
}

// appUpdateProgressAck — промежуточные статусы OTA с телефона (не ошибка и не финал).
var appUpdateProgressAck = map[string]struct{}{
	"accepted":   {},
	"downloaded": {},
	"installing": {},
}

// Ack подтверждает выполнение или ошибку команды на устройстве.
func (svc *DeviceCommandService) Ack(commandID string, userID int, status, message string) error {
	cmd, err := svc.DAO.GetByID(commandID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrDeviceCommandNotFound
	}
	if err != nil {
		return err
	}
	if cmd.UserID != userID {
		return ErrDeviceCommandWrongUser
	}

	now := time.Now()
	status = strings.TrimSpace(strings.ToLower(status))
	success := status == "ok" || status == "success"
	if success {
		if err := svc.DAO.MarkAcked(commandID, status, message, now); err != nil {
			return err
		}
	} else if cmd.Type == models.DeviceCommandTypeAppUpdate {
		if _, progress := appUpdateProgressAck[status]; progress {
			if err := svc.DAO.MarkProgress(commandID, status, message, now); err != nil {
				return err
			}
		} else if err := svc.DAO.MarkFailed(commandID, status, message, now); err != nil {
			return err
		}
	} else if err := svc.DAO.MarkFailed(commandID, status, message, now); err != nil {
		return err
	}

	if success && cmd.Type == models.DeviceCommandTypeLocationRequest && svc.LocationRequests != nil {
		requestID := commandID
		if payload, err := CommandPayloadMap(cmd); err == nil {
			if rid, ok := payload["request_id"].(string); ok && rid != "" {
				requestID = rid
			}
		}
		_ = svc.LocationRequests.Complete(requestID, userID)
	}
	return nil
}

// CompleteLinkedLocationRequest завершает запрос координат: телефон может прислать
// request_id из location_requests или id команды device (старые клиенты).
func (svc *DeviceCommandService) CompleteLinkedLocationRequest(requestID string, userID int) error {
	if requestID == "" || svc.LocationRequests == nil {
		return nil
	}
	err := svc.LocationRequests.Complete(requestID, userID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrLocationRequestNotFound) {
		return err
	}
	cmd, cmdErr := svc.DAO.GetByID(requestID)
	if errors.Is(cmdErr, gorm.ErrRecordNotFound) {
		return err
	}
	if cmdErr != nil {
		return cmdErr
	}
	if cmd.UserID != userID || cmd.Type != models.DeviceCommandTypeLocationRequest {
		return err
	}
	payload, pErr := CommandPayloadMap(cmd)
	if pErr != nil {
		return err
	}
	rid, _ := payload["request_id"].(string)
	if rid == "" {
		return err
	}
	return svc.LocationRequests.Complete(rid, userID)
}

// CommandPayloadMap разбирает JSON payload команды в map.
func CommandPayloadMap(cmd *models.DeviceCommand) (map[string]interface{}, error) {
	if cmd == nil || len(cmd.Payload) == 0 {
		return map[string]interface{}{}, nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal(cmd.Payload, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]interface{}{}
	}
	return out, nil
}
