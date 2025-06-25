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

	// Доступные без авторизации статические файлы (если они нужны)
	router.Static("/static", "./static")

	// Базовый маршрут для API, без middleware
	apiGroup := router.Group("/api")

	// Маршруты, доступные всем авторизованным пользователям
	basicAuthGroup := apiGroup.Group("")
	basicAuthGroup.Use(middleware.BasicAuthMiddleware(userService))
	{
		// Информация о текущем пользователе
		basicAuthGroup.GET("/users/me", userController.GetCurrentUser)

		// Разрешаем всем пользователям создавать и получать локации
		basicAuthGroup.POST("/location", locationController.PostLocation)
		basicAuthGroup.GET("/location/single", locationController.GetLocation)
	}

	// Остальные маршруты API требуют полной аутентификации с проверкой на админа
	protectedApiGroup := apiGroup.Group("")
	protectedApiGroup.Use(middleware.APIKeyAuthMiddleware(userService))
	{
		locationGroup := protectedApiGroup.Group("/location")
		{
			locationGroup.GET("/", locationController.GetLocations)
			// Убираем дублирующийся маршрут, так как он уже доступен через basicAuthGroup
			// locationGroup.GET("/single", locationController.GetLocation)
			// locationGroup.POST("/", locationController.PostLocation)
		}

		// Группа маршрутов для работы с чекпоинтами.
		checkpointGroup := protectedApiGroup.Group("/checkpoint")
		{
			checkpointGroup.GET("/", checkpointController.GetCheckpoints)
			checkpointGroup.POST("/", checkpointController.PostCheckpoint)
			checkpointGroup.PUT("/:id", checkpointController.UpdateCheckpoint)
			checkpointGroup.GET("/check", checkpointController.CheckUserInCheckpoint)
		}

		// Группа маршрутов для работы с визитами.
		visitGroup := protectedApiGroup.Group("/visits")
		{
			// Эндпоинт для получения визитов с фильтром.
			visitGroup.GET("/", visitController.GetVisitsByFilters)
		}

		// Группа маршрутов для публикации событий (например, в RabbitMQ).
		eventGroup := protectedApiGroup.Group("/event")
		{
			eventGroup.POST("/publish", eventController.PublishEvent)
		}

		// Группа маршрутов для работы с пользователями.
		userGroup := protectedApiGroup.Group("/users")
		{
			userGroup.POST("/", userController.CreateUser)
			userGroup.GET("/:id", userController.GetUser)
			userGroup.GET("/", userController.GetAllUsers)
			userGroup.GET("/qr-code", userController.GetQRCode)
			userGroup.GET("/qr-code-file", userController.GetQRCodeFile)
			userGroup.GET("/:id/qr-code", userController.GetUserQRCode)
			userGroup.GET("/:id/qr-code-file", userController.GetUserQRCodeFile)
		}
	}

	return router
}
