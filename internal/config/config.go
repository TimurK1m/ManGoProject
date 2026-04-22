// config.go
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Database struct {
	Host     string
	User     string
	Password string
	Name     string
	Port     string
	SSLMode  string
}

type Server struct {
	Port string
}

type JWT struct {
	Secret string
}

type App struct {
	Database Database
	Server   Server
	JWT      JWT
}

func Load() *App {
	 _ = godotenv.Load()
	return &App{
		Database: Database{
			Host:     getEnv("DB_HOST", "localhost"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "112407"),
			Name:     getEnv("DB_NAME", "projectDB"),
			Port:     getEnv("DB_PORT", "5432"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Server: Server{
			Port: getEnv("SERVER_PORT", "8080"),
		},
		JWT: JWT{
			Secret: getEnv("JWT_SECRET", "super_secret_key"),
		},
	}
}

func (d *Database) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		d.Host, d.User, d.Password, d.Name, d.Port, d.SSLMode,
	)
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	
	return defaultValue
}
