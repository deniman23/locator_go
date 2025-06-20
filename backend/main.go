package main

import (
	"locator/config"
	"locator/config/bootstrap"
	"log"
)

func main() {
	// Инициализируем основной логгер приложения с ежедневной ротацией логов.
	// Логи будут писаться в "logs/app.log" с архивированием.
	config.InitLogger("logs/app.log")

	// Инициализируем логгер для логирования SQL-запросов к базе данных.
	// В продакшене рекомендуется использовать уровень logger.Warn,
	// чтобы логировались только предупреждения, ошибки и медленные запросы.
	dbLogger := config.InitDBQueryLogger("logs/db.log")

	// Инициализируем приложение и передаём логгер для работы с БД.
	// Функция bootstrap.InitializeApp должна быть обновлена для приема параметра dbLogger.
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
