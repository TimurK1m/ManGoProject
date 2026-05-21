// worker.go
package worker

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sync"
	"time"

	"manGo/internal/config"
	"manGo/internal/models"

	"gorm.io/gorm"
)

const (
	checkTimeout = 15 * time.Second
)

var alertLogger *log.Logger
var workerConfig *config.App

func init() {
	
	file, err := os.OpenFile("alerts.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Printf("worker: failed to open alerts.log: %v", err)
		return
	}
	alertLogger = log.New(file, "ALERT: ", log.Ldate|log.Ltime)
}

func Start(db *gorm.DB, cfg *config.App) {
	ctx := context.Background()
	StartWithContext(ctx, db, cfg)
}

func StartWithContext(ctx context.Context, db *gorm.DB, cfg *config.App) {
	workerConfig = cfg
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Println("worker: started")

	for {
		select {
		case <-ctx.Done():
			log.Println("worker: stopping gracefully")
			return
		case <-ticker.C:
			log.Println("worker: checking services...")
			var services []models.Service
			err := db.Raw(`
				SELECT * FROM services s
				WHERE NOT EXISTS (
					SELECT 1 FROM checks c 
					WHERE c.service_id = s.id 
					AND c.created_at >= NOW() - (s.check_interval * interval '1 second')
				)
			`).Scan(&services).Error
			if err != nil {
				log.Printf("worker: failed to fetch services: %v", err)
				continue
			}

			if len(services) > 0 {
				checkServices(db, services)
			}
		}
	}
}

func checkServices(db *gorm.DB, services []models.Service) {
	sem := make(chan struct{}, 10) 
	var wg sync.WaitGroup

	for _, s := range services {
		wg.Add(1)
		sem <- struct{}{}

		go func(service models.Service) {
			defer wg.Done()
			defer func() { <-sem }()

			checkService(db, service)
		}(s)
	}

	wg.Wait()
}

func checkService(db *gorm.DB, s models.Service) {
	
	u, err := url.ParseRequestURI(s.URL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		log.Printf("worker: invalid URL %q: %v", s.URL, err)
		return
	}

	
	var auth models.ServiceAuth
	if err := db.Where("service_id = ?", s.ID).First(&auth).Error; err == nil {
		checkServiceWithAuth(db, s, &auth)
		return
	}

	
	

	
	checkServiceWithoutAuth(db, s)
}

func checkServiceWithoutAuth(db *gorm.DB, s models.Service) {
	jar, _ := cookiejar.New(nil)

	
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
	}

	client := &http.Client{
		Timeout: checkTimeout,
		Jar:     jar,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
	}

	req, err := http.NewRequest("GET", s.URL, nil)
	if err != nil {
		log.Printf("worker: failed to create request for %q: %v", s.URL, err)
		if err := recordCheck(db, s, "DOWN", 0); err != nil {
			log.Printf("worker: failed to create check for service %d: %v", s.ID, err)
		}
		return
	}

	
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")

	start := time.Now()
	resp, err := client.Do(req)
	
	duration := time.Since(start).Milliseconds()

	status := "UP"

	if err != nil {
		status = "DOWN"
		log.Printf("worker: request failed for %q: %v", s.URL, err)
	} else if resp != nil {
		if resp.StatusCode >= 400 {
			status = "DOWN"
		}
		if resp.Body != nil {
			resp.Body.Close()
		}
	}

	if err := recordCheck(db, s, status, duration); err != nil {
		log.Printf("worker: failed to create check for service %d: %v", s.ID, err)
	}
}

func checkServiceWithAuth(db *gorm.DB, s models.Service, auth *models.ServiceAuth) {
	
	client, err := AuthenticatedClient(auth)
	if err != nil {
		log.Printf("worker: authentication failed for service %d: %v", s.ID, err)
		if err := recordCheck(db, s, "DOWN", 0); err != nil {
			log.Printf("worker: failed to create check for service %d: %v", s.ID, err)
		}
		return
	}

	
	checkURL := auth.MonitorURL
	if checkURL == "" {
		checkURL = s.URL
	}

	req, err := http.NewRequest("GET", checkURL, nil)
	if err != nil {
		log.Printf("worker: failed to create request for %q: %v", checkURL, err)
		if err := recordCheck(db, s, "DOWN", 0); err != nil {
			log.Printf("worker: failed to create check for service %d: %v", s.ID, err)
		}
		return
	}

	setDefaultHeaders(req)

	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start).Milliseconds()

	status := "UP"

	if err != nil {
		status = "DOWN"
		log.Printf("worker: authenticated request failed for %q: %v", checkURL, err)
	} else if resp != nil {
		if resp.StatusCode >= 400 {
			status = "DOWN"
		}
		if resp.Body != nil {
			resp.Body.Close()
		}
	}

	if err := recordCheck(db, s, status, duration); err != nil {
		log.Printf("worker: failed to create check for service %d: %v", s.ID, err)
	}
}

func recordCheck(db *gorm.DB, s models.Service, status string, duration int64) error {
	
	var lastCheck models.Check
	err := db.Where("service_id = ?", s.ID).Order("created_at desc").First(&lastCheck).Error
	
	shouldAlert := false
	var oldStatus string

	if err == nil {
		if lastCheck.Status != status {
			shouldAlert = true
			oldStatus = lastCheck.Status
		}
	} else if err == gorm.ErrRecordNotFound {
		shouldAlert = true
		oldStatus = "PENDING"
	} else {
		log.Printf("worker: failed to fetch last check: %v", err)
	}

	if shouldAlert {
		msg := fmt.Sprintf("%s state changed: %s -> %s", s.URL, oldStatus, status)
		if alertLogger != nil {
			alertLogger.Println(msg)
		} else {
			log.Println("ALERT:", msg)
		}
		
		if workerConfig != nil && workerConfig.Telegram.BotToken != "" && workerConfig.Telegram.ChatID != "" {
			go sendTelegramNotification(workerConfig.Telegram.BotToken, workerConfig.Telegram.ChatID, msg)
		} else {
			log.Println("worker: telegram config is missing, alert not sent")
		}
	}

	check := &models.Check{
		ServiceID:    s.ID,
		Status:       status,
		ResponseTime: duration,
	}
	if err := db.Create(check).Error; err != nil {
		return err
	}

	// Оставляем только последние 1000 записей для данного сервиса
	db.Exec(`
		DELETE FROM checks
		WHERE service_id = ?
		AND id NOT IN (
			SELECT id FROM checks
			WHERE service_id = ?
			ORDER BY created_at DESC
			LIMIT 1000
		)
	`, s.ID, s.ID)

	return nil
}

func sendTelegramNotification(token, chatID, message string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	body, _ := json.Marshal(map[string]string{
		"chat_id": chatID,
		"text":    "🚨 " + message,
	})
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("worker: failed to send telegram alert: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("worker: telegram API returned status %d", resp.StatusCode)
	}
}
