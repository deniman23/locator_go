package router

import (
	"locator/controllers"

	"github.com/gin-gonic/gin"
)

// InitRoutes настраивает маршруты для работы с локациями, чекпоинтами, визитами и событиями, и возвращает *gin.Engine.
func InitRoutes(
	locationController *controllers.LocationController,
	checkpointController *controllers.CheckpointController,
	visitController *controllers.VisitController,
	eventController *controllers.EventController, // новый контроллер для работы с событиями
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

	// Новая группа маршрутов для работы с событиями (например, публикация сообщений в RabbitMQ).
	eventGroup := router.Group("/event")
	{
		// Например, POST-запрос для публикации события.
		eventGroup.POST("/publish", eventController.PublishEvent)
	}

	return router
}
