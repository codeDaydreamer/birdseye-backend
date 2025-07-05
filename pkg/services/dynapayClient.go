package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type dynapayClient struct {
	ClientID     string
	ClientSecret string
	APIBaseURL   string

	token     string
	tokenExp  time.Time
	tokenLock sync.Mutex
}

type tokenResponse struct {
	Token string `json:"token"`
}

type stkPushRequest struct {
	PhoneNumber string  `json:"phone_number"`
	Amount      float64 `json:"amount"`
	Reference   string  `json:"reference"`
	CallbackURL string  `json:"callback_url"`
}

type stkPushResponse struct {
	Message           string `json:"message"`
	CheckoutRequestID string `json:"checkout_id"`
	MerchantRequestID string `json:"merchant_id"`
	PaymentID         string `json:"payment_id"`
	Error             string `json:"error,omitempty"`
}

var Dynapay *dynapayClient

func InitDynapayClient() {
	Dynapay = &dynapayClient{
		ClientID:     os.Getenv("DYNAPAY_CLIENT_ID"),
		ClientSecret: os.Getenv("DYNAPAY_CLIENT_SECRET"),
		APIBaseURL:   os.Getenv("DYNAPAY_API_URL"),
	}
	log.Println("Dynapay client initialized")
}

// fetchToken retrieves a new JWT token from Dynapay
func (d *dynapayClient) fetchToken() error {
	d.tokenLock.Lock()
	defer d.tokenLock.Unlock()

	if time.Now().Before(d.tokenExp) {
		return nil // token still valid
	}

	payload := map[string]string{
		"client_id":     d.ClientID,
		"client_secret": d.ClientSecret,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(d.APIBaseURL+"/api/token", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch token: status %d", resp.StatusCode)
	}

	var result tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	d.token = result.Token
	d.tokenExp = time.Now().Add(4 * time.Minute) // safe buffer

	// Log the token once (for copying into env if needed)
	log.Println("🔐 Dynapay token fetched:", d.token)
	return nil
}

// SendSTKPush sends an STK push request to Dynapay
func (d *dynapayClient) SendSTKPush(phone string, amount float64, reference, webhook string) (*stkPushResponse, error) {
	// Ensure valid token
	if err := d.fetchToken(); err != nil {
		return nil, err
	}

	// Try the request once
	result, err := d.makeSTKRequest(phone, amount, reference, webhook)
	if err != nil && errors.Is(err, ErrUnauthorized) {
		// Token possibly expired, force refresh
		log.Println("⚠️ Token expired. Refreshing token and retrying...")
		d.tokenExp = time.Time{} // force token refresh

		if err := d.fetchToken(); err != nil {
			return nil, err
		}

		// Retry request once
		return d.makeSTKRequest(phone, amount, reference, webhook)
	}
	return result, err
}


var ErrUnauthorized = errors.New("unauthorized")

func (d *dynapayClient) makeSTKRequest(phone string, amount float64, reference, webhook string) (*stkPushResponse, error) {
	payload := stkPushRequest{
		PhoneNumber: phone,
		Amount:      amount,
		Reference:   reference,
		CallbackURL: webhook,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", d.APIBaseURL+"/api/payments/stk-push", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+d.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}

	var result stkPushResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Error != "" {
		return &result, errors.New(result.Error)
	}

	return &result, nil
}