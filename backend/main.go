package main

import (
	"context"
	"locator/config"
	"locator/config/bootstrap"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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
	if ginMode == gin.ReleaseMode {
		switch strings.TrimSpace(os.Getenv("DB_PASSWORD")) {
		case "", "change_me":
			log.Fatal("DB_PASSWORD пустой или равен change_me")
		}
		switch strings.TrimSpace(os.Getenv("RABBITMQ_USER")) {
		case "", "guest":
			log.Fatal("RABBITMQ_USER пустой или равен guest")
		}
	}

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

	srv := &http.Server{Addr: ":8080", Handler: app.Router}
	go func() {
		log.Println("Сервер запущен на порту 8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("Остановка сервера")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	app.RMQClient.Close()
}
