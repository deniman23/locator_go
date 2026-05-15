package controllers

import (
	"errors"
	"locator/models"
	"locator/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// DeviceController — poll, отчёты и ack для мобильного коннектора.
type DeviceController struct {
	CommandService *service.DeviceCommandService
	ReportService  *service.DeviceReportService
	RequestService *service.LocationRequestService
}

func NewDeviceController(
	commandService *service.DeviceCommandService,
	reportService *service.DeviceReportService,
	requestService *service.LocationRequestService,
) *DeviceController {
	return &DeviceController{
		CommandService: commandService,
		ReportService:  reportService,
		RequestService: requestService,
	}
}

// PollDevice — GET /api/device/poll
func (dc *DeviceController) PollDevice(ctx *gin.Context) {
	currentUser, ok := getCurrentUserFromContext(ctx)
	if !ok {
		return
	}

	cmd, err := dc.CommandService.Poll(currentUser.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка опроса команд"})
		return
	}

	if cmd == nil && dc.RequestService != nil {
		locReq, err := dc.RequestService.PollPending(currentUser.ID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка опроса"})
			return
		}
		if locReq != nil {
			ctx.JSON(http.StatusOK, gin.H{
				"command": gin.H{
					"id":   locReq.ID,
					"type": models.DeviceCommandTypeLocationRequest,
					"payload": gin.H{
						"request_id": locReq.ID,
					},
				},
			})
			return
		}
	}

	if cmd == nil {
		if ctx.Query("json") == "1" || ctx.Query("json") == "true" {
			ctx.JSON(http.StatusOK, gin.H{"pending": false})
			return
		}
		ctx.Status(http.StatusNoContent)
		return
	}

	payload, err := service.CommandPayloadMap(cmd)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка payload команды"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"command": gin.H{
			"id":      cmd.ID,
			"type":    cmd.Type,
			"payload": payload,
		},
	})
}

// PostDeviceReport — POST /api/device/report
func (dc *DeviceController) PostDeviceReport(ctx *gin.Context) {
	currentUser, ok := getCurrentUserFromContext(ctx)
	if !ok {
		return
	}

	var body map[string]interface{}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Некорректное тело отчёта"})
		return
	}

	report, err := dc.ReportService.SaveReport(currentUser.ID, body)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сохранить отчёт"})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"id":         report.ID,
		"user_id":    report.UserID,
		"created_at": report.CreatedAt,
	})
}

// PostCommandAck — POST /api/device/command/ack
func (dc *DeviceController) PostCommandAck(ctx *gin.Context) {
	currentUser, ok := getCurrentUserFromContext(ctx)
	if !ok {
		return
	}

	var body struct {
		CommandID string `json:"command_id" binding:"required"`
		Status    string `json:"status" binding:"required"`
		Message   string `json:"message"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Укажите command_id и status"})
		return
	}

	if err := dc.CommandService.Ack(body.CommandID, currentUser.ID, body.Status, body.Message); err != nil {
		if errors.Is(err, service.ErrDeviceCommandNotFound) && dc.RequestService != nil &&
			(body.Status == "ok" || body.Status == "success") {
			if completeErr := dc.RequestService.Complete(body.CommandID, currentUser.ID); completeErr == nil {
				ctx.JSON(http.StatusOK, gin.H{"ok": true, "legacy": true})
				return
			}
		}
		switch {
		case errors.Is(err, service.ErrDeviceCommandNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Команда не найдена"})
		case errors.Is(err, service.ErrDeviceCommandWrongUser):
			ctx.JSON(http.StatusForbidden, gin.H{"error": "Команда принадлежит другому пользователю"})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка подтверждения"})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

// GetUserHealth — GET /api/users/:id/health (admin)
func (dc *DeviceController) GetUserHealth(ctx *gin.Context) {
	currentUser, ok := getCurrentUserFromContext(ctx)
	if !ok {
		return
	}
	if !currentUser.IsAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Требуются права администратора"})
		return
	}

	userID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
		return
	}

	report, err := dc.ReportService.GetLatestByUserID(userID)
	if err != nil {
		if errors.Is(err, service.ErrDeviceReportNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Отчёты не найдены"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения отчёта"})
		return
	}

	reportMap, _ := service.ReportAsMap(report)
	issues, _ := service.IssuesSlice(report)

	ctx.JSON(http.StatusOK, gin.H{
		"user_id":        userID,
		"last_report_at": report.CreatedAt,
		"app_version":    report.AppVersion,
		"platform":       report.Platform,
		"issues":         issues,
		"issue_count":    len(issues),
		"healthy":        len(issues) == 0,
		"report":         reportMap,
	})
}

// PostAdminUserCommand — POST /api/admin/users/:id/commands
func (dc *DeviceController) PostAdminUserCommand(ctx *gin.Context) {
	currentUser, ok := getCurrentUserFromContext(ctx)
	if !ok {
		return
	}
	if !currentUser.IsAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Требуются права администратора"})
		return
	}

	userID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
		return
	}

	var body struct {
		Type    string                 `json:"type" binding:"required"`
		Payload map[string]interface{} `json:"payload"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Укажите type"})
		return
	}

	cmd, err := dc.CommandService.EnqueueCommand(userID, body.Type, body.Payload)
	if err != nil {
		if errors.Is(err, service.ErrDeviceCommandInvalidType) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Неизвестный type команды"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать команду"})
		return
	}

	payload, _ := service.CommandPayloadMap(cmd)
	ctx.JSON(http.StatusAccepted, gin.H{
		"command_id": cmd.ID,
		"type":       cmd.Type,
		"status":     cmd.Status,
		"user_id":    cmd.UserID,
		"payload":    payload,
	})
}
