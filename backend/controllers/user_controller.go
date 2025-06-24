package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"locator/models"
	"locator/service"

	"github.com/gin-gonic/gin"
)

// UserController отвечает за обработку запросов, связанных с пользователями.
type UserController struct {
	Service *service.UserService
}

// NewUserController создаёт новый экземпляр UserController.
func NewUserController(svc *service.UserService) *UserController {
	return &UserController{Service: svc}
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
	qrFilePath := fmt.Sprintf("static/qrcode/%d.png", currentUser.ID)
	// Отдаем файл с MIME-типом image/png.
	ctx.File(qrFilePath)
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

	// Формируем путь к файлу QR-кода
	qrFilePath := fmt.Sprintf("static/qrcode/%d.png", targetUser.ID)

	// Отдаем файл с MIME-типом image/png
	ctx.File(qrFilePath)
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
