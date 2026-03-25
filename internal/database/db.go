package database

import (
    "log"

    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "manGo/internal/config"
    "manGo/internal/models"
)

func Connect(cfg *config.Database) *gorm.DB {
    db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
    if err != nil {
        log.Fatalf("failed to connect to database: %v", err)
    }

    log.Println("connected to database successfully")

    // create tables
    if err := db.AutoMigrate(&models.Service{}, &models.Check{}, &models.ServiceAuth{}); err != nil {
        log.Fatalf("failed to run migrations: %v", err)
    }

    return db
}