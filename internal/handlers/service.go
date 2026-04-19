package handlers

import (
	"log"
	"net/http"
	"net/url"

	// <-- Add this import
	"manGo/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB) {

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "server running"})
	})

	r.POST("/services", func(c *gin.Context) {
		var service models.Service

		if err := c.BindJSON(&service); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON input"})
			return
		}

		// Validate URL
		if service.URL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "URL is required"})
			return
		}

		if _, err := url.Parse(service.URL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid URL format"})
			return
		}

		// Create service
		result := db.Create(&service)
		if result.Error != nil {
			log.Printf("handlers: failed to create service: %v", result.Error)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create service"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "service created successfully",
			"service": service,
		})
	})

	r.GET("/services", func(c *gin.Context) {
		var services []models.Service
		result := db.Find(&services)
		if result.Error != nil {
			log.Printf("handlers: failed to fetch services: %v", result.Error)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch services"})
			return
		}

		if len(services) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"count":    0,
				"services": []models.Service{},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"count":    len(services),
			"services": services,
		})
	})

	r.GET("/services/:id/checks", func(c *gin.Context) {
		serviceID := c.Param("id")

		var checks []models.Check
		result := db.Where("service_id = ?", serviceID).
			Order("id DESC").
			Limit(100).
			Find(&checks)

		if result.Error != nil {
			log.Printf("handlers: failed to fetch checks: %v", result.Error)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch checks"})
			return
		}

		if len(checks) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"count":  0,
				"checks": []models.Check{},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"count":  len(checks),
			"checks": checks,
		})
	})

	// Add authentication to a service
	r.POST("/services/:id/auth", func(c *gin.Context) {
		serviceID := c.Param("id")

		var req struct {
			LoginURL    string `json:"login_url" binding:"required"`
			Username    string `json:"username" binding:"required"`
			Password    string `json:"password" binding:"required"`
			UsernameKey string `json:"username_key" binding:"required"`
			PasswordKey string `json:"password_key" binding:"required"`
			MonitorURL  string `json:"monitor_url"`
		}

		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
			return
		}

		// Check if service exists
		var service models.Service
		if err := db.First(&service, serviceID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
			return
		}

		// Check if auth already exists
		var existingAuth models.ServiceAuth
		if err := db.Where("service_id = ?", serviceID).First(&existingAuth).Error; err == nil {
			// Update existing auth
			result := db.Model(&existingAuth).Updates(&models.ServiceAuth{
				LoginURL:    req.LoginURL,
				Username:    req.Username,
				Password:    req.Password,
				UsernameKey: req.UsernameKey,
				PasswordKey: req.PasswordKey,
				MonitorURL:  req.MonitorURL,
			})
			if result.Error != nil {
				log.Printf("handlers: failed to update auth: %v", result.Error)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update authentication"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "authentication updated successfully"})
			return
		}

		// Create new auth
		auth := models.ServiceAuth{
			ServiceID:   uint(service.ID),
			LoginURL:    req.LoginURL,
			Username:    req.Username,
			Password:    req.Password,
			UsernameKey: req.UsernameKey,
			PasswordKey: req.PasswordKey,
			MonitorURL:  req.MonitorURL,
		}

		result := db.Create(&auth)
		if result.Error != nil {
			log.Printf("handlers: failed to create auth: %v", result.Error)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add authentication"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "authentication added successfully",
			"auth":    gin.H{"service_id": auth.ServiceID, "login_url": auth.LoginURL},
		})
	})

	// Get authentication status for a service
	r.GET("/services/:id/auth", func(c *gin.Context) {
		serviceID := c.Param("id")

		var auth models.ServiceAuth
		result := db.Where("service_id = ?", serviceID).First(&auth)

		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusOK, gin.H{"authenticated": false})
				return
			}
			log.Printf("handlers: failed to fetch auth: %v", result.Error)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch authentication"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"authenticated": true,
			"login_url":     auth.LoginURL,
			"monitor_url":   auth.MonitorURL,
			"username_key":  auth.UsernameKey,
			"password_key":  auth.PasswordKey,
		})
	})

	// Delete authentication from a service
	r.DELETE("/services/:id/auth", func(c *gin.Context) {
		serviceID := c.Param("id")

		result := db.Where("service_id = ?", serviceID).Delete(&models.ServiceAuth{})
		if result.Error != nil {
			log.Printf("handlers: failed to delete auth: %v", result.Error)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete authentication"})
			return
		}

		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "authentication not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "authentication removed successfully"})
	})

	// Update a service
	r.PUT("/services/:id", func(c *gin.Context) {
		serviceID := c.Param("id")

		var req struct {
			URL string `json:"url" binding:"required"`
		}

		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
			return
		}

		if _, err := url.Parse(req.URL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid URL format"})
			return
		}

		var service models.Service
		if err := db.First(&service, "id = ?", serviceID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
			return
		}

		service.URL = req.URL
		if err := db.Save(&service).Error; err != nil {
			log.Printf("handlers: failed to update service: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update service"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "service updated successfully",
			"service": service,
		})
	})

	// Delete a service
	r.DELETE("/services/:id", func(c *gin.Context) {
		serviceID := c.Param("id")

		var service models.Service
		// First check if the service exists
		if err := db.First(&service, "id = ?", serviceID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
			return
		}

		db.Where("service_id = ?", service.ID).Delete(&models.ServiceAuth{})
		db.Where("service_id = ?", service.ID).Delete(&models.Check{})

		// Delete the service itself
		if err := db.Delete(&service).Error; err != nil {
			log.Printf("handlers: failed to delete service: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete service"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "service deleted successfully"})
	})
}
