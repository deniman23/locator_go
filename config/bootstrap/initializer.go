package bootstrap

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"locator/config"
	"locator/config/messaging"
	"locator/controllers"
	"locator/dao"
	"locator/router"
	"locator/service"
)

// App содержит зависимости приложения.
// Можно добавить сюда любые необходимые элементы, например, пул соединений, конфигурацию и т.д.
type App struct {
	Router    *gin.Engine // предположим, что в вашем роутере у вас тип RouterEngine
	DB        interface{} // оставляем в виде interface{} или конкретный *gorm.DB
	RMQClient *messaging.RabbitMQClient
}

// InitializeApp собирает все зависимости приложения и возвращает готовый инстанс App.
func InitializeApp() (*App, error) {
	// 1. Инициализация подключения к БД.
	dbConn := config.InitDB()

	// Здесь можно проводить миграции, либо оставить их в main или в отдельном скрипте.
	// dbConn.AutoMigrate(&models.Location{}, &models.Checkpoint{}, &models.Visit{})

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

	// Инициализируем EventController, передавая в него Publisher.
	eventController := controllers.NewEventController(publisher)

	// 4. Инициализация роутера с добавлением EventController.
	// Обратите внимание, что функция InitRoutes теперь принимает 4 параметра.
	routerEngine := router.InitRoutes(locationController, checkpointController, visitController, eventController)

	// Запуск потребителя сообщений (consumer) для обработки событий из очереди "location_events".
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

	app := &App{
		Router:    routerEngine,
		DB:        dbConn,
		RMQClient: rmqClient,
	}

	// Демонстрационная публикация события (опционально).
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

	return app, nil
}
