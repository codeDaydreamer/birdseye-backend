package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

// Structs for Brevo email request
type EmailRequest struct {
	Sender      Sender      `json:"sender"`
	To          []Recipient `json:"to"`
	Subject     string      `json:"subject"`
	HtmlContent string      `json:"htmlContent"`
}

type Sender struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type Recipient struct {
	Email string `json:"email"`
}

// sendEmail sends an email via the Brevo API
func sendEmail(to, subject, htmlBody string) error {
	apiKey := os.Getenv("BREVO_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("brevo api  key is missing")
	}

	// Brevo API endpoint
	url := "https://api.brevo.com/v3/smtp/email"

	// Construct email request payload
	emailReq := EmailRequest{
		Sender: Sender{
			Email: "noreply@birdseye-poultry.com", // Replace with your sender email
			Name:  "Birdseye Poultry",
		},
		To: []Recipient{
			{Email: to},
		},
		Subject:    subject,
		HtmlContent: htmlBody,
	}

	// Marshal the email request into JSON
	data, err := json.Marshal(emailReq)
	if err != nil {
		return fmt.Errorf("failed to marshal email request: %w", err)
	}

	// Create the POST request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set necessary headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", apiKey)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body for additional information
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for successful response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to send email, status code: %d, response: %s", resp.StatusCode, string(respBody))
	}

	// Log the response body for further analysis
	fmt.Printf("Email sent successfully. Response: %s\n", string(respBody))

	return nil
}
