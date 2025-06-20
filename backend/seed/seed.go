package seed

import (
	"errors"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"locator/models"
)

// DefaultAdmin SeedDefaultAdmin создает дефолтного администратора, если его нет в базе.
func DefaultAdmin(db *gorm.DB) {
	// Читаем данные из переменных окружения
	defaultName := os.Getenv("DEFAULT_ADMIN_NAME")
	defaultAPIKey := os.Getenv("DEFAULT_ADMIN_API_KEY")

	if defaultName == "" || defaultAPIKey == "" {
		log.Println("Данные дефолтного администратора (DEFAULT_ADMIN_NAME или DEFAULT_ADMIN_API_KEY) не заданы в .env")
		return
	}

	// Проверяем, существует ли уже пользователь с таким API-ключом
	var admin models.User
	err := db.Where("api_key = ?", defaultAPIKey).First(&admin).Error
	if err == nil {
		log.Println("Дефолтный администратор уже существует")
		return
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Ошибка поиска пользователя: %v", err)
		return
	}

	// Хэшируем API ключ с использованием bcrypt
	hashedKey, err := bcrypt.GenerateFromPassword([]byte(defaultAPIKey), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Ошибка хеширования API ключа: %v", err)
		return
	}

	// Если пользователь не найден, создаем нового с хэшированным API ключом
	admin = models.User{
		Name:    defaultName,
		ApiKey:  string(hashedKey),
		IsAdmin: true,
		// Прочие поля можно заполнить при необходимости
	}
	if err := db.Create(&admin).Error; err != nil {
		log.Printf("Ошибка создания дефолтного администратора: %v", err)
		return
	}
	log.Println("Дефолтный администратор успешно создан")
}
