package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// SMTP2GoRequest defines the payload for sending email
type SMTP2GoRequest struct {
	Sender   string   `json:"sender"`
	To       []string `json:"to"`
	Subject  string   `json:"subject"`
	HtmlBody string   `json:"html_body,omitempty"`
	TextBody string   `json:"text_body,omitempty"`
}

// SendEmail sends an email using the SMTP2GO API
func SendEmail(to, subject, htmlBody string) error {
	apiKey := os.Getenv("SMTP2GO_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("SMTP2GO API key is missing")
	}

	url := "https://api.smtp2go.com/v3/email/send"

	emailReq := SMTP2GoRequest{
		Sender:   "birdseye-poultry@816-dynamics.com", 
		To:       []string{to},
		Subject:  subject,
		HtmlBody: htmlBody,
		TextBody: "Thank you for using Birdseye Poultry.",
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
	req.Header.Set("X-Smtp2go-Api-Key", apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("email failed: status %d - %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Email sent successfully. Response: %s\n", string(body))
	return nil
}
