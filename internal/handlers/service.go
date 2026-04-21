package handlers

import (
	"log"
	"net/http"
	"net/url"

	"manGo/internal/middleware" // <-- Add this import

	"manGo/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB) {

	// Public endpoints
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "server running"})
	})

	// Authentication endpoints
	r.POST("/login", Login(db))
	r.POST("/register", Register(db))

	// Protected endpoints (require JWT)
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		// Services
		protected.POST("/services", createService(db))
		protected.GET("/services", listServices(db))
		protected.GET("/services/:id/checks", getServiceChecks(db))

		// Service authentication management
		protected.POST("/services/:id/auth", manageServiceAuth(db))
		protected.GET("/services/:id/auth", getServiceAuth(db))
		protected.DELETE("/services/:id/auth", deleteServiceAuth(db))

		protected.PUT("/services/:id", updateService(db))
		protected.DELETE("/services/:id", deleteService(db))
		protected.GET("/services/:id/stats", getServiceStats(db))
	}
}

// createService handles POST /services
func createService(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var service models.Service

		// Get userID from context (set by AuthMiddleware)
		userIDRaw, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userID := userIDRaw.(uint)

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

		// Assign owner
		service.OwnerID = userID

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
	}
}

// listServices handles GET /services with filter
func listServices(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var services []models.Service

		// Get user info
		userIDRaw, _ := c.Get("userID")
		roleRaw, _ := c.Get("role")

		userID := userIDRaw.(uint)
		role := roleRaw.(string)

		query := db

		// Role filtering
		if role != "admin" {
			query = query.Where("owner_id = ?", userID)
		}

		// Get status filter from query param
		statusFilter := c.Query("status")

		// Load services
		if err := query.Find(&services).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch services"})
			return
		}

		// If no filter → return all
		if statusFilter == "" {
			c.JSON(http.StatusOK, gin.H{
				"count":    len(services),
				"services": services,
			})
			return
		}

		// Filter services by last check status
		var filtered []models.Service

		for _, s := range services {
			var lastCheck models.Check

			err := db.Where("service_id = ?", s.ID).
				Order("created_at DESC").
				First(&lastCheck).Error

			if err != nil {
				continue
			}

			if lastCheck.Status == statusFilter {
				filtered = append(filtered, s)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"count":    len(filtered),
			"services": filtered,
		})
	}
}

// getServiceChecks handles GET /services/:id/checks
func getServiceChecks(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceID := c.Param("id")

		// Check if user has access to this service
		if !canAccessService(c, db, serviceID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "you do not have access to this service"})
			return
		}

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

		c.JSON(http.StatusOK, gin.H{
			"count":  len(checks),
			"checks": checks,
		})
	}
}

// manageServiceAuth handles POST /services/:id/auth (create or update auth)
func manageServiceAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceID := c.Param("id")

		// Check ownership or admin
		if !canManageService(c, db, serviceID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "you do not have permission to manage auth for this service"})
			return
		}

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
	}
}

// getServiceAuth handles GET /services/:id/auth
func getServiceAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceID := c.Param("id")

		// Check access
		if !canAccessService(c, db, serviceID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "you do not have access to this service"})
			return
		}

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
	}
}

// deleteServiceAuth handles DELETE /services/:id/auth
func deleteServiceAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceID := c.Param("id")

		// Check permission
		if !canManageService(c, db, serviceID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "you do not have permission to manage auth for this service"})
			return
		}

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
	}
}

// Helper: check if user can view service details (owner or admin)
func canAccessService(c *gin.Context, db *gorm.DB, serviceID string) bool {
	userIDRaw, _ := c.Get("userID")
	roleRaw, _ := c.Get("role")
	userID := userIDRaw.(uint)
	role := roleRaw.(string)

	if role == "admin" {
		return true
	}

	var service models.Service
	if err := db.First(&service, serviceID).Error; err != nil {
		return false
	}
	return service.OwnerID == userID
}

// Helper: check if user can manage auth for service (owner or admin)
func canManageService(c *gin.Context, db *gorm.DB, serviceID string) bool {
	userIDRaw, _ := c.Get("userID")
	roleRaw, _ := c.Get("role")
	userID := userIDRaw.(uint)
	role := roleRaw.(string)

	if role == "admin" {
		return true
	}

	var service models.Service
	if err := db.First(&service, serviceID).Error; err != nil {
		return false
	}
	return service.OwnerID == userID
}

// updateService handles PUT /services/:id
func updateService(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceID := c.Param("id")

		// Check permission
		if !canManageService(c, db, serviceID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "no permission"})
			return
		}

		var req struct {
			URL string `json:"url" binding:"required"`
		}

		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
			return
		}

		if _, err := url.Parse(req.URL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid URL"})
			return
		}

		var service models.Service
		if err := db.First(&service, serviceID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
			return
		}

		service.URL = req.URL

		if err := db.Save(&service).Error; err != nil {
			log.Printf("handlers: failed to update service: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "service updated",
			"service": service,
		})
	}
}

// deleteService handles DELETE /services/:id
func deleteService(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceID := c.Param("id")

		// Check permission
		if !canManageService(c, db, serviceID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "no permission"})
			return
		}

		var service models.Service
		if err := db.First(&service, serviceID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
			return
		}

		// delete related data
		db.Where("service_id = ?", service.ID).Delete(&models.ServiceAuth{})
		db.Where("service_id = ?", service.ID).Delete(&models.Check{})

		if err := db.Delete(&service).Error; err != nil {
			log.Printf("handlers: failed to delete service: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "service deleted"})
	}
}

func getServiceStats(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceID := c.Param("id")

		// Check access
		if !canAccessService(c, db, serviceID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "no access"})
			return
		}

		var checks []models.Check
		if err := db.Where("service_id = ?", serviceID).Find(&checks).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch checks"})
			return
		}

		if len(checks) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"uptime":            0,
				"avg_response_time": 0,
				"total_checks":      0,
			})
			return
		}

		var upCount int
		var totalTime int64

		for _, check := range checks {
			if check.Status == "UP" {
				upCount++
			}
			totalTime += check.ResponseTime
		}

		uptime := float64(upCount) / float64(len(checks)) * 100
		avgTime := totalTime / int64(len(checks))

		c.JSON(http.StatusOK, gin.H{
			"uptime_percent":    uptime,
			"avg_response_time": avgTime,
			"total_checks":      len(checks),
		})
	}
}
