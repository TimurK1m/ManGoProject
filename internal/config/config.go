package config

import (
	"fmt"
	"os"
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

type App struct {
	Database Database
	Server   Server
}

func Load() *App {
	return &App{
		Database: Database{
			Host:     getEnv("DB_HOST", "localhost"),
			User:     getEnv("DB_USER", "appuser"),
			Password: getEnv("DB_PASSWORD", "appuser"),
			Name:     getEnv("DB_NAME", "mango"),
			Port:     getEnv("DB_PORT", "5432"),
			SSLMode:  getEnv("DB_SSLMODE", "require"),
		},
		Server: Server{
			Port: getEnv("SERVER_PORT", "8080"),
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
