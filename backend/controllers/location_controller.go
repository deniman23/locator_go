package controllers

import (
	"fmt"
	"locator/config/messaging"
	"locator/models"
	"locator/service"
	"net/http"
	"strconv"
	"strings"
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
	Service        *service.LocationService
	Publisher      *messaging.Publisher
	RoutingBaseURL string // OSRM/совместимый инстанс, без завершающего /; пусто — эндпоинт match недоступен
	HTTPRouting    *http.Client
}

// NewLocationController создаёт новый экземпляр контроллера для работы с локациями.
// routingBaseURL — из переменной окружения ROUTING_BASE_URL (опционально).
func NewLocationController(service *service.LocationService, publisher *messaging.Publisher, routingBaseURL string) *LocationController {
	return &LocationController{
		Service:        service,
		Publisher:      publisher,
		RoutingBaseURL: strings.TrimSpace(routingBaseURL),
		HTTPRouting:    &http.Client{Timeout: 60 * time.Second},
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

// GetLocations обрабатывает GET-запрос для получения локаций.
// Query raw=true|1 — все точки из БД без фильтра «значимых» (по умолчанию — значимые, как раньше).
// Параметры from и to — интервал времени (RFC3339 или YYYY-MM-DDTHH:mm в Europe/Minsk).
func (lc *LocationController) GetLocations(ctx *gin.Context) {
	from := ctx.Query("from")
	to := ctx.Query("to")
	raw := ctx.Query("raw")
	useRaw := raw == "true" || raw == "1"

	var locations []models.Location
	var err error

	if from != "" && to != "" {
		if useRaw {
			locations, err = lc.Service.GetLocationsBetweenRaw(from, to)
		} else {
			locations, err = lc.Service.GetLocationsBetween(from, to)
		}
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Ошибка фильтрации по интервалу: %v", err)})
			return
		}
	} else {
		if useRaw {
			locations, err = lc.Service.GetLocationsRaw()
		} else {
			locations, err = lc.Service.GetLocations()
		}
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения записей"})
			return
		}
	}
	ctx.JSON(http.StatusOK, locations)
}

// GetMatchedRoute возвращает координаты линии, привязанной к дорогам (OSRM match).
// Требуются query: user_id, from, to. Нужна переменная окружения ROUTING_BASE_URL.
func (lc *LocationController) GetMatchedRoute(ctx *gin.Context) {
	currentUser, ok := getCurrentUserFromContext(ctx)
	if !ok {
		return
	}
	if lc.RoutingBaseURL == "" {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "Маршрутизация не настроена (ROUTING_BASE_URL)"})
		return
	}

	userIDStr := ctx.Query("user_id")
	from := ctx.Query("from")
	to := ctx.Query("to")
	if userIDStr == "" || from == "" || to == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Укажите user_id, from и to"})
		return
	}
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id должен быть числом"})
		return
	}
	if !currentUser.IsAdmin && userID != currentUser.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав"})
		return
	}

	all, err := lc.Service.GetLocationsBetweenRaw(from, to)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Интервал: %v", err)})
		return
	}
	var forUser []models.Location
	for _, l := range all {
		if l.UserID == userID {
			forUser = append(forUser, l)
		}
	}
	if len(forUser) < 2 {
		ctx.JSON(http.StatusOK, gin.H{"coordinates": [][]float64{}})
		return
	}

	coords, err := service.MatchLocationsToRoads(lc.HTTPRouting, lc.RoutingBaseURL, forUser)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"coordinates": coords})
}
