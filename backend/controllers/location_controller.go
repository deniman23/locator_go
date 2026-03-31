package controllers

import (
	"fmt"
	"locator/config/messaging"
	"locator/models"
	"locator/service"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func getCurrentUserFromContext(ctx *gin.Context) (*models.User, bool) {
	userInterface, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Необходима авторизация"})
		return nil, false
	}

	currentUser, ok := userInterface.(*models.User)
	if !ok || currentUser == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка определения текущего пользователя"})
		return nil, false
	}

	return currentUser, true
}

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
	currentUser, ok := getCurrentUserFromContext(ctx)
	if !ok {
		return
	}

	requestedUserID := currentUser.ID
	userIDStr := ctx.Query("user_id")
	if userIDStr != "" {
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id должен быть числом"})
			return
		}
		requestedUserID = userID
	}

	if !currentUser.IsAdmin && requestedUserID != currentUser.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав для просмотра чужой локации"})
		return
	}

	location, err := lc.Service.GetLocation(requestedUserID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Данные о местоположении не найдены"})
		return
	}
	ctx.JSON(http.StatusOK, location)
}

// PostLocation обрабатывает POST-запрос для создания или обновления локации.
func (lc *LocationController) PostLocation(ctx *gin.Context) {
	currentUser, ok := getCurrentUserFromContext(ctx)
	if !ok {
		return
	}

	var req struct {
		UserID    int     `json:"user_id"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Некорректное тело запроса"})
		return
	}

	targetUserID := req.UserID
	if targetUserID == 0 {
		targetUserID = currentUser.ID
	}

	if !currentUser.IsAdmin && targetUserID != currentUser.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав для обновления чужой локации"})
		return
	}

	// Создаём новую запись о локации вместо обновления существующей.
	location, err := lc.Service.CreateLocation(targetUserID, req.Latitude, req.Longitude)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания записи"})
		return
	}

	// Формируем событие для RabbitMQ на основе новой локации.
	event := models.LocationEvent{
		UserID:     targetUserID,
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
	from := ctx.Query("from")
	to := ctx.Query("to")
	var locations []models.Location
	var err error

	if from != "" && to != "" {
		locations, err = lc.Service.GetLocationsBetween(from, to)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Ошибка фильтрации по интервалу: %v", err)})
			return
		}
	} else {
		// Если не переданы параметры интервала, возвращаем все записи или можно оставить другую логику фильтрации.
		locations, err = lc.Service.GetLocations()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения записей"})
			return
		}
	}
	ctx.JSON(http.StatusOK, locations)
}
