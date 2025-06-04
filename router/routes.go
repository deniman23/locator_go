package router

import (
	"locator/controllers"

	"github.com/gin-gonic/gin"
)

// InitRoutes настраивает маршруты для работы с локациями и чекпоинтами и возвращает *gin.Engine.
func InitRoutes(locationController *controllers.LocationController, checkpointController *controllers.CheckpointController) *gin.Engine {
	router := gin.Default()

	// Группа маршрутов для работы с локациями.
	locationGroup := router.Group("/models")
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

	return router
}
