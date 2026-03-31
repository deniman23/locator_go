package main

import (
	"locator/config"
	"locator/config/bootstrap"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func main() {
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Fatalf("Ошибка создания директории логов: %v", err)
	}
	if err := os.MkdirAll("static/qrcode", 0755); err != nil {
		log.Fatalf("Ошибка создания директории QR-кодов: %v", err)
	}

	config.InitLogger("logs/app.log")

	dbLogger := config.InitDBQueryLogger("logs/db.log")

	ginMode := strings.TrimSpace(os.Getenv("GIN_MODE"))
	if ginMode == "" {
		ginMode = gin.ReleaseMode
	}
	gin.SetMode(ginMode)

	// Инициализируем приложение и передаём логгер для работы с БД.
	app, err := bootstrap.InitializeApp(dbLogger)
	if err != nil {
		log.Fatalf("Ошибка инициализации приложения: %v", err)
	}

	// По умолчанию не доверяем прокси, чтобы не принимать X-Forwarded-* от любого источника.
	trustedProxiesRaw := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES"))
	if trustedProxiesRaw == "" {
		if err := app.Router.SetTrustedProxies(nil); err != nil {
			log.Fatalf("Ошибка настройки trusted proxies: %v", err)
		}
	} else {
		rawList := strings.Split(trustedProxiesRaw, ",")
		trustedProxies := make([]string, 0, len(rawList))
		for _, proxy := range rawList {
			proxy = strings.TrimSpace(proxy)
			if proxy != "" {
				trustedProxies = append(trustedProxies, proxy)
			}
		}
		if err := app.Router.SetTrustedProxies(trustedProxies); err != nil {
			log.Fatalf("Ошибка настройки trusted proxies: %v", err)
		}
	}

	defer app.RMQClient.Close()
	log.Println("Сервер запущен на порту 8080")

	if err := app.Router.Run(":8080"); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
