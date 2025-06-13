// Package controllers содержит контроллеры для работы с различными сущностями.
package controllers

import (
	"net/http"

	"locator/config/messaging"
	"locator/models"

	"github.com/gin-gonic/gin"
)

// EventController отвечает за обработку HTTP-запросов, связанных с событиями.
type EventController struct {
	Publisher *messaging.Publisher
}

// NewEventController создаёт новый EventController с переданным Publisher.
func NewEventController(publisher *messaging.Publisher) *EventController {
	return &EventController{
		Publisher: publisher,
	}
}

// PublishEvent обрабатывает POST-запрос для публикации события.
// Ожидается, что в теле запроса будет JSON, соответствующий модели models.LocationEvent.
func (ec *EventController) PublishEvent(c *gin.Context) {
	var event models.LocationEvent

	// Привязываем JSON из запроса к структуре event.
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные события"})
		return
	}

	// При необходимости можно обновить время события:
	// event.OccurredAt = time.Now()

	// Публикуем событие в RabbitMQ через Publisher.
	if err := ec.Publisher.PublishJSON(event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка публикации события"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Событие успешно опубликовано"})
}
