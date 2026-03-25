package main

import (
	"database/sql"
	"log"

	"Project/internal/handler"
	"Project/internal/repository"
	"Project/internal/service"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	r := gin.Default()

	db, err := sql.Open("postgres", "user=postgres password=112407 dbname=projectDB sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	// 🔗 связываем слои
	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo)

	serviceRepo := repository.NewServiceRepository(db)
	serviceService := service.NewServiceService(serviceRepo)

	h := handler.NewHandler(authService, serviceService)

	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)

	r.POST("/services", h.CreateService)

	r.Run(":8080")
}