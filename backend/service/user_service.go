package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"locator/dao"
	"locator/models"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
)

// UserService отвечает за бизнес-логику, связанную с пользователями.
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

func (svc *UserService) CreateUser(name string, isAdmin bool) (*models.User, string, error) {
	// Если это дефолтный пользователь и задан ключ из ENV, используем его
	envKey := os.Getenv("DEFAULT_ADMIN_API_KEY")
	var plainKey string
	var err error

	if envKey != "" {
		plainKey = envKey
	} else {
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

	log.Printf("[UserService CreateUser] Пользователь создан: ID=%d, Name=%s, IsAdmin=%t", user.ID, user.Name, user.IsAdmin)
	// Возвращаем plaintext API-ключ только при создании (его больше не показываем)
	return user, plainKey, nil
}

// AuthenticateUser проверяет, соответствует ли предоставленный API-ключ хэшированному значению в записях.
func (svc *UserService) AuthenticateUser(providedKey string) (*models.User, error) {
	users, err := svc.DAO.GetAll()
	if err != nil {
		log.Printf("[UserService AuthenticateUser] Ошибка получения пользователей: %v", err)
		return nil, err
	}

	for _, user := range users {
		if err := bcrypt.CompareHashAndPassword([]byte(user.ApiKey), []byte(providedKey)); err == nil {
			log.Printf("[UserService AuthenticateUser] Аутентификация успешна для пользователя ID=%d", user.ID)
			return &user, nil
		}
	}

	log.Printf("[UserService AuthenticateUser] Недействительный API ключ")
	return nil, errors.New("недействительный API ключ")
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
