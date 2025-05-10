package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"locator/controllers"
	"locator/dao"
	"locator/location"
)

func main() {
	// Инициализируем подключение к базе данных.
	dbConn := dao.InitDB()

	// Выполняем автоматическую миграцию моделей Location и Checkpoint.
	if err := dbConn.AutoMigrate(&location.Location{}, &location.Checkpoint{}); err != nil {
		log.Fatalf("Ошибка миграции БД: %v", err)
	}

	// Инициализируем слои DAO и сервисов для Location.
	locationDAO := dao.NewLocationDAO(dbConn)
	locationService := dao.NewLocationService(locationDAO)
	locationController := controllers.NewLocationController(locationService)

	// Инициализируем слои DAO и сервисов для Checkpoint.
	// Предполагается, что функции NewCheckpointDAO и NewCheckpointService реализованы в пакете dao.
	checkpointDAO := dao.NewCheckpointDAO(dbConn)
	checkpointService := dao.NewCheckpointService(checkpointDAO)
	checkpointController := controllers.NewCheckpointController(checkpointService)

	// Инициализируем Gin.
	router := gin.Default()

	// Регистрируем маршруты для работы с локациями.
	locationGroup := router.Group("/location")
	{
		locationGroup.GET("/", locationController.GetLocations)
		locationGroup.GET("/single", locationController.GetLocation) // Например, для запроса по user_id
		locationGroup.POST("/", locationController.PostLocation)
	}

	// Регистрируем маршруты для работы с чекпоинтами.
	checkpointGroup := router.Group("/checkpoint")
	{
		checkpointGroup.GET("/", checkpointController.GetCheckpoints)
		checkpointGroup.POST("/", checkpointController.PostCheckpoint)
	}

	log.Println("Сервер запущен на порту 8080")
	router.Run(":8080")
}
