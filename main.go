package main

import (
	"log"

	"locator/config/bootstrap"
)

func main() {
	// Инициализируем приложение: сбор всех зависимостей и настройка сервера.
	app, err := bootstrap.InitializeApp()
	if err != nil {
		log.Fatalf("Ошибка инициализации приложения: %v", err)
	}

	// Отложенное закрытие соединения с RabbitMQ при завершении работы приложения.
	defer app.RMQClient.Close()

	log.Println("Сервер запущен на порту 8080")

	// Запускаем HTTP сервер.
	if err := app.Router.Run(":8080"); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
