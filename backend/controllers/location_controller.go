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

// LocationController отвечает за обработку запросов, связанных с локациями.
type LocationController struct {
	Service   *service.LocationService
	Publisher *messaging.Publisher // Предположим, что Publisher интегрирован в сервисы и доступен из контроллера
}

// NewLocationController создаёт новый экземпляр контроллера для работы с локациями.
func NewLocationController(service *service.LocationService, publisher *messaging.Publisher) *LocationController {
	return &LocationController{
		Service:   service,
		Publisher: publisher,
	}
}

// GetLocation обрабатывает GET-запрос для получения данных о местоположении по user_id.
func (lc *LocationController) GetLocation(ctx *gin.Context) {
	userIDStr := ctx.Query("user_id")
	if userIDStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Параметр user_id не указан"})
		return
	}
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id должен быть числом"})
		return
	}
	location, err := lc.Service.GetLocation(userID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Данные о местоположении не найдены"})
		return
	}
	ctx.JSON(http.StatusOK, location)
}

// PostLocation обрабатывает POST-запрос для создания или обновления локации.
// PostLocation обрабатывает POST-запрос для создания новой локации.
func (lc *LocationController) PostLocation(ctx *gin.Context) {
	var req struct {
		UserID    int     `json:"user_id"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Некорректное тело запроса"})
		return
	}

	// Создаём новую запись о локации вместо обновления существующей.
	location, err := lc.Service.CreateLocation(req.UserID, req.Latitude, req.Longitude)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания записи"})
		return
	}

	// Формируем событие для RabbitMQ на основе новой локации.
	event := models.LocationEvent{
		UserID:     req.UserID,
		Latitude:   req.Latitude,
		Longitude:  req.Longitude,
		OccurredAt: time.Now(),
	}
	// Публикуем событие в RabbitMQ.
	if err := lc.Publisher.PublishJSON(event); err != nil {
		// Ошибку публикации можно залогировать, но не блокировать ответ клиенту.
		// log.Printf("Ошибка публикации события: %v", err)
	}

	ctx.JSON(http.StatusOK, location)
}

// GetLocations обрабатывает GET-запрос для получения всех локаций.
func (lc *LocationController) GetLocations(ctx *gin.Context) {
	locations, err := lc.Service.GetLocations()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения записей"})
		return
	}
	ctx.JSON(http.StatusOK, locations)
}
