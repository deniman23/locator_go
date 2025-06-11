package router

import (
	"locator/controllers"

	"github.com/gin-gonic/gin"
)

// InitRoutes настраивает маршруты для работы с локациями, чекпоинтами и визитами, и возвращает *gin.Engine.
func InitRoutes(
	locationController *controllers.LocationController,
	checkpointController *controllers.CheckpointController,
	visitController *controllers.VisitController,
) *gin.Engine {
	router := gin.Default()

	// Группа маршрутов для работы с локациями.
	locationGroup := router.Group("/location")
	{
		locationGroup.GET("/", locationController.GetLocations)
		locationGroup.GET("/single", locationController.GetLocation) // например, получение по user_id
		locationGroup.POST("/", locationController.PostLocation)
	}

	// Группа маршрутов для работы с чекпоинтами.
	checkpointGroup := router.Group("/checkpoint")
	{
		checkpointGroup.GET("/", checkpointController.GetCheckpoints)
		checkpointGroup.POST("/", checkpointController.PostCheckpoint)
		// Новый маршрут для проверки, находится ли локация пользователя в чекпоинте.
		checkpointGroup.GET("/check", checkpointController.CheckUserInCheckpoint)
	}

	// Группа маршрутов для работы с визитами.
	visitGroup := router.Group("/visits")
	{
		// Эндпоинт для получения истории визитов пользователя.
		visitGroup.GET("/", visitController.GetVisitsByUser)
		// Здесь можно добавить и другие эндпоинты, связанные с визитами.
	}

	return router
}
