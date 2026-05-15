package controllers

import (
	"errors"
	"locator/models"
	"locator/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// LocationRequestController — on-demand запросы локации.
type LocationRequestController struct {
	RequestService *service.LocationRequestService
	CommandService *service.DeviceCommandService
}

func NewLocationRequestController(requestService *service.LocationRequestService, commandService *service.DeviceCommandService) *LocationRequestController {
	return &LocationRequestController{
		RequestService: requestService,
		CommandService: commandService,
	}
}

// PostLocationRequest создаёт pending-запрос для user_id (admin).
// POST /api/location/request  {"user_id": 5}
func (rc *LocationRequestController) PostLocationRequest(ctx *gin.Context) {
	currentUser, ok := getCurrentUserFromContext(ctx)
	if !ok {
		return
	}
	if !currentUser.IsAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Требуются права администратора"})
		return
	}

	var body struct {
		UserID int `json:"user_id" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil || body.UserID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Укажите user_id"})
		return
	}

	var cmdID string
	if rc.CommandService != nil {
		cmd, err := rc.CommandService.EnqueueCommand(body.UserID, models.DeviceCommandTypeLocationRequest, nil)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать запрос"})
			return
		}
		cmdID = cmd.ID
		payload, _ := service.CommandPayloadMap(cmd)
		ctx.JSON(http.StatusAccepted, gin.H{
			"request_id": payload["request_id"],
			"command_id": cmd.ID,
			"status":     cmd.Status,
			"user_id":    body.UserID,
		})
		return
	}

	req, err := rc.RequestService.CreatePending(body.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать запрос"})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{
		"request_id": req.ID,
		"command_id": cmdID,
		"status":     req.Status,
		"user_id":    req.UserID,
	})
}

// PollLocationRequest — устройство забирает pending-запрос.
// GET /api/location/request → 204 или 200 {"request_id":"..."}
func (rc *LocationRequestController) PollLocationRequest(ctx *gin.Context) {
	currentUser, ok := getCurrentUserFromContext(ctx)
	if !ok {
		return
	}

	req, err := rc.RequestService.PollPending(currentUser.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка опроса запроса"})
		return
	}
	if req == nil {
		if ctx.Query("json") == "1" || ctx.Query("json") == "true" {
			ctx.JSON(http.StatusOK, gin.H{"pending": false})
			return
		}
		ctx.Status(http.StatusNoContent)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"request_id": req.ID,
		"pending":    true,
	})
}

// GetLocationRequestStatus — статус запроса по request_id (admin).
// GET /api/location/request/:request_id
func (rc *LocationRequestController) GetLocationRequestStatus(ctx *gin.Context) {
	currentUser, ok := getCurrentUserFromContext(ctx)
	if !ok {
		return
	}
	if !currentUser.IsAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Требуются права администратора"})
		return
	}

	requestID := ctx.Param("request_id")
	if requestID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "request_id обязателен"})
		return
	}

	req, err := rc.RequestService.GetByID(requestID)
	if err != nil {
		if errors.Is(err, service.ErrLocationRequestNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Запрос не найден"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения запроса"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"request_id":   req.ID,
		"user_id":      req.UserID,
		"status":       req.Status,
		"created_at":   req.CreatedAt,
		"completed_at": req.CompletedAt,
	})
}
