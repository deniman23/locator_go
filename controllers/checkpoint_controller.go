package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"locator/dao"
)

// CheckpointController отвечает за обработку запросов, связанных с чекпоинтами.
type CheckpointController struct {
	Service *dao.CheckpointService
}

// NewCheckpointController создаёт новый экземпляр контроллера для работы с чекпоинтами.
func NewCheckpointController(service *dao.CheckpointService) *CheckpointController {
	return &CheckpointController{Service: service}
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
