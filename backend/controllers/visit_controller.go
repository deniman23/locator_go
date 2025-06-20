package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"locator/service"
)

// VisitController отвечает за обработку запросов, связанных с визитами (посещениями чекпоинтов).
type VisitController struct {
	VisitService *service.VisitService
}

// NewVisitController создаёт новый экземпляр VisitController.
func NewVisitController(visitService *service.VisitService) *VisitController {
	return &VisitController{
		VisitService: visitService,
	}
}

// GetVisitsByUser возвращает историю посещений для указанного пользователя.
func (vc *VisitController) GetVisitsByUser(ctx *gin.Context) {
	userIDStr := ctx.Query("user_id")
	if userIDStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Параметр user_id обязателен"})
		return
	}
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id должен быть числом"})
		return
	}

	// Здесь предполагается, что в сервисе или DAO реализован метод получения списка визитов.
	visits, err := vc.VisitService.GetVisitsByUser(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения визитов"})
		return
	}
	ctx.JSON(http.StatusOK, visits)
}
