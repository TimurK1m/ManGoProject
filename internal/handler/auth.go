package handler

import (
	"net/http"

	"Project/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	auth    *service.AuthService
	service *service.ServiceService
}

func NewHandler(auth *service.AuthService, service *service.ServiceService) *Handler {
	return &Handler{
		auth:    auth,
		service: service,
	}
}

type registerInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Register(c *gin.Context) {
	var input registerInput

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.auth.Register(input.Email, input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "registered"})
}

func (h *Handler) Login(c *gin.Context) {
	var input registerInput

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.auth.Login(input.Email, input.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}