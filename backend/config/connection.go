package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB инициализирует подключение к базе данных Postgres через GORM.
// Принимает параметр dbLogger для логирования SQL-запросов. Если dbLogger равен nil,
// используется логгер по умолчанию с уровнем Info.
func InitDB(dbLogger logger.Interface) *gorm.DB {
	// Загрузка переменных окружения из файла .env.
	// Если файл не найден, используются системные переменные.
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Не удалось загрузить файл .env, используется системные переменные окружения")
	}

	// Чтение настроек подключения из переменных окружения.
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	dbname := os.Getenv("DB_NAME")
	password := os.Getenv("DB_PASSWORD")
	sslmode := os.Getenv("DB_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}

	// Формирование строки подключения.
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	// Если переданный dbLogger равен nil, устанавливаем логгер по умолчанию.
	if dbLogger == nil {
		dbLogger = logger.Default.LogMode(logger.Info)
	}

	// Открытие подключения к базе данных с использованием заданного логгера.
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: dbLogger,
	})
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Ошибка получения sql.DB: %v", err)
	}

	// Настройка пула соединений.
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Подключение к БД успешно установлено")
	return db
}
