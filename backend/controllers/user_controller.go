package controllers

import (
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
	// Извлекаем текущего пользователя, установленного, например, через middleware авторизации.
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

	// Если запрошено создание суперпользователя, проверяем, что запрос исходит от админа.
	if req.IsAdmin && !currentUser.IsAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Нет прав для создания администратора"})
		return
	}

	// Создаем пользователя, используя UserService.
	// Здесь обрабатываем три возвращаемых значения. Нам не нужен plaintext API-ключ в ответе, поэтому отбросим его.
	user, _, err := uc.Service.CreateUser(req.Name, req.IsAdmin)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания пользователя"})
		return
	}
	// Поле API ключа не выводится в JSON, благодаря тегу json:"-"
	ctx.JSON(http.StatusOK, user)
}

// GetUser обрабатывает запрос на получение информации о пользователе по его ID.
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

// GetAllUsers обрабатывает запрос на получение списка всех пользователей.
func (uc *UserController) GetAllUsers(ctx *gin.Context) {
	users, err := uc.Service.GetAllUsers()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения пользователей"})
		return
	}
	ctx.JSON(http.StatusOK, users)
}
