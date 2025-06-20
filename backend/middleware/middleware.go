package middleware

import (
	"net/http"

	"locator/service"

	"github.com/gin-gonic/gin"
)

// APIKeyAuthMiddleware проверяет наличие API-ключа в заголовке и аутентифицирует пользователя.
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Неверный API ключ"})
			return
		}

		// Сохраняем информацию о пользователе в контексте запроса для дальнейшего использования в контроллерах
		c.Set("user", user)
		c.Next()
	}
}
