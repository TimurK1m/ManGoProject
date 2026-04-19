package worker

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sync"
	"time"

	"manGo/internal/models"

	"gorm.io/gorm"
)

const (
	checkTimeout = 15 * time.Second
)

var alertLogger *log.Logger

func init() {
	file, err := os.OpenFile("alerts.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Printf("worker: failed to open alerts.log: %v", err)
		return
	}
	alertLogger = log.New(file, "ALERT: ", log.Ldate|log.Ltime)
}

func Start(db *gorm.DB) {
	ctx := context.Background()
	StartWithContext(ctx, db)
}

func StartWithContext(ctx context.Context, db *gorm.DB) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("worker: started")

	for {
		select {
		case <-ctx.Done():
			log.Println("worker: stopping gracefully")
			return
		case <-ticker.C:
			var services []models.Service
			if err := db.Find(&services).Error; err != nil {
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
	var wg sync.WaitGroup

	for _, s := range services {
		wg.Add(1)
		go func(service models.Service) {
			defer wg.Done()
			checkService(db, service)
		}(s)
	}

	wg.Wait()
}

func checkService(db *gorm.DB, s models.Service) {
	// Validate URL
	if _, err := url.Parse(s.URL); err != nil {
		log.Printf("worker: invalid URL %q: %v", s.URL, err)
		return
	}

	// Check if this service requires authentication
	var auth *models.ServiceAuth
	if err := db.Where("service_id = ?", s.ID).First(&auth).Error; err == nil {
		// Authentication exists, use authenticated client
		checkServiceWithAuth(db, s, auth)
		return
	}

	// No authentication needed, use regular client
	checkServiceWithoutAuth(db, s)
}

func checkServiceWithoutAuth(db *gorm.DB, s models.Service) {
	jar, _ := cookiejar.New(nil)

	// TLS configuration that skips certificate verification
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
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
		if err := recordCheck(db, s.ID, "DOWN", 0); err != nil {
			log.Printf("worker: failed to create check for service %d: %v", s.ID, err)
		}
		return
	}

	// headers to bypass basic anti-bot protection
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

	if err := recordCheck(db, s.ID, status, duration); err != nil {
		log.Printf("worker: failed to create check for service %d: %v", s.ID, err)
	}
}

func checkServiceWithAuth(db *gorm.DB, s models.Service, auth *models.ServiceAuth) {
	// Get authenticated client
	client, err := AuthenticatedClient(auth)
	if err != nil {
		log.Printf("worker: authentication failed for service %d: %v", s.ID, err)
		if err := recordCheck(db, s.ID, "DOWN", 0); err != nil {
			log.Printf("worker: failed to create check for service %d: %v", s.ID, err)
		}
		return
	}

	// Determine URL to check
	checkURL := auth.MonitorURL
	if checkURL == "" {
		checkURL = s.URL
	}

	req, err := http.NewRequest("GET", checkURL, nil)
	if err != nil {
		log.Printf("worker: failed to create request for %q: %v", checkURL, err)
		if err := recordCheck(db, s.ID, "DOWN", 0); err != nil {
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

	if err := recordCheck(db, s.ID, status, duration); err != nil {
		log.Printf("worker: failed to create check for service %d: %v", s.ID, err)
	}
}

func recordCheck(db *gorm.DB, serviceID uint, status string, duration int64) error {
	// Check previous state
	var lastCheck models.Check
	err := db.Where("service_id = ?", serviceID).Order("created_at desc").First(&lastCheck).Error
	if err == nil {
		if lastCheck.Status != status {
			msg := fmt.Sprintf("Service %d state changed: %s -> %s", serviceID, lastCheck.Status, status)
			if alertLogger != nil {
				alertLogger.Println(msg)
			} else {
				log.Println("ALERT:", msg)
			}
		}
	} else if err != gorm.ErrRecordNotFound {
		log.Printf("worker: failed to fetch last check: %v", err)
	}

	check := &models.Check{
		ServiceID:    serviceID,
		Status:       status,
		ResponseTime: duration,
	}
	return db.Create(check).Error
}
