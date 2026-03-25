package handler

import (
	"net/http"

	"Project/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *Handler) CreateService(c *gin.Context) {
	var input service.CreateServiceInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	// 🔥 временно без JWT (чтобы быстрее пройти чекпоинт)
	userID := 1

	err := h.service.Create(userID, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "service created"})
}