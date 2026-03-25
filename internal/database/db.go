package database

import (
	"log"

	"manGo/internal/config"
	"manGo/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(cfg *config.Database) *gorm.DB {
    db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
    if err != nil {
        log.Fatalf("failed to connect to database: %v", err)
    }

    log.Println("connected to database successfully")

    // create tables
    if err := db.AutoMigrate(&models.Service{}, &models.Check{}, &models.ServiceAuth{}, &models.User{}); err != nil {
        log.Fatalf("failed to run migrations: %v", err)
    }

    return db
}