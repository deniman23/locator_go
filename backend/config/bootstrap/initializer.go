package bootstrap

import (
	"fmt"
	"log"
	"os"
	"time"

	"locator/config"
	"locator/config/messaging"
	"locator/controllers"
	"locator/dao"
	"locator/models"
	"locator/router"
	"locator/seed"
	"locator/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/logger"
)

// App содержит зависимости приложения.
type App struct {
	Router    *gin.Engine
	DB        interface{}
	RMQClient *messaging.RabbitMQClient
}

// InitializeApp собирает все зависимости приложения и возвращает готовый инстанс App.
// Функция принимает параметр dbLogger для настройки уровня логирования SQL-запросов.
func InitializeApp(dbLogger logger.Interface) (*App, error) {
	// 1. Инициализация подключения к БД с указанным логгером.
	dbConn := config.InitDB(dbLogger)

	if err := dbConn.AutoMigrate(
		&models.User{},
		&models.Location{},
		&models.Checkpoint{},
		&models.Visit{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate failed: %w", err)
	}

	seed.DefaultAdmin(dbConn)

	// 2. Настройка параметров подключения к RabbitMQ из окружения.
	user := os.Getenv("RABBITMQ_USER")
	if user == "" {
		user = "guest"
	}
	pass := os.Getenv("RABBITMQ_PASS")
	if pass == "" {
		pass = "guest"
	}
	host := os.Getenv("RABBITMQ_HOST")
	if host == "" {
		host = "rabbitmq"
	}
	port := os.Getenv("RABBITMQ_PORT")
	if port == "" {
		port = "5672"
	}

	// 3. Ждём, когда RabbitMQ станет доступен, и создаём клиент.
	var rmqClient *messaging.RabbitMQClient
	var err error
	for i := 0; i < 15; i++ {
		rmqURL := fmt.Sprintf("amqp://%s:%s@%s:%s/", user, pass, host, port)
		rmqClient, err = messaging.NewRabbitMQClient(rmqURL)
		if err == nil {
			log.Printf("Подключились к RabbitMQ по %s", rmqURL)
			break
		}
		log.Printf("⏳ waiting for RabbitMQ at %s:%s … (%v)", host, port, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("rabbitmq connect failed: %w", err)
	}

	// Создаём Publisher с exchange="" и routing key="location_events"
	publisher := messaging.NewPublisher(rmqClient, "", "location_events")

	// Объявляем очередь "location_events"
	queue, err := rmqClient.Channel.QueueDeclare(
		"location_events",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		log.Printf("Ошибка объявления очереди: %v", err)
	} else {
		log.Printf("Очередь '%s' объявлена успешно", queue.Name)
	}

	// 4. Инициализация DAO, сервисов и контроллеров

	// Location
	locationDAO := dao.NewLocationDAO(dbConn)
	locationService := service.NewLocationService(locationDAO)
	locationController := controllers.NewLocationController(locationService, publisher)

	// Checkpoint и Visit
	checkpointDAO := dao.NewCheckpointDAO(dbConn)
	checkpointService := service.NewCheckpointService(checkpointDAO)
	visitDAO := dao.NewVisitDAO(dbConn)
	visitService := service.NewVisitService(visitDAO)
	checkpointController := controllers.NewCheckpointController(
		checkpointService, locationService, visitService, publisher,
	)
	visitController := controllers.NewVisitController(visitService)

	// EventController
	eventController := controllers.NewEventController(publisher)

	// User
	userDAO := dao.NewUserDAO(dbConn)
	userService := service.NewUserService(userDAO)
	userController := controllers.NewUserController(userService)

	// 5. Инициализация роутера
	routerEngine := router.InitRoutes(
		locationController,
		checkpointController,
		visitController,
		eventController,
		userController,
		userService,
	)

	// 6. Запуск background-потребителя сообщений
	visitEventProcessor := service.NewVisitEventProcessor(checkpointService, visitService)
	consumer := messaging.NewConsumer(rmqClient, queue.Name)
	go func() {
		if err := consumer.Consume(func(message []byte) error {
			return visitEventProcessor.ProcessEvent(message)
		}); err != nil {
			log.Printf("Ошибка при потреблении сообщений: %v", err)
		}
	}()

	// 7. (Опционально) публикация демо-события
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
