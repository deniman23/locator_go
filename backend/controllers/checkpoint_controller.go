package controllers

import (
	"locator/config/messaging"
	"locator/models"
	"locator/service"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// CheckpointController отвечает за обработку запросов, связанных с чекпоинтами.
type CheckpointController struct {
	Service         *service.CheckpointService
	LocationService *service.LocationService
	VisitService    *service.VisitService // для работы с визитами
	Publisher       *messaging.Publisher  // интеграция RabbitMQ
}

// NewCheckpointController создаёт новый экземпляр контроллера для работы с чекпоинтами.
// Обратите внимание, что теперь передаётся Publisher для отправки событий в RabbitMQ.
func NewCheckpointController(
	checkpointService *service.CheckpointService,
	locationService *service.LocationService,
	visitService *service.VisitService,
	publisher *messaging.Publisher,
) *CheckpointController {
	return &CheckpointController{
		Service:         checkpointService,
		LocationService: locationService,
		VisitService:    visitService,
		Publisher:       publisher,
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

// UpdateCheckpoint обрабатывает PUT-запрос для обновления существующего чекпоинта.
func (cc *CheckpointController) UpdateCheckpoint(ctx *gin.Context) {
	// Получаем ID чекпоинта из параметров пути
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID чекпоинта"})
		return
	}

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

	cp, err := cc.Service.UpdateCheckpoint(id, req.Name, req.Latitude, req.Longitude, req.Radius)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления чекпоинта"})
		return
	}
	ctx.JSON(http.StatusOK, cp)
}

// CheckUserInCheckpoint обрабатывает GET-запрос для проверки, находится ли локация пользователя в указанном чекпоинте.
// Вместо синхронной обработки, формируется событие и отправляется в очередь RabbitMQ для асинхронной обработки.
// После публикации события клиенту возвращается сообщение о том, что событие отправлено на обработку.
func (cc *CheckpointController) CheckUserInCheckpoint(ctx *gin.Context) {
	userIDStr := ctx.Query("user_id")
	checkpointIDStr := ctx.Query("checkpoint_id")
	if userIDStr == "" || checkpointIDStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Не переданы необходимые параметры: user_id и checkpoint_id"})
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id должен быть числом"})
		return
	}

	checkpointID, err := strconv.Atoi(checkpointIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "checkpoint_id должен быть числом"})
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

	// Формируем событие на основе полученных данных.
	// Модель LocationEvent должна содержать поля: UserID, CheckpointID, Latitude, Longitude, OccurredAt.
	event := models.LocationEvent{
		UserID:       userID,
		CheckpointID: checkpointID,
		Latitude:     loc.Latitude,
		Longitude:    loc.Longitude,
		OccurredAt:   time.Now(),
	}

	// Публикуем событие в RabbitMQ для дальнейшей асинхронной обработки (например, создания или завершения визита).
	if err := cc.Publisher.PublishJSON(event); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка публикации события"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Событие отправлено на обработку"})
}
