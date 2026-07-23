package controllers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"locator/models"
	"locator/service"

	"github.com/gin-gonic/gin"
)

func serveQRCodePNG(ctx *gin.Context, userID int) {
	qrFilePath := fmt.Sprintf("static/qrcode/%d.png", userID)
	ctx.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	ctx.Header("Pragma", "no-cache")
	ctx.Header("Expires", "0")
	ctx.File(qrFilePath)
}

// UserController отвечает за обработку запросов, связанных с пользователями.
type UserController struct {
	Service        *service.UserService
	CommandService *service.DeviceCommandService
}

// NewUserController создаёт новый экземпляр UserController.
func NewUserController(svc *service.UserService, cmdSvc *service.DeviceCommandService) *UserController {
	return &UserController{Service: svc, CommandService: cmdSvc}
}

func (uc *UserController) CreateUser(ctx *gin.Context) {
	// Извлекаем текущего пользователя из контекста (например, через middleware авторизации)
	currentUserInterface, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Необходима авторизация"})
		return
	}
	currentUser, ok := currentUserInterface.(*models.User)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка определения текущего пользователя"})
		return
	}

	var req struct {
		Name    string `json:"name" binding:"required"`
		IsAdmin bool   `json:"is_admin"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные запроса"})
		return
	}

	// Если запрошено создание администратора, проверяем, что запрос исходит от администратора.
	if req.IsAdmin && !currentUser.IsAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Нет прав для создания администратора"})
		return
	}

	// Создаем пользователя через UserService.
	// После создания пользователю сгенерируется QR‑код с данными (JSON: user_id и api_key).
	user, _, err := uc.Service.CreateUser(req.Name, req.IsAdmin)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания пользователя"})
		return
	}
	// Поле API‑ключа не выводится в JSON благодаря тегу json:"-" в модели.
	ctx.JSON(http.StatusOK, user)
}

func (uc *UserController) GetUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
		return
	}

	user, err := uc.Service.GetUserByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}
	ctx.JSON(http.StatusOK, user)
}

// UpdateUser обрабатывает PUT-запрос для изменения имени пользователя.
func (uc *UserController) UpdateUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные запроса"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Имя не может быть пустым"})
		return
	}

	user, err := uc.Service.UpdateUserName(id, name)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}
	ctx.JSON(http.StatusOK, user)
}

func (uc *UserController) GetAllUsers(ctx *gin.Context) {
	users, err := uc.Service.GetAllUsers()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения пользователей"})
		return
	}
	ctx.JSON(http.StatusOK, users)
}

// GetQRCode возвращает JSON с данными QR‑кода для текущего пользователя.
func (uc *UserController) GetQRCode(ctx *gin.Context) {
	// Извлекаем текущего пользователя (например, через middleware авторизации).
	currentUserInterface, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Необходима авторизация"})
		return
	}
	currentUser, ok := currentUserInterface.(*models.User)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка определения текущего пользователя"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"user_id": currentUser.ID,
		"qr_code": currentUser.QRCode,
	})
}

// GetQRCodeFile возвращает напрямую изображение QR‑кода для текущего пользователя.
func (uc *UserController) GetQRCodeFile(ctx *gin.Context) {
	// Извлекаем текущего пользователя.
	currentUserInterface, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Необходима авторизация"})
		return
	}
	currentUser, ok := currentUserInterface.(*models.User)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка определения текущего пользователя"})
		return
	}

	// Формируем путь к файлу QR‑кода.
	serveQRCodePNG(ctx, currentUser.ID)
}

// GetUserQRCode позволяет администратору получить QR‑код другого пользователя по его ID.
// Доступно только для пользователей с isAdmin = true.
func (uc *UserController) GetUserQRCode(ctx *gin.Context) {
	// Извлекаем текущего пользователя.
	currentUserInterface, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Необходима авторизация"})
		return
	}
	currentUser, ok := currentUserInterface.(*models.User)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка определения текущего пользователя"})
		return
	}

	// Проверяем, что текущий пользователь является администратором.
	if !currentUser.IsAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Доступ разрешен только администраторам"})
		return
	}

	// Получаем ID целевого пользователя из параметра пути.
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
		return
	}

	// Получаем целевого пользователя.
	targetUser, err := uc.Service.GetUserByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"user_id": targetUser.ID,
		"qr_code": targetUser.QRCode,
	})

}

// GetUserQRCodeFile возвращает изображение QR-кода указанного пользователя.
// Доступно только для администраторов.
func (uc *UserController) GetUserQRCodeFile(ctx *gin.Context) {
	// Извлекаем текущего пользователя
	currentUserInterface, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Необходима авторизация"})
		return
	}
	currentUser, ok := currentUserInterface.(*models.User)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка определения текущего пользователя"})
		return
	}

	// Проверяем, что текущий пользователь является администратором
	if !currentUser.IsAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Доступ разрешен только администраторам"})
		return
	}

	// Получаем ID целевого пользователя из параметра пути
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
		return
	}

	// Получаем целевого пользователя
	targetUser, err := uc.Service.GetUserByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	serveQRCodePNG(ctx, targetUser.ID)
}

// ServeStaticQRCode — GET /static/qrcode/:filename (без кэша, до gin.Static).
func (uc *UserController) ServeStaticQRCode(ctx *gin.Context) {
	name := ctx.Param("filename")
	if len(name) < 5 || name[len(name)-4:] != ".png" {
		ctx.Status(http.StatusNotFound)
		return
	}
	id, err := strconv.Atoi(name[:len(name)-4])
	if err != nil || id <= 0 {
		ctx.Status(http.StatusNotFound)
		return
	}
	serveQRCodePNG(ctx, id)
}

// PostRegenerateUserQR — POST /api/admin/users/:id/regenerate-qr
// Создаёт новый API-ключ, перезаписывает PNG и опционально ставит config_update на устройство.
func (uc *UserController) PostRegenerateUserQR(ctx *gin.Context) {
	currentUserInterface, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Необходима авторизация"})
		return
	}
	currentUser, ok := currentUserInterface.(*models.User)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка определения текущего пользователя"})
		return
	}
	if !currentUser.IsAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Доступ разрешен только администраторам"})
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
		return
	}

	var body struct {
		PushToDevice *bool  `json:"push_to_device"`
		APIKey       string `json:"api_key"`
	}
	_ = ctx.ShouldBindJSON(&body)
	pushToDevice := body.PushToDevice == nil || *body.PushToDevice

	var user *models.User
	var plainKey string
	if body.APIKey != "" {
		user, plainKey, err = uc.Service.RegenerateUserQR(id, body.APIKey)
	} else {
		user, plainKey, err = uc.Service.RegenerateUserQR(id)
	}
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	response := gin.H{
		"id":       user.ID,
		"name":     user.Name,
		"is_admin": user.IsAdmin,
		"qr_code":  user.QRCode,
		"api_key":  plainKey,
	}

	if pushToDevice && uc.CommandService != nil {
		apiBase := os.Getenv("BASE_URL")
		if apiBase == "" {
			apiBase = "http://localhost:8080"
		}
		payload := map[string]interface{}{
			"api_base_url": apiBase,
			"api_key":      plainKey,
			"user_id":      user.ID,
		}
		cmd, err := uc.CommandService.EnqueueCommand(user.ID, models.DeviceCommandTypeConfigUpdate, payload)
		if err != nil {
			log.Printf("[PostRegenerateUserQR] config_update не поставлен в очередь: %v", err)
		} else {
			response["config_command_id"] = cmd.ID
		}
	}

	ctx.JSON(http.StatusOK, response)
}

// GetCurrentUser возвращает информацию о текущем аутентифицированном пользователе
func (uc *UserController) GetCurrentUser(c *gin.Context) {
	// Получаем пользователя из контекста (установлен middleware)
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не аутентифицирован"})
		return
	}

	user, ok := userInterface.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения данных пользователя"})
		return
	}

	// Возвращаем информацию о пользователе
	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"name":     user.Name,
		"is_admin": user.IsAdmin,
		"qr_code":  user.QRCode,
	})
}
