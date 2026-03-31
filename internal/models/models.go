package models

import "time"

type Service struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    URL       string    `json:"url"`
    OwnerID  uint      `json:"owner_id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type Check struct {
    ID           uint      `gorm:"primaryKey" json:"id"`
    ServiceID    uint      `json:"service_id"`
    Status       string    `json:"status"`
    ResponseTime int64     `json:"response_time"`
    CreatedAt    time.Time `json:"created_at"`
}

// ServiceAuth represents authentication credentials for protected services
type ServiceAuth struct {
    ID           uint   `gorm:"primaryKey" json:"id"`
    ServiceID    uint   `gorm:"uniqueIndex" json:"service_id"`
    LoginURL     string `json:"login_url"`
    Username     string `json:"-"` // Hidden in JSON for security
    Password     string `json:"-"` // Hidden in JSON for security
    UsernameKey  string `json:"username_key"`
    PasswordKey  string `json:"password_key"`
    MonitorURL   string `json:"monitor_url"` // URL to monitor after login
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}