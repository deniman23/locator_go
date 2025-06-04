package controllers

import (
	"locator/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CheckpointController отвечает за обработку запросов, связанных с чекпоинтами.
type CheckpointController struct {
	Service         *service.CheckpointService
	LocationService *service.LocationService
}

// NewCheckpointController создаёт новый экземпляр контроллера для работы с чекпоинтами.
// Он принимает и CheckpointService, и LocationService для проверки попадания локации в чекпоинт.
func NewCheckpointController(checkpointService *service.CheckpointService, locationService *service.LocationService) *CheckpointController {
	return &CheckpointController{
		Service:         checkpointService,
		LocationService: locationService,
	}
}

// PostCheckpoint обрабатывает POST-запрос для создания нового чекпоинта.
func (cc *CheckpointController) PostCheckpoint(ctx *gin.Context) {
	var req struct {
		Name      string  `json:"name"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Radius    float64 `json:"radius"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Некорректное тело запроса"})
		return
	}
	cp, err := cc.Service.CreateCheckpoint(req.Name, req.Latitude, req.Longitude, req.Radius)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания чекпоинта"})
		return
	}
	ctx.JSON(http.StatusOK, cp)
}

// GetCheckpoints обрабатывает GET-запрос для получения всех чекпоинтов.
func (cc *CheckpointController) GetCheckpoints(ctx *gin.Context) {
	checkpoints, err := cc.Service.GetCheckpoints()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения чекпоинтов"})
		return
	}
	ctx.JSON(http.StatusOK, checkpoints)
}

// CheckUserInCheckpoint обрабатывает GET-запрос для проверки, находится ли локация пользователя в указанном чекпоинте.
// Ожидает в параметрах запроса: user_id и checkpoint_id.
func (cc *CheckpointController) CheckUserInCheckpoint(ctx *gin.Context) {
	userIDStr := ctx.Query("user_id")
	checkpointID := ctx.Query("checkpoint_id")
	if userIDStr == "" || checkpointID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Не переданы необходимые параметры: user_id и checkpoint_id"})
		return
	}
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id должен быть числом"})
		return
	}

	// Получаем локацию пользователя через LocationService.
	loc, err := cc.LocationService.GetLocation(userID)
	if err != nil || loc == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Локация пользователя не найдена"})
		return
	}

	// Получаем чекпоинт по его ID.
	cp, err := cc.Service.GetCheckpointByID(checkpointID)
	if err != nil || cp == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Чекпоинт не найден"})
		return
	}

	// Проверяем, находится ли локация пользователя в зоне чекпоинта.
	inCheckpoint := cc.Service.IsLocationInCheckpoint(loc, cp)
	ctx.JSON(http.StatusOK, gin.H{"in_checkpoint": inCheckpoint})
}
