package router

import (
	"locator/controllers"
	"locator/middleware"
	"locator/service"

	"github.com/gin-gonic/gin"
)

func InitRoutes(
	locationController *controllers.LocationController,
	checkpointController *controllers.CheckpointController,
	visitController *controllers.VisitController,
	eventController *controllers.EventController,
	userController *controllers.UserController,
	userService *service.UserService,
) *gin.Engine {
	router := gin.Default()

	apiGroup := router.Group("/api")
	apiGroup.Use(middleware.APIKeyAuthMiddleware(userService))
	{
		locationGroup := apiGroup.Group("/location")
		{
			locationGroup.GET("/", locationController.GetLocations)
			locationGroup.GET("/single", locationController.GetLocation)
			locationGroup.POST("/", locationController.PostLocation)
		}

		// Группа маршрутов для работы с чекпоинтами.
		checkpointGroup := apiGroup.Group("/checkpoint")
		{
			checkpointGroup.GET("/", checkpointController.GetCheckpoints)
			checkpointGroup.POST("/", checkpointController.PostCheckpoint)
			// Новый маршрут для проверки, находится ли локация пользователя в чекпоинте.
			checkpointGroup.GET("/check", checkpointController.CheckUserInCheckpoint)
		}

		// Группа маршрутов для работы с визитами.
		visitGroup := apiGroup.Group("/visits")
		{
			// Эндпоинт для получения истории визитов пользователя.
			visitGroup.GET("/", visitController.GetVisitsByUser)
		}

		// Группа маршрутов для публикации событий (например, в RabbitMQ).
		eventGroup := apiGroup.Group("/event")
		{
			eventGroup.POST("/publish", eventController.PublishEvent)
		}

		// Группа маршрутов для работы с пользователями.
		userGroup := apiGroup.Group("/users")
		{
			userGroup.POST("/", userController.CreateUser)
			userGroup.GET("/:id", userController.GetUser)
			userGroup.GET("/", userController.GetAllUsers)
		}
	}

	return router
}
