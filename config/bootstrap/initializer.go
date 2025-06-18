package bootstrap

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/logger"
	"locator/config"
	"locator/config/messaging"
	"locator/controllers"
	"locator/dao"
	"locator/router"
	"locator/service"
)

// App содержит зависимости приложения.
type App struct {
	Router    *gin.Engine // основная маршрутизация приложения
	DB        interface{} // подключение к БД (например, *gorm.DB)
	RMQClient *messaging.RabbitMQClient
}

// InitializeApp собирает все зависимости приложения и возвращает готовый инстанс App.
// Теперь функция принимает параметр dbLogger, который используется для инициализации БД с нужным уровнем логирования SQL-запросов.
func InitializeApp(dbLogger logger.Interface) (*App, error) {
	// 1. Инициализация подключения к БД с использованием переданного логгера для SQL-запросов.
	// Для этого предполагается, что функция config.InitDB обновлена и принимает параметр logger.
	dbConn := config.InitDB(dbLogger)

	// Здесь можно проводить миграции, либо оставить их в main или в отдельном скрипте.
	// Пример: dbConn.AutoMigrate(&models.Location{}, &models.Checkpoint{}, &models.Visit{})

	// 2. Инициализация подключения к RabbitMQ.
	rmqURL := "amqp://guest:guest@localhost:5672/"
	rmqClient, err := messaging.NewRabbitMQClient(rmqURL)
	if err != nil {
		return nil, err
	}

	// Создаем Publisher с нужными параметрами: пустая строка для обмена и "location_events" в качестве ключа маршрутизации.
	publisher := messaging.NewPublisher(rmqClient, "", "location_events")

	// Объявляем очередь для обработчика событий локаций.
	queue, err := rmqClient.Channel.QueueDeclare(
		"location_events", // имя очереди
		true,              // долговечная очередь
		false,             // не удалять, если не используется
		false,             // не эксклюзивная
		false,             // без ожидания
		nil,               // аргументы
	)
	if err != nil {
		log.Printf("Ошибка объявления очереди: %v", err)
	} else {
		log.Printf("Очередь '%s' объявлена успешно", queue.Name)
	}

	// 3. Инициализация слоев DAO, сервисов и контроллеров.

	// Для Location:
	locationDAO := dao.NewLocationDAO(dbConn)
	locationService := service.NewLocationService(locationDAO)
	locationController := controllers.NewLocationController(locationService, publisher)

	// Для Checkpoint и Visit:
	checkpointDAO := dao.NewCheckpointDAO(dbConn)
	checkpointService := service.NewCheckpointService(checkpointDAO)
	visitDAO := dao.NewVisitDAO(dbConn)
	visitService := service.NewVisitService(visitDAO)
	checkpointController := controllers.NewCheckpointController(checkpointService, locationService, visitService, publisher)
	visitController := controllers.NewVisitController(visitService)

	// Инициализация EventController с Publisher.
	eventController := controllers.NewEventController(publisher)

	// 4. Инициализация роутера с добавлением всех контроллеров.
	routerEngine := router.InitRoutes(locationController, checkpointController, visitController, eventController)

	// Запуск потребителя сообщений для обработки событий из очереди "location_events".
	visitEventProcessor := service.NewVisitEventProcessor(checkpointService, visitService)
	consumer := messaging.NewConsumer(rmqClient, queue.Name)
	go func() {
		if err := consumer.Consume(func(message []byte) error {
			// Обработка сообщения (например, создание или завершение визита)
			return visitEventProcessor.ProcessEvent(message)
		}); err != nil {
			log.Printf("Ошибка при потреблении сообщений: %v", err)
		}
	}()

	// Пример публикации демо-события (опционально).
	go func() {
		demoEvent := map[string]interface{}{
			"user_id":       1,
			"checkpoint_id": 1,
			"latitude":      55.7522,
			"longitude":     37.6156,
			"occurred_at":   time.Now(),
		}

		if err := publisher.PublishJSON(demoEvent); err != nil {
			log.Printf("Ошибка публикации демо-события в RabbitMQ: %v", err)
		} else {
			log.Println("Демо-событие опубликовано в RabbitMQ")
		}
	}()

	app := &App{
		Router:    routerEngine,
		DB:        dbConn,
		RMQClient: rmqClient,
	}

	return app, nil
}
