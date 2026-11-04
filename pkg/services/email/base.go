package email

import (
	"birdseye-backend/pkg/appconfig"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// defaultSender is used when EMAIL_FROM isn't set; birdseye-poultry.com must be verified in Resend
const defaultSender = "Birdseye Poultry <hello@birdseye-poultry.com>"

const appURL = "https://app.birdseye-poultry.com"

var brand = Brand{Product: "Birdseye Poultry", Color: "#249225", Dark: "#1a6b1b", Tint: "#ecf8ec", URL: appURL}

// resendRequest is the payload for Resend's send-email API
type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html,omitempty"`
	Text    string   `json:"text,omitempty"`
}

// send renders a message in the Birdseye layout and sends it
func send(to, subject string, m Message) error {
	htmlBody, textBody := Render(brand, m)
	return deliver(to, subject, htmlBody, textBody)
}

// SendEmail sends a short message whose body is trusted HTML, in the Birdseye layout
func SendEmail(to, subject, htmlBody string) error {
	return send(to, subject, Message{Title: subject, Lines: []string{htmlBody}})
}

// deliver sends through the Resend API (RESEND_API_KEY, optional EMAIL_FROM)
func deliver(to, subject, htmlBody, textBody string) error {
	apiKey := appconfig.Get("RESEND_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("the Resend API key isn't set (admin console > Keys)")
	}

	sender := appconfig.Get("EMAIL_FROM")
	if sender == "" {
		sender = defaultSender
	}

	url := os.Getenv("RESEND_API_URL") // overridable for local testing
	if url == "" {
		url = "https://api.resend.com/emails"
	}

	data, err := json.Marshal(resendRequest{From: sender, To: []string{to}, Subject: subject, HTML: htmlBody, Text: textBody})
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2000))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("email failed: status %d - %s", resp.StatusCode, string(body))
	}
	return nil
}
