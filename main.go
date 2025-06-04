package main

import (
	"locator/config"
	"locator/controllers"
	"locator/dao"
	"locator/models"
	"locator/router"
	"locator/service"
	"log"
)

func main() {
	// Инициализируем подключение к базе данных.
	dbConn := config.InitDB()

	// Выполняем автоматическую миграцию моделей Location и Checkpoint.
	if err := dbConn.AutoMigrate(&models.Location{}, &models.Checkpoint{}); err != nil {
		log.Fatalf("Ошибка миграции БД: %v", err)
	}

	// Инициализируем слои DAO и сервисов для Location.
	locationDAO := dao.NewLocationDAO(dbConn)
	locationService := service.NewLocationService(locationDAO)
	locationController := controllers.NewLocationController(locationService)

	// Инициализируем слои DAO и сервисов для Checkpoint.
	// Предполагается, что функции NewCheckpointDAO и NewCheckpointService реализованы в пакете dao.
	checkpointDAO := dao.NewCheckpointDAO(dbConn)
	checkpointService := service.NewCheckpointService(checkpointDAO)
	// Передаём как CheckpointService, так и LocationService для проверки попадания локации в чекпоинт.
	checkpointController := controllers.NewCheckpointController(checkpointService, locationService)

	// Инициализируем роутер через отдельный файл конфигурации маршрутов.
	routerEngine := router.InitRoutes(locationController, checkpointController)

	log.Println("Сервер запущен на порту 8080")
	routerEngine.Run(":8080")
}
