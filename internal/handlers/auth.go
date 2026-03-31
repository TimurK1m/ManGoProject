package handlers

import (
	"manGo/internal/auth"
	"manGo/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Login(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.BindJSON(&input); err != nil {
			return
		}

		var user models.User
		if err := db.Where("username = ?", input.Username).First(&user).Error; err != nil {
			c.JSON(401, gin.H{"error": "User not found"})
			return
		}

		if !user.CheckPassword(input.Password) {
			c.JSON(401, gin.H{"error": "Wrong password"})
			return
		}

		token, _ := auth.GenerateToken(user.ID, user.Role)
		c.JSON(200, gin.H{"token": token})
	}
}

func Register(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        var input struct {
            Username string `json:"username" binding:"required"`
            Password string `json:"password" binding:"required"`
            Role     string `json:"role"` // Можно передать "admin", если нужно
        }
        if err := c.BindJSON(&input); err != nil {
            c.JSON(400, gin.H{"error": "Invalid input"})
            return
        }

        user := models.User{Username: input.Username, Role: input.Role}
        if user.Role == "" { user.Role = "user" } // По дефолту обычный юзер

        // Хешируем пароль перед сохранением
        if err := user.HashPassword(input.Password); err != nil {
            c.JSON(500, gin.H{"error": "Failed to hash password"})
            return
        }

        if err := db.Create(&user).Error; err != nil {
            c.JSON(400, gin.H{"error": "Username already exists"})
            return
        }

        c.JSON(201, gin.H{"message": "User created successfully"})
    }
}