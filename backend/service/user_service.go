package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
	"locator/dao"
	"locator/models"
)

type UserService struct {
	DAO *dao.UserDAO
}

// NewUserService создаёт новый экземпляр UserService.
func NewUserService(dao *dao.UserDAO) *UserService {
	log.Println("[UserService] Инициализация сервиса пользователей")
	return &UserService{DAO: dao}
}

// generateSecureAPIKey генерирует 32-байтовый ключ и возвращает его в виде шестнадцатеричной строки.
func generateSecureAPIKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}

// CreateUser Обновленный метод CreateUser
func (svc *UserService) CreateUser(name string, isAdmin bool, forceAPIKey ...string) (*models.User, string, error) {
	var plainKey string
	var err error

	// Используем forceAPIKey только если он явно передан (для сидера)
	if len(forceAPIKey) > 0 && forceAPIKey[0] != "" {
		plainKey = forceAPIKey[0]
	} else {
		// Для всех обычных пользователей генерируем новый случайный ключ
		plainKey, err = generateSecureAPIKey()
		if err != nil {
			log.Printf("[UserService CreateUser] Ошибка генерации API ключа: %v", err)
			return nil, "", err
		}
	}

	hashedKey, err := bcrypt.GenerateFromPassword([]byte(plainKey), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[UserService CreateUser] Ошибка хеширования API ключа: %v", err)
		return nil, "", err
	}

	user := &models.User{
		Name:    name,
		ApiKey:  string(hashedKey),
		IsAdmin: isAdmin,
	}

	if err := svc.DAO.Create(user); err != nil {
		log.Printf("[UserService CreateUser] Ошибка создания пользователя: %v", err)
		return nil, "", err
	}

	// Формируем содержимое для QR‑кода на основе plaintext API‑ключа.
	qrContent := fmt.Sprintf(`{"user_id": %d, "api_key": "%s"}`, user.ID, plainKey)

	// Определяем путь для сохранения изображения QR‑кода.
	// Убедитесь, что папка static/qrcode существует и доступна для записи.
	qrFilePath := fmt.Sprintf("static/qrcode/%d.png", user.ID)
	err = qrcode.WriteFile(qrContent, qrcode.Medium, 256, qrFilePath)
	if err != nil {
		log.Printf("[UserService CreateUser] Ошибка генерации QR кода: %v", err)
		return nil, "", err
	}

	// Формируем публичную ссылку на QR‑код, используя BASE_URL из переменных окружения
	baseURL := os.Getenv("BASE_URL")

	qrCodeURL := fmt.Sprintf("%s/static/qrcode/%d.png", baseURL, user.ID)
	user.QRCode = qrCodeURL

	// Обновляем запись пользователя, сохраняя ссылку на QR‑код.
	if err := svc.DAO.Update(user); err != nil {
		log.Printf("[UserService CreateUser] Ошибка обновления пользователя с QR кодом: %v", err)
		return nil, "", err
	}

	log.Printf("[UserService CreateUser] Пользователь создан: ID=%d, Name=%s, IsAdmin=%t", user.ID, user.Name, user.IsAdmin)
	// Возвращаем plaintext API‑ключ только при создании (в дальнейшем не показываем его)
	return user, plainKey, nil
}

// AuthenticateUser проверяет, соответствует ли предоставленный API‑ключ хешированному значению в базе.
// AuthenticateUser проверяет API-ключ с правильной обработкой указателей
func (svc *UserService) AuthenticateUser(providedKey string) (*models.User, error) {
	if providedKey == "" {
		return nil, fmt.Errorf("API ключ не может быть пустым")
	}

	users, err := svc.DAO.GetAll()
	if err != nil {
		return nil, fmt.Errorf("ошибка доступа к базе данных")
	}

	// Вместо работы с указателями в цикле, найдем ID подходящего пользователя
	var matchedUserID int = -1

	for _, user := range users {
		if err := bcrypt.CompareHashAndPassword([]byte(user.ApiKey), []byte(providedKey)); err == nil {
			matchedUserID = user.ID
			log.Printf("[AuthenticateUser] Найдено совпадение для пользователя ID=%d, Name=%s",
				user.ID, user.Name)
			break
		}
	}

	// Если нашли подходящего пользователя, загружаем его заново из базы по ID
	if matchedUserID != -1 {
		matchedUser, err := svc.DAO.GetByID(matchedUserID)
		if err != nil {
			return nil, fmt.Errorf("ошибка получения данных пользователя")
		}
		log.Printf("[AuthenticateUser] Успешная аутентификация пользователя: ID=%d, Name=%s",
			matchedUser.ID, matchedUser.Name)
		return matchedUser, nil
	}

	log.Printf("[AuthenticateUser] Недействительный API ключ")
	return nil, fmt.Errorf("недействительный API ключ")
}

// GetUserByID возвращает пользователя по его ID.
func (svc *UserService) GetUserByID(id int) (*models.User, error) {
	user, err := svc.DAO.GetByID(id)
	if err != nil {
		log.Printf("[UserService GetUserByID] Пользователь с ID=%d не найден: %v", id, err)
		return nil, err
	}
	return user, nil
}

// GetAllUsers возвращает список всех пользователей.
func (svc *UserService) GetAllUsers() ([]models.User, error) {
	users, err := svc.DAO.GetAll()
	if err != nil {
		log.Printf("[UserService GetAllUsers] Ошибка получения пользователей: %v", err)
		return nil, err
	}
	return users, nil
}
