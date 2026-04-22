// service.go
package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"manGo/internal/config"
	"manGo/internal/middleware"

	"manGo/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ServiceWithStatus struct {
	models.Service
	LastStatus sql.NullString `json:"last_status"`
}

func RegisterRoutes(r *gin.Engine, db *gorm.DB, cfg *config.App){

	
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "server running"})
	})

	
	r.POST("/login", Login(db, cfg))
	r.POST("/register", Register(db))

	
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	{
		
		protected.POST("/services", createService(db))
		protected.GET("/services", listServices(db))
		protected.GET("/services/:id/checks", getServiceChecks(db))

		
		protected.POST("/services/:id/auth", manageServiceAuth(db))
		protected.GET("/services/:id/auth", getServiceAuth(db))
		protected.DELETE("/services/:id/auth", deleteServiceAuth(db))

		protected.PUT("/services/:id", updateService(db))
		protected.DELETE("/services/:id", deleteService(db))
		protected.GET("/services/:id/stats", getServiceStats(db))
	}
}


func createService(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var service models.Service

		
		userIDRaw, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userID, ok := userIDRaw.(uint)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			return
		}

		if err := c.BindJSON(&service); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON input"})
			return
		}

		
		if service.URL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "URL is required"})
			return
		}
		if _, err := url.Parse(service.URL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid URL format"})
			return
		}

		
		service.OwnerID = userID

		
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


func listServices(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		userIDRaw, _ := c.Get("userID")
		roleRaw, _ := c.Get("role")

		userID, ok := userIDRaw.(uint)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			return
		}

		role, ok := roleRaw.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid role"})
			return
		}

		statusFilter := c.Query("status")

		var services []ServiceWithStatus

		baseQuery := `
			SELECT s.*, c.status as last_status
			FROM services s
			LEFT JOIN LATERAL (
				SELECT status
				FROM checks
				WHERE service_id = s.id
				ORDER BY created_at DESC
				LIMIT 1
			) c ON true
		`

		
		if role != "admin" {
			baseQuery += " WHERE s.owner_id = ?"
			if err := db.Raw(baseQuery, userID).Scan(&services).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch services"})
				return
			}
		} else {
			if err := db.Raw(baseQuery).Scan(&services).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch services"})
				return
			}
		}

		
		if statusFilter != "" {
			var filtered []ServiceWithStatus
			for _, s := range services {
				if s.LastStatus.Valid && s.LastStatus.String == statusFilter {
					filtered = append(filtered, s)
				}
			}
			services = filtered
		}

		c.JSON(http.StatusOK, gin.H{
			"count":    len(services),
			"services": services,
		})
	}
}

func getServiceChecks(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		
		
		serviceID, ok := parseID(c)
		if !ok { return }




		
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


func manageServiceAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceID := c.Param("id")

		
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

		
		var service models.Service
		if err := db.First(&service, serviceID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
			return
		}

		
		var existingAuth models.ServiceAuth
		if err := db.Where("service_id = ?", serviceID).First(&existingAuth).Error; err == nil {
			
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


func getServiceAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceID, ok := parseID(c)

		if !ok { return }

		
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


func deleteServiceAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceID := c.Param("id")

		
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


func canAccessService(c *gin.Context, db *gorm.DB, serviceID uint) bool {
	userIDRaw, _ := c.Get("userID")
	roleRaw, _ := c.Get("role")

	userID, ok := userIDRaw.(uint)
	if !ok {
		return false
	}

	role, ok := roleRaw.(string)
	if !ok {
		return false
	}

	if role == "admin" {
		return true
	}

	var service models.Service
	if err := db.First(&service, serviceID).Error; err != nil {
		return false
	}

	return service.OwnerID == userID
}

func canManageService(c *gin.Context, db *gorm.DB, serviceID string) bool {
	userIDRaw, _ := c.Get("userID")
	roleRaw, _ := c.Get("role")
	userID, ok := userIDRaw.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return false
	}
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


func updateService(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceID := c.Param("id")

		
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


func deleteService(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceID := c.Param("id")

		
		if !canManageService(c, db, serviceID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "no permission"})
			return
		}

		var service models.Service
		if err := db.First(&service, serviceID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
			return
		}

		
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
		serviceID, ok := parseID(c)
		if !ok {
			return
		}

		if !canAccessService(c, db, serviceID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "no access"})
			return
		}

		type Stats struct {
			Total int64
			Up    int64
			Avg   float64
		}

		var stats Stats

		err := db.Raw(`
			SELECT 
				COUNT(*) as total,
				COALESCE(SUM(CASE WHEN status = 'UP' THEN 1 ELSE 0 END), 0) as up,
				COALESCE(AVG(response_time), 0) as avg
			FROM checks
			WHERE service_id = ?
		`, serviceID).Scan(&stats).Error

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch stats"})
			return
		}

		if stats.Total == 0 {
			c.JSON(http.StatusOK, gin.H{
				"uptime_percent":    0,
				"avg_response_time": 0,
				"total_checks":      0,
			})
			return
		}

		uptime := float64(stats.Up) / float64(stats.Total) * 100

		c.JSON(http.StatusOK, gin.H{
			"uptime_percent":    uptime,
			"avg_response_time": stats.Avg,
			"total_checks":      stats.Total,
		})
	}
}

func parseID(c *gin.Context) (uint, bool) {
	idStr := c.Param("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}

	return uint(id), true
}
