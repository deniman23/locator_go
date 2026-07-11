package controllers

import (
	"errors"
	"locator/models"
	"locator/service"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

// DeviceController — poll, отчёты и ack для мобильного коннектора.
type DeviceController struct {
	CommandService    *service.DeviceCommandService
	ReportService     *service.DeviceReportService
	StatusService     *service.DeviceStatusService
	RequestService    *service.LocationRequestService
	ReleaseController *AppReleaseController
}

func NewDeviceController(
	commandService *service.DeviceCommandService,
	reportService *service.DeviceReportService,
	statusService *service.DeviceStatusService,
	requestService *service.LocationRequestService,
	releaseController *AppReleaseController,
) *DeviceController {
	return &DeviceController{
		CommandService:    commandService,
		ReportService:     reportService,
		StatusService:     statusService,
		RequestService:    requestService,
		ReleaseController: releaseController,
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

// GetAdminDevicesStatus — GET /api/admin/devices/status
// Пакетная сводка GPS + health для всех пользователей (вместо N×2 запросов из админки).
func (dc *DeviceController) GetAdminDevicesStatus(ctx *gin.Context) {
	currentUser, ok := getCurrentUserFromContext(ctx)
	if !ok {
		return
	}
	if !currentUser.IsAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Требуются права администратора"})
		return
	}
	if dc.StatusService == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "Сервис статусов не настроен"})
		return
	}

	summary, err := dc.StatusService.AllUsersSummary()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения статусов"})
		return
	}

	users := make(map[string]service.UserDeviceStatusSummary, len(summary))
	for id, s := range summary {
		users[strconv.Itoa(id)] = s
	}
	ctx.JSON(http.StatusOK, gin.H{"users": users})
}

// PostAdminWakeDevice — POST /api/admin/users/:id/wake
// Пробуждение трекинга: config_update wake_device + health + GPS (сработает, когда телефон снова опросит сервер).
func (dc *DeviceController) PostAdminWakeDevice(ctx *gin.Context) {
	currentUser, ok := getCurrentUserFromContext(ctx)
	if !ok {
		return
	}
	if !currentUser.IsAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Требуются права администратора"})
		return
	}

	userID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || userID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
		return
	}

	wake := true
	enableLoc := true
	apiBase := os.Getenv("BASE_URL")
	if apiBase == "" {
		apiBase = "http://87.232.65.52:8080"
	}
	payload, err := service.BuildConfigUpdatePayload(userID, service.DeviceConfigUpdateInput{
		WakeDevice:     &wake,
		EnableLocation: &enableLoc,
		APIBaseURL:     &apiBase,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось собрать команду"})
		return
	}

	configCmd, err := dc.CommandService.EnqueueCommand(userID, models.DeviceCommandTypeConfigUpdate, payload)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать config_update"})
		return
	}

	resp := gin.H{
		"user_id":           userID,
		"config_command_id": configCmd.ID,
		"note":              "Команда пробуждения в очереди. Сработает при следующем опросе телефона (~15 с).",
	}
	ctx.JSON(http.StatusAccepted, resp)
}

// PostAdminEnableLocation — POST /api/admin/users/:id/enable-location
// Включает разрешения и системную геолокацию на устройстве (Device Owner).
func (dc *DeviceController) PostAdminEnableLocation(ctx *gin.Context) {
	currentUser, ok := getCurrentUserFromContext(ctx)
	if !ok {
		return
	}
	if !currentUser.IsAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Требуются права администратора"})
		return
	}

	userID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || userID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
		return
	}

	enableLoc := true
	wake := true
	apiBase := os.Getenv("BASE_URL")
	if apiBase == "" {
		apiBase = "http://87.232.65.52:8080"
	}
	payload, err := service.BuildConfigUpdatePayload(userID, service.DeviceConfigUpdateInput{
		EnableLocation: &enableLoc,
		WakeDevice:     &wake,
		APIBaseURL:     &apiBase,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось собрать команду"})
		return
	}

	configCmd, err := dc.CommandService.EnqueueCommand(userID, models.DeviceCommandTypeConfigUpdate, payload)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать config_update"})
		return
	}

	resp := gin.H{
		"user_id":           userID,
		"config_command_id": configCmd.ID,
		"note":              "Команда включения GPS в очереди. Телефон получит её при следующем опросе (~15 с).",
	}
	ctx.JSON(http.StatusAccepted, resp)
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

// PostAdminUserDeviceConfig — POST /api/admin/users/:id/device/config
// Валидированный config_update (ключ, интервалы, PIN, пауза трекинга, скрытие из лаунчера).
func (dc *DeviceController) PostAdminUserDeviceConfig(ctx *gin.Context) {
	currentUser, ok := getCurrentUserFromContext(ctx)
	if !ok {
		return
	}
	if !currentUser.IsAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Требуются права администратора"})
		return
	}

	userID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || userID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
		return
	}

	var body service.DeviceConfigUpdateInput
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Некорректное тело запроса"})
		return
	}

	payload, err := service.BuildConfigUpdatePayload(userID, body)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDeviceConfigUpdateEmpty):
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrDeviceConfigUpdateInvalid):
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные параметры (интервалы, PIN или URL)"})
		default:
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	cmd, err := dc.CommandService.EnqueueCommand(userID, models.DeviceCommandTypeConfigUpdate, payload)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать команду"})
		return
	}

	cmdPayload, _ := service.CommandPayloadMap(cmd)
	ctx.JSON(http.StatusAccepted, gin.H{
		"command_id": cmd.ID,
		"type":       cmd.Type,
		"status":     cmd.Status,
		"user_id":    cmd.UserID,
		"payload":    cmdPayload,
	})
}

// PostPublishAppUpdate — POST /api/admin/releases/publish-update/:user_id
// Команда app_update из manifest.json на устройство.
func (dc *DeviceController) PostPublishAppUpdate(ctx *gin.Context) {
	currentUser, ok := getCurrentUserFromContext(ctx)
	if !ok {
		return
	}
	if !currentUser.IsAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Требуются права администратора"})
		return
	}
	if dc.ReleaseController == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "Релизы не настроены"})
		return
	}

	userID, err := strconv.Atoi(ctx.Param("user_id"))
	if err != nil || userID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Неверный user_id"})
		return
	}

	payload, err := dc.ReleaseController.ManifestForAppUpdate()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Манифест релиза недоступен"})
		return
	}

	cmd, err := dc.CommandService.EnqueueCommand(userID, models.DeviceCommandTypeAppUpdate, payload)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать команду"})
		return
	}
	cmdPayload, _ := service.CommandPayloadMap(cmd)
	ctx.JSON(http.StatusAccepted, gin.H{
		"command_id": cmd.ID,
		"type":       cmd.Type,
		"user_id":    userID,
		"payload":    cmdPayload,
	})
}
