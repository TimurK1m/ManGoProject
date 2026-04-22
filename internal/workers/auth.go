// auth.go
package worker

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"manGo/internal/models"
)


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

	
	if err := login(client, auth); err != nil {
		log.Printf("worker: login failed for service %d: %v", auth.ServiceID, err)
		return nil, err
	}

	log.Printf("worker: logged in successfully for service %d", auth.ServiceID)
	return client, nil
}


func login(client *http.Client, auth *models.ServiceAuth) error {
	
	getReq, err := http.NewRequest("GET", auth.LoginURL, nil)
	if err != nil {
		return err
	}
	setDefaultHeaders(getReq)

	resp, err := client.Do(getReq)
	if err != nil {
		return err
	}
	if resp.Body != nil {
		resp.Body.Close()
	}

	
	form := url.Values{}
	form.Set(auth.UsernameKey, auth.Username)
	form.Set(auth.PasswordKey, auth.Password)

	
	postReq, err := http.NewRequest("POST", auth.LoginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}

	setDefaultHeaders(postReq)
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = client.Do(postReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("login failed: status %d", resp.StatusCode)
	}

	return nil
}


func setDefaultHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
}
