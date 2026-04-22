// models.go
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


type ServiceAuth struct {
    ID           uint   `gorm:"primaryKey" json:"id"`
    ServiceID    uint   `gorm:"uniqueIndex" json:"service_id"`
    LoginURL     string `json:"login_url"`
    Username     string `json:"-"` 
    Password     string `json:"-"` 
    UsernameKey  string `json:"username_key"`
    PasswordKey  string `json:"password_key"`
    MonitorURL   string `json:"monitor_url"` 
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}