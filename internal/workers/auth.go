package worker

import (
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"manGo/internal/models"
)

// AuthenticatedClient performs login and returns an authenticated HTTP client
func AuthenticatedClient(auth *models.ServiceAuth) (*http.Client, error) {
	jar, _ := cookiejar.New(nil)

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

	// Perform login
	if err := login(client, auth); err != nil {
		log.Printf("worker: login failed for service %d: %v", auth.ServiceID, err)
		return nil, err
	}

	log.Printf("worker: logged in successfully for service %d", auth.ServiceID)
	return client, nil
}

// login performs the login request
func login(client *http.Client, auth *models.ServiceAuth) error {
	// Prepare login credentials
	data := url.Values{}
	data.Set(auth.UsernameKey, auth.Username)
	data.Set(auth.PasswordKey, auth.Password)

	req, err := http.NewRequest("POST", auth.LoginURL, nil)
	if err != nil {
		return err
	}

	// Set headers
	setDefaultHeaders(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Create a new request with form data
	req2, err := http.NewRequest("POST", auth.LoginURL, nil)
	if err != nil {
		return err
	}

	// Set headers
	setDefaultHeaders(req2)
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.Body = io.NopCloser(nil)
	req2.ContentLength = 0

	// Try to get login page first to get any CSRF tokens
	getReq, _ := http.NewRequest("GET", auth.LoginURL, nil)
	setDefaultHeaders(getReq)
	resp, err := client.Do(getReq)
	if err != nil {
		return err
	}
	if resp.Body != nil {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		log.Printf("worker: login page received, size: %d bytes", len(body))
	}

	// Now perform POST with credentials
	postReq, _ := http.NewRequest("POST", auth.LoginURL, nil)
	setDefaultHeaders(postReq)
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Body = io.NopCloser(nil)
	postReq.ContentLength = 0

	resp, err = client.Do(postReq)
	if err != nil {
		return err
	}
	if resp.Body != nil {
		resp.Body.Close()
	}

	return nil
}

// setDefaultHeaders sets standard browser headers
func setDefaultHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
}
