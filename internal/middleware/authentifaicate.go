package middleware

import (
	"net/http"
	"strings"

	"manGo/internal/auth" // Твой пакет, где лежит GenerateToken и Claims

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		// 1. Проверяем, что заголовок вообще есть
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// 2. Отсекаем префикс "Bearer "
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization format must be Bearer {token}"})
			c.Abort()
			return
		}

		tokenStr := parts[1]
		claims := &auth.Claims{}

		// 3. Парсим и валидируем токен
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte("your_secret_key"), nil // Тот же ключ, что при создании
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// 4. Прокидываем данные в контекст, чтобы хендлеры их видели
		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		
		c.Next()
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")

		if !exists || role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "This action requires admin privileges"})
			c.Abort()
			return
		}
		
		c.Next()
	}
}