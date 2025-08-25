package main

import (
	"locator/config"
	"locator/config/bootstrap"
	"log"
)

func main() {
	config.InitLogger("logs/app.log")

	dbLogger := config.InitDBQueryLogger("logs/db.log")

	// Инициализируем приложение и передаём логгер для работы с БД.
	app, err := bootstrap.InitializeApp(dbLogger)
	if err != nil {
		log.Fatalf("Ошибка инициализации приложения: %v", err)
	}

	defer app.RMQClient.Close()
	log.Println("Сервер запущен на порту 8080")

	if err := app.Router.Run(":8080"); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
