// main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"manGo/internal/config"
	"manGo/internal/database"
	"manGo/internal/handlers"
	workers "manGo/internal/workers"

	"github.com/gin-gonic/gin"
)

func main() {
    
    cfg := config.Load()

    
    db := database.Connect(&cfg.Database)
    sqlDB, err := db.DB()
    if err != nil {
        log.Fatalf("failed to get database instance: %v", err)
    }
    defer sqlDB.Close()

    
    r := gin.Default()

    
    handlers.RegisterRoutes(r, db, cfg)

    
    workerCtx, cancelWorker := context.WithCancel(context.Background())
    var wg sync.WaitGroup

    wg.Add(1)
    go func() {
        defer wg.Done()
        workers.StartWithContext(workerCtx, db)
    }()

    log.Println("starting server on port", cfg.Server.Port)

    
    srv := &http.Server{
        Addr:    ":" + cfg.Server.Port,
        Handler: r,
    }

    
    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("server error: %v", err)
        }
    }()

    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    log.Println("shutting down gracefully...")

    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Printf("server shutdown error: %v", err)
    }

    
    cancelWorker()
    wg.Wait()

    log.Println("server stopped")
}