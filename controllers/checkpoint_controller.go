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
	VisitService    *service.VisitService // добавляем сервис для работы с визитами
}

// NewCheckpointController создаёт новый экземпляр контроллера для работы с чекпоинтами.
// Теперь он принимает также VisitService.
func NewCheckpointController(
	checkpointService *service.CheckpointService,
	locationService *service.LocationService,
	visitService *service.VisitService,
) *CheckpointController {
	return &CheckpointController{
		Service:         checkpointService,
		LocationService: locationService,
		VisitService:    visitService,
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
// Если пользователь входит в зону, создаётся новый визит (при отсутствии активного);
// если выходит — завершается активный визит, записывая время окончания и длительность визита.
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

	// Проверяем, находится ли локация пользователя в зоне чекпоинта.
	inCheckpoint := cc.Service.IsLocationInCheckpoint(loc, cp)

	// Логика визитов:
	// Если пользователь находится в зоне и активного визита ещё нет – стартуем новый визит.
	// Если же пользователь покинул зону и активный визит существует – завершаем визит.
	activeVisit, _ := cc.VisitService.GetActiveVisit(userID, checkpointID)
	if inCheckpoint {
		if activeVisit == nil {
			_, err := cc.VisitService.StartVisit(userID, checkpointID)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка начала визита"})
				return
			}
		}
	} else {
		if activeVisit != nil {
			err := cc.VisitService.EndVisit(activeVisit)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка завершения визита"})
				return
			}
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"in_checkpoint": inCheckpoint})
}
