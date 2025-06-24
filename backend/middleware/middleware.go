package middleware

import (
	"log"
	"net/http"

	"locator/models"
	"locator/service"

	"github.com/gin-gonic/gin"
)

// BasicAuthMiddleware проверяет только наличие API-ключа и аутентифицирует пользователя
// без проверки на права администратора
func BasicAuthMiddleware(userService *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ожидаем, что API ключ передается в заголовке "X-API-Key"
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Отсутствует API ключ"})
			return
		}

		// Пытаемся аутентифицировать пользователя на основе предоставленного API ключа
		user, err := userService.AuthenticateUser(apiKey)
		if err != nil {
			log.Printf("[Middleware] Ошибка аутентификации: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Неверный API ключ"})
			return
		}

		if user == nil {
			log.Printf("[Middleware] Критическая ошибка: AuthenticateUser вернул nil без ошибки")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
			return
		}

		// Создаем копию пользователя, чтобы избежать проблем с указателями
		userCopy := models.User{
			ID:      user.ID,
			Name:    user.Name,
			ApiKey:  user.ApiKey,
			IsAdmin: user.IsAdmin,
			QRCode:  user.QRCode,
			// Добавьте другие поля, если они есть в модели
		}

		log.Printf("[BasicAuthMiddleware] Пользователь аутентифицирован: ID=%d, Name=%s, IsAdmin=%t",
			userCopy.ID, userCopy.Name, userCopy.IsAdmin)

		// Сохраняем КОПИЮ пользователя в контексте запроса
		c.Set("user", &userCopy)
		c.Next()
	}
}

// APIKeyAuthMiddleware проверяет наличие API-ключа в заголовке и аутентифицирует пользователя,
// а также проверяет права администратора
func APIKeyAuthMiddleware(userService *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ожидаем, что API ключ передается в заголовке "X-API-Key"
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Отсутствует API ключ"})
			return
		}

		// Пытаемся аутентифицировать пользователя на основе предоставленного API ключа
		user, err := userService.AuthenticateUser(apiKey)
		if err != nil {
			log.Printf("[Middleware] Ошибка аутентификации: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Неверный API ключ"})
			return
		}

		if user == nil {
			log.Printf("[Middleware] Критическая ошибка: AuthenticateUser вернул nil без ошибки")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
			return
		}

		// Проверяем, является ли пользователь администратором
		if !user.IsAdmin {
			log.Printf("[Middleware] Отказано в доступе: пользователь не администратор: ID=%d, Name=%s",
				user.ID, user.Name)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Доступ запрещен: требуются права администратора"})
			return
		}

		// Создаем копию пользователя, чтобы избежать проблем с указателями
		userCopy := models.User{
			ID:      user.ID,
			Name:    user.Name,
			ApiKey:  user.ApiKey,
			IsAdmin: user.IsAdmin,
			QRCode:  user.QRCode,
			// Добавьте другие поля, если они есть в модели
		}

		log.Printf("[AdminAuthMiddleware] Пользователь аутентифицирован: ID=%d, Name=%s, IsAdmin=%t",
			userCopy.ID, userCopy.Name, userCopy.IsAdmin)

		// Сохраняем КОПИЮ пользователя в контексте запроса
		c.Set("user", &userCopy)
		c.Next()
	}
}
