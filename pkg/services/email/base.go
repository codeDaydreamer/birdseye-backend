package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// defaultSender is used when EMAIL_FROM isn't set; its domain must be verified in Resend
const defaultSender = "Birdseye Poultry <birdseye-poultry@816-dynamics.com>"

// resendRequest is the payload for Resend's send-email API
type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html,omitempty"`
	Text    string   `json:"text,omitempty"`
}

// SendEmail sends an email using the Resend API (RESEND_API_KEY, optional EMAIL_FROM)
func SendEmail(to, subject, htmlBody string) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY is missing")
	}

	sender := os.Getenv("EMAIL_FROM")
	if sender == "" {
		sender = defaultSender
	}

	url := "https://api.resend.com/emails"

	emailReq := resendRequest{
		From:    sender,
		To:      []string{to},
		Subject: subject,
		HTML:    htmlBody,
		Text:    "Thank you for using Birdseye Poultry.",
	}

	data, err := json.Marshal(emailReq)
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

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("email failed: status %d - %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Email sent successfully. Response: %s\n", string(body))
	return nil
}
