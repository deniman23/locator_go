package router

import (
	"net/http"

	"locator/controllers"
	"locator/middleware"
	"locator/service"

	"github.com/gin-gonic/gin"
)

func InitRoutes(
	locationController *controllers.LocationController,
	locationRequestController *controllers.LocationRequestController,
	deviceController *controllers.DeviceController,
	appReleaseController *controllers.AppReleaseController,
	checkpointController *controllers.CheckpointController,
	visitController *controllers.VisitController,
	eventController *controllers.EventController,
	userController *controllers.UserController,
	userService *service.UserService,
) *gin.Engine {
	router := gin.Default()

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Block direct access to QR PNG files — they embed plaintext API keys.
	// Authenticated callers use /api/users/:id/qr-code-file instead.
	router.GET("/static/qrcode/*filepath", func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "direct QR access forbidden"})
	})
	router.HEAD("/static/qrcode/*filepath", func(c *gin.Context) {
		c.AbortWithStatus(http.StatusForbidden)
	})

	// Serve only non-QR static assets publicly.
	router.Static("/static/releases", "./static/releases")

	// Базовый маршрут для API, без middleware
	apiGroup := router.Group("/api")
	apiGroup.GET("/app/release/latest", appReleaseController.GetLatestRelease)

	// Маршруты, доступные всем авторизованным пользователям
	basicAuthGroup := apiGroup.Group("")
	basicAuthGroup.Use(middleware.BasicAuthMiddleware(userService))
	{
		// Информация о текущем пользователе
		basicAuthGroup.GET("/users/me", userController.GetCurrentUser)

		// Разрешаем всем пользователям создавать и получать локации
		basicAuthGroup.POST("/location", locationController.PostLocation)
		basicAuthGroup.GET("/location/single", locationController.GetLocation)
		basicAuthGroup.GET("/location/current", locationController.GetLocation)

		// Мобильный коннектор и legacy poll location request
		basicAuthGroup.GET("/device/poll", deviceController.PollDevice)
		basicAuthGroup.POST("/device/report", deviceController.PostDeviceReport)
		basicAuthGroup.POST("/device/command/ack", deviceController.PostCommandAck)
		basicAuthGroup.GET("/location/request", locationRequestController.PollLocationRequest)
	}

	// Остальные маршруты API требуют полной аутентификации с проверкой на админа
	protectedApiGroup := apiGroup.Group("")
	protectedApiGroup.Use(middleware.APIKeyAuthMiddleware(userService))
	{
		locationGroup := protectedApiGroup.Group("/location")
		{
			locationGroup.GET("/match-route", locationController.GetMatchedRoute)
			locationGroup.GET("/", locationController.GetLocations)
			locationGroup.POST("/request", locationRequestController.PostLocationRequest)
			locationGroup.GET("/request/:request_id", locationRequestController.GetLocationRequestStatus)
		}

		adminGroup := protectedApiGroup.Group("/admin")
		{
			adminGroup.GET("/devices/status", deviceController.GetAdminDevicesStatus)
			adminGroup.POST("/users/:id/wake", deviceController.PostAdminWakeDevice)
			adminGroup.POST("/users/:id/enable-location", deviceController.PostAdminEnableLocation)
			adminGroup.POST("/users/:id/commands", deviceController.PostAdminUserCommand)
			adminGroup.POST("/users/:id/device/config", deviceController.PostAdminUserDeviceConfig)
			adminGroup.POST("/users/:id/regenerate-qr", userController.PostRegenerateUserQR)
			adminGroup.POST("/releases/publish-update/:user_id", deviceController.PostPublishAppUpdate)
			adminGroup.POST("/releases/sync-manifest", appReleaseController.PostSyncReleaseManifest)
			adminGroup.POST("/locations/backfill-captured-at", locationController.PostBackfillCapturedAt)
		}

		// Группа маршрутов для работы с чекпоинтами.
		checkpointGroup := protectedApiGroup.Group("/checkpoint")
		{
			checkpointGroup.GET("/", checkpointController.GetCheckpoints)
			checkpointGroup.POST("/", checkpointController.PostCheckpoint)
			checkpointGroup.PUT("/:id", checkpointController.UpdateCheckpoint)
			checkpointGroup.POST("/:id/archive", checkpointController.ArchiveCheckpoint)
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
			userGroup.PUT("/:id", userController.UpdateUser)
			userGroup.POST("/:id/disable", userController.PostDisableUser)
			userGroup.POST("/:id/enable", userController.PostEnableUser)
			userGroup.GET("/:id", userController.GetUser)
			userGroup.GET("/", userController.GetAllUsers)
			userGroup.GET("/qr-code", userController.GetQRCode)
			userGroup.GET("/qr-code-file", userController.GetQRCodeFile)
			userGroup.GET("/:id/qr-code", userController.GetUserQRCode)
			userGroup.GET("/:id/qr-code-file", userController.GetUserQRCodeFile)
			userGroup.GET("/:id/health", deviceController.GetUserHealth)
		}
	}

	return router
}
