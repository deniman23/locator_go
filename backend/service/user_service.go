package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
	"locator/models"
)

const authCacheTTL = 10 * time.Minute

// authCacheEntry holds a cached successful auth mapping (key hash → user ID + expiry).
// We never store the plaintext key or the bcrypt hash in this cache.
type authCacheEntry struct {
	userID    int
	expiresAt time.Time
}

type UserService struct {
	DAO       userRepository
	authCache sync.Map // sha256hex(providedKey) → *authCacheEntry
}

// NewUserService создаёт новый экземпляр UserService.
func NewUserService(dao userRepository) *UserService {
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

	if len(plainKey) < 6 || plainKey == "change_me" {
		return nil, "", fmt.Errorf("API ключ слишком короткий")
	}

	user := &models.User{
		Name:      name,
		ApiKey:    string(hashedKey),
		KeyLookup: sha256KeyHex(plainKey),
		IsAdmin:   isAdmin,
	}

	if err := svc.DAO.Create(user); err != nil {
		log.Printf("[UserService CreateUser] Ошибка создания пользователя: %v", err)
		return nil, "", err
	}

	qrCodeURL, err := svc.writeUserQRCode(user.ID, plainKey)
	if err != nil {
		log.Printf("[UserService CreateUser] Ошибка генерации QR кода: %v", err)
		return nil, "", err
	}
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

// AuthenticateUser проверяет API-ключ.
// Успешный ключ ищется по SHA-256 (один SELECT и один bcrypt).
// Полный перебор остаётся только для строк без key_lookup, пока пользователь не войдёт один раз.
func (svc *UserService) AuthenticateUser(providedKey string) (*models.User, error) {
	if providedKey == "" {
		return nil, fmt.Errorf("API ключ не может быть пустым")
	}

	keyHash := sha256KeyHex(providedKey)
	if svc.negativeCached(keyHash) {
		return nil, fmt.Errorf("недействительный API ключ")
	}

	if raw, ok := svc.authCache.Load(keyHash); ok {
		entry := raw.(*authCacheEntry)
		if time.Now().Before(entry.expiresAt) {
			user, err := svc.DAO.GetByID(entry.userID)
			if err == nil && user.DisabledAt == nil && bcrypt.CompareHashAndPassword([]byte(user.ApiKey), []byte(providedKey)) == nil {
				log.Printf("[AuthenticateUser] cache hit: ID=%d", user.ID)
				return user, nil
			}
			svc.authCache.Delete(keyHash)
		} else {
			svc.authCache.Delete(keyHash)
		}
	}

	if user, err := svc.DAO.GetByKeyLookup(keyHash); err == nil && user != nil {
		if accept, authErr := svc.acceptKey(user, providedKey, keyHash); authErr != nil || accept != nil {
			return accept, authErr
		}
	}

	legacy, err := svc.DAO.ListMissingKeyLookup()
	if err != nil {
		return nil, fmt.Errorf("ошибка доступа к базе данных")
	}
	for i := range legacy {
		user := &legacy[i]
		if bcrypt.CompareHashAndPassword([]byte(user.ApiKey), []byte(providedKey)) != nil {
			continue
		}
		user.KeyLookup = keyHash
		if err := svc.DAO.Update(user); err != nil {
			log.Printf("[AuthenticateUser] не удалось записать key_lookup для ID=%d: %v", user.ID, err)
		}
		return svc.finishAuth(user, keyHash)
	}

	svc.rememberNegative(keyHash)
	log.Printf("[AuthenticateUser] Недействительный API ключ")
	return nil, fmt.Errorf("недействительный API ключ")
}

func (svc *UserService) acceptKey(user *models.User, providedKey, keyHash string) (*models.User, error) {
	if bcrypt.CompareHashAndPassword([]byte(user.ApiKey), []byte(providedKey)) != nil {
		return nil, nil
	}
	return svc.finishAuth(user, keyHash)
}

func (svc *UserService) finishAuth(user *models.User, keyHash string) (*models.User, error) {
	if user.DisabledAt != nil {
		return nil, fmt.Errorf("учётная запись отключена")
	}
	svc.authCache.Store(keyHash, &authCacheEntry{
		userID:    user.ID,
		expiresAt: time.Now().Add(authCacheTTL),
	})
	log.Printf("[AuthenticateUser] Успешная аутентификация пользователя: ID=%d, Name=%s", user.ID, user.Name)
	return user, nil
}

func (svc *UserService) negativeCached(keyHash string) bool {
	raw, ok := svc.authCache.Load("neg:" + keyHash)
	if !ok {
		return false
	}
	entry := raw.(*authCacheEntry)
	if time.Now().Before(entry.expiresAt) {
		return true
	}
	svc.authCache.Delete("neg:" + keyHash)
	return false
}

func (svc *UserService) rememberNegative(keyHash string) {
	svc.authCache.Store("neg:"+keyHash, &authCacheEntry{
		expiresAt: time.Now().Add(time.Minute),
	})
}

// sha256KeyHex returns the hex-encoded SHA-256 of key. Used as cache key so no
// plaintext secret is held in memory beyond the duration of the call.
func sha256KeyHex(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
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

// UpdateUserName меняет отображаемое имя пользователя.
func (svc *UserService) UpdateUserName(id int, name string) (*models.User, error) {
	user, err := svc.DAO.GetByID(id)
	if err != nil {
		log.Printf("[UserService UpdateUserName] Пользователь с ID=%d не найден: %v", id, err)
		return nil, err
	}

	user.Name = name
	if err := svc.DAO.Update(user); err != nil {
		log.Printf("[UserService UpdateUserName] Ошибка обновления пользователя ID=%d: %v", id, err)
		return nil, err
	}

	log.Printf("[UserService UpdateUserName] Имя обновлено: ID=%d, Name=%s", user.ID, user.Name)
	return user, nil
}

func apiBaseURL() string {
	apiBase := os.Getenv("BASE_URL")
	if apiBase == "" {
		return "http://localhost:8080"
	}
	return apiBase
}

func (svc *UserService) writeUserQRCode(userID int, plainKey string) (string, error) {
	apiBase := apiBaseURL()
	qrContent := fmt.Sprintf(`{"user_id": %d, "api_key": "%s", "api_base_url": "%s"}`, userID, plainKey, apiBase)

	if err := os.MkdirAll("static/qrcode", 0o755); err != nil {
		return "", err
	}

	qrFilePath := fmt.Sprintf("static/qrcode/%d.png", userID)
	if err := qrcode.WriteFile(qrContent, qrcode.Medium, 256, qrFilePath); err != nil {
		return "", err
	}

	// Public /static/qrcode is blocked; metadata URL points at the auth-gated API.
	return fmt.Sprintf("%s/api/users/%d/qr-code-file?v=%d", apiBase, userID, time.Now().Unix()), nil
}

// invalidateAuthCacheForUser drops cached key→user mappings for userID (e.g. after key rotate).
func (svc *UserService) invalidateAuthCacheForUser(userID int) {
	svc.authCache.Range(func(k, v interface{}) bool {
		if entry, ok := v.(*authCacheEntry); ok && entry.userID == userID {
			svc.authCache.Delete(k)
		}
		return true
	})
}

// RegenerateUserQR создаёт новый API-ключ и перезаписывает PNG QR-кода с текущим BASE_URL.
// Если plainKey не пустой — используется указанный ключ вместо генерации нового.
func (svc *UserService) RegenerateUserQR(userID int, plainKey ...string) (*models.User, string, error) {
	user, err := svc.DAO.GetByID(userID)
	if err != nil {
		log.Printf("[UserService RegenerateUserQR] Пользователь с ID=%d не найден: %v", userID, err)
		return nil, "", err
	}

	var key string
	if len(plainKey) > 0 && plainKey[0] != "" {
		if len(plainKey[0]) < 16 {
			return nil, "", fmt.Errorf("API ключ слишком короткий")
		}
		key = plainKey[0]
	} else {
		key, err = generateSecureAPIKey()
		if err != nil {
			log.Printf("[UserService RegenerateUserQR] Ошибка генерации API ключа: %v", err)
			return nil, "", err
		}
	}

	hashedKey, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[UserService RegenerateUserQR] Ошибка хеширования API ключа: %v", err)
		return nil, "", err
	}

	qrCodeURL, err := svc.writeUserQRCode(userID, key)
	if err != nil {
		log.Printf("[UserService RegenerateUserQR] Ошибка генерации QR кода: %v", err)
		return nil, "", err
	}

	user.ApiKey = string(hashedKey)
	user.KeyLookup = sha256KeyHex(key)
	user.QRCode = qrCodeURL
	if err := svc.DAO.Update(user); err != nil {
		log.Printf("[UserService RegenerateUserQR] Ошибка обновления пользователя: %v", err)
		return nil, "", err
	}

	svc.invalidateAuthCacheForUser(userID)

	log.Printf("[UserService RegenerateUserQR] QR перегенерирован: ID=%d, Name=%s", user.ID, user.Name)
	return user, key, nil
}

// SetUserDisabled отключает или включает учётку, не удаляя локации и визиты.
func (svc *UserService) SetUserDisabled(id int, disabled bool, actorID int) (*models.User, error) {
	user, err := svc.DAO.GetByID(id)
	if err != nil {
		return nil, err
	}
	if disabled && user.IsAdmin && user.ID == actorID {
		admins, err := svc.DAO.GetAll()
		if err != nil {
			return nil, err
		}
		activeAdmins := 0
		for _, a := range admins {
			if a.IsAdmin && a.DisabledAt == nil {
				activeAdmins++
			}
		}
		if activeAdmins <= 1 {
			return nil, fmt.Errorf("нельзя отключить последнего администратора")
		}
	}
	if disabled {
		now := time.Now().UTC()
		user.DisabledAt = &now
	} else {
		user.DisabledAt = nil
	}
	if err := svc.DAO.Update(user); err != nil {
		return nil, err
	}
	svc.invalidateAuthCacheForUser(id)
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
