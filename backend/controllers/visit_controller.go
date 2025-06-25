package controllers

import (
	"github.com/gin-gonic/gin"
	"locator/service"
	"net/http"
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

func (vc *VisitController) GetVisitsByFilters(ctx *gin.Context) {
	visits, err := vc.VisitService.GetVisitsByFilters(ctx.Request.URL.Query())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения визитов"})
		return
	}
	ctx.JSON(http.StatusOK, visits)
}
