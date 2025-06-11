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

	// Выполняем автоматическую миграцию моделей Location, Checkpoint и Visit.
	if err := dbConn.AutoMigrate(&models.Location{}, &models.Checkpoint{}, &models.Visit{}); err != nil {
		log.Fatalf("Ошибка миграции БД: %v", err)
	}

	// Инициализируем слои DAO и сервисов для Location.
	locationDAO := dao.NewLocationDAO(dbConn)
	locationService := service.NewLocationService(locationDAO)
	locationController := controllers.NewLocationController(locationService)

	// Инициализируем слои DAO и сервисов для Checkpoint.
	checkpointDAO := dao.NewCheckpointDAO(dbConn)
	checkpointService := service.NewCheckpointService(checkpointDAO)
	// Инициализируем DAO и сервис для работы с визитами.
	visitDAO := dao.NewVisitDAO(dbConn)
	visitService := service.NewVisitService(visitDAO)
	// Теперь конструктор CheckpointController принимает также VisitService.
	checkpointController := controllers.NewCheckpointController(checkpointService, locationService, visitService)

	// Инициализируем контроллер визитов.
	visitController := controllers.NewVisitController(visitService)

	// Инициализируем роутер через отдельный файл конфигурации маршрутов.
	routerEngine := router.InitRoutes(locationController, checkpointController, visitController)

	log.Println("Сервер запущен на порту 8080")
	routerEngine.Run(":8080")
}
