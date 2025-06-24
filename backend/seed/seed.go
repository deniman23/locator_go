package seed

import (
	"errors"
	"log"
	"os"

	"gorm.io/gorm"
	"locator/dao"
	"locator/models"
	"locator/service"
)

// DefaultAdmin DefaultAdminSeed создаёт дефолтного администратора (если его нет в базе)
// с использованием UserService, что обеспечивает генерацию QR кода и корректное хэширование API ключа.
func DefaultAdmin(db *gorm.DB) {
	// Читаем данные из переменных окружения
	defaultName := os.Getenv("DEFAULT_ADMIN_NAME")
	defaultAPIKey := os.Getenv("DEFAULT_ADMIN_API_KEY")
	if defaultName == "" || defaultAPIKey == "" {
		log.Println("Данные дефолтного администратора (DEFAULT_ADMIN_NAME или DEFAULT_ADMIN_API_KEY) не заданы в переменных окружения")
		return
	}

	// Проверяем, существует ли уже администратор с заданным именем и флагом is_admin=true.
	var admin models.User
	err := db.Where("name = ? AND is_admin = ?", defaultName, true).First(&admin).Error
	if err == nil {
		log.Println("Дефолтный администратор уже существует")
		return
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Ошибка при поиске дефолтного администратора: %v", err)
		return
	}

	// Инициализируем DAO и создаём экземпляр UserService
	userDAO := dao.NewUserDAO(db)
	userService := service.NewUserService(userDAO)

	// Создаём администратора через UserService с явным указанием API ключа
	user, plainKey, err := userService.CreateUser(defaultName, true, defaultAPIKey)
	if err != nil {
		log.Printf("Ошибка создания дефолтного администратора: %v", err)
		return
	}

	// Поле QRCode в модели обновится внутри UserService с публичной ссылкой на QR картинку,
	// а в логах мы получаем также plaintext API ключ (он доступен только при создании).
	log.Printf("Дефолтный администратор успешно создан: %s (ID: %d). Plain API ключ: %s", user.Name, user.ID, plainKey)
}
