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

// listServices handles GET /services
func listServices(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        var services []models.Service
        var result *gorm.DB

        // Get user info from context
        userIDRaw, _ := c.Get("userID")
        roleRaw, _ := c.Get("role")

        userID := userIDRaw.(uint)
        role := roleRaw.(string)

        if role == "admin" {
            // Admin sees all services
            result = db.Find(&services)
        } else {
            // Regular user sees only their own services
            result = db.Where("owner_id = ?", userID).Find(&services)
        }

        if result.Error != nil {
            log.Printf("handlers: failed to fetch services: %v", result.Error)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch services"})
            return
        }

        c.JSON(http.StatusOK, gin.H{
            "count":    len(services),
            "services": services,
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