package config

import (
	"log"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"gorm.io/gorm/logger"
)

// InitLogger инициализирует основной логгер приложения с ежедневной ротацией логов.
// Логи сохраняются в файлах:
// - активный лог: logs/app.log
// - архивы: logs/app.log.YYYYMMDD (за предыдущие 3 дня, всего 4 файла)
func InitLogger(logPath string) {
	writer, err := rotatelogs.New(
		// Например: "logs/app.log.20250613"
		logPath+".%Y%m%d",
		rotatelogs.WithRotationTime(24*time.Hour), // новая ротация каждые 24 часа
		rotatelogs.WithRotationCount(4),           // сохранять максимум 4 файла (активный + 3 архива)
	)
	if err != nil {
		log.Fatalf("Не удалось настроить ротацию логов: %v", err)
	}

	// Настраиваем стандартный логгер
	log.SetOutput(writer)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("Логгер инициализирован, логи пишутся в:", logPath)
}

// InitDBQueryLogger инициализирует логгер для логирования SQL-запросов к БД с ротацией логов.
// В продакшене рекомендуется использовать уровень logger.Warn, чтобы фиксировать только важные события и медленные запросы.
func InitDBQueryLogger(logPath string) logger.Interface {
	writer, err := rotatelogs.New(
		// Например: "logs/db.log.20250613"
		logPath+".%Y%m%d",
		rotatelogs.WithRotationTime(24*time.Hour), // новая ротация каждые 24 часа
		rotatelogs.WithRotationCount(4),           // сохранять максимум 4 файла (активный + 3 архива)
	)
	if err != nil {
		log.Fatalf("Не удалось настроить ротацию логов для DB запросов: %v", err)
	}

	dbLogger := logger.New(
		log.New(writer, "[DB] ", log.Ldate|log.Ltime|log.Lshortfile),
		logger.Config{
			SlowThreshold:             time.Second, // Порог для медленных запросов (например, 1 секунда)
			LogLevel:                  logger.Warn, // Логирование только предупреждений и ошибок в продакшене
			IgnoreRecordNotFoundError: true,        // Игнорировать ошибку "record not found"
			Colorful:                  false,       // Отключаем цветной вывод для файлов
		},
	)
	return dbLogger
}
