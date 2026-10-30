package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
)

// dynapayClient talks to Dynapay's tenant API (/v1). Dynapay charges the farm's M-Pesa number on
// Birdseye's own paybill/till and reports the outcome via a signed webhook (see VerifyDynapaySignature).
type dynapayClient struct {
	ClientID     string
	ClientSecret string
	APIBaseURL   string

	http      *http.Client
	token     string
	tokenExp  time.Time
	tokenLock sync.Mutex
}

// DynapayPayment is a payment as Dynapay reports it
type DynapayPayment struct {
	ID            string `json:"id"`
	Status        string `json:"status"` // pending | succeeded | failed | cancelled | expired
	Amount        int64  `json:"amount"`
	PaidAmount    int64  `json:"paid_amount"`
	PhoneNumber   string `json:"phone_number"`
	Reference     string `json:"reference"`
	Receipt       string `json:"receipt"`
	FailureReason string `json:"failure_reason"`

	ProviderRequestID  string `json:"provider_request_id"`
	ProviderMerchantID string `json:"provider_merchant_id"`
}

var Dynapay *dynapayClient

func InitDynapayClient() {
	Dynapay = &dynapayClient{
		ClientID:     os.Getenv("DYNAPAY_CLIENT_ID"),
		ClientSecret: os.Getenv("DYNAPAY_CLIENT_SECRET"),
		APIBaseURL:   strings.TrimRight(os.Getenv("DYNAPAY_API_URL"), "/"),
		http:         &http.Client{Timeout: 30 * time.Second},
	}
	log.Println("Dynapay client initialized")
}

// accessToken returns a cached API token, fetching a new one shortly before it expires
func (d *dynapayClient) accessToken(force bool) (string, error) {
	d.tokenLock.Lock()
	defer d.tokenLock.Unlock()
	if !force && d.token != "" && time.Now().Before(d.tokenExp) {
		return d.token, nil
	}

	body, _ := json.Marshal(map[string]string{"client_id": d.ClientID, "client_secret": d.ClientSecret})
	resp, err := d.http.Post(d.APIBaseURL+"/v1/token", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("dynapay token request failed: status %d", resp.StatusCode)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.AccessToken == "" {
		return "", errors.New("dynapay returned no access token")
	}
	d.token = result.AccessToken
	d.tokenExp = time.Now().Add(time.Duration(result.ExpiresIn)*time.Second - time.Minute)
	return d.token, nil
}

// do sends an authenticated request, refreshing the token once if Dynapay rejects it
func (d *dynapayClient) do(method, path string, payload interface{}, headers map[string]string, out interface{}) (int, error) {
	var body []byte
	if payload != nil {
		body, _ = json.Marshal(payload)
	}
	for attempt := 0; attempt < 2; attempt++ {
		token, err := d.accessToken(attempt > 0)
		if err != nil {
			return 0, err
		}
		req, err := http.NewRequest(method, d.APIBaseURL+path, bytes.NewReader(body))
		if err != nil {
			return 0, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := d.http.Do(req)
		if err != nil {
			return 0, err
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized && attempt == 0 {
			continue
		}
		if resp.StatusCode >= 300 {
			var apiErr struct {
				Error   string `json:"error"`
				Details string `json:"details"`
			}
			json.Unmarshal(raw, &apiErr)
			msg := apiErr.Error
			if apiErr.Details != "" {
				msg += ": " + apiErr.Details
			}
			return resp.StatusCode, fmt.Errorf("dynapay: %s (HTTP %d)", msg, resp.StatusCode)
		}
		return resp.StatusCode, json.Unmarshal(raw, out)
	}
	return http.StatusUnauthorized, errors.New("dynapay rejected the API token")
}

// CreatePayment asks Dynapay to charge a phone. The reference doubles as the idempotency key, so a
// retried request never charges twice.
func (d *dynapayClient) CreatePayment(phone string, amount float64, reference, description string, metadata map[string]interface{}) (*DynapayPayment, error) {
	var payment DynapayPayment
	_, err := d.do(http.MethodPost, "/v1/payments", map[string]interface{}{
		"amount":       int64(amount),
		"phone_number": phone,
		"reference":    reference,
		"description":  description,
		"metadata":     metadata,
	}, map[string]string{"Idempotency-Key": reference}, &payment)
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// GetPayment fetches a payment's current state from Dynapay
func (d *dynapayClient) GetPayment(id string) (*DynapayPayment, error) {
	var payment DynapayPayment
	if _, err := d.do(http.MethodGet, "/v1/payments/"+id, nil, nil, &payment); err != nil {
		return nil, err
	}
	return &payment, nil
}

// VerifyDynapaySignature checks a webhook's Dynapay-Signature header ("t=<unix>,v1=<hex hmac>") against
// DYNAPAY_WEBHOOK_SECRET, rejecting signatures more than 5 minutes old (replays)
func VerifyDynapaySignature(header string, body []byte) error {
	secret := os.Getenv("DYNAPAY_WEBHOOK_SECRET")
	if secret == "" {
		return errors.New("DYNAPAY_WEBHOOK_SECRET is not set")
	}
	var ts, sig string
	for _, part := range strings.Split(header, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			ts = kv[1]
		case "v1":
			sig = kv[1]
		}
	}
	unix, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || sig == "" {
		return errors.New("malformed signature header")
	}
	if age := time.Since(time.Unix(unix, 0)); age > 5*time.Minute || age < -5*time.Minute {
		return errors.New("signature timestamp too old")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "."))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return errors.New("signature mismatch")
	}
	return nil
}

// ApplyDynapayResult records Dynapay's final outcome on a Birdseye payment: success extends the farm's
// subscription, anything else marks it failed. Returns true if the payment changed (it only changes
// once, however many webhooks or checks report it).
func ApplyDynapayResult(payment *models.Payment, result *DynapayPayment) (bool, error) {
	var status string
	switch result.Status {
	case "succeeded":
		status = "success"
		if result.Amount != int64(payment.Amount) {
			status = "failed" // amount doesn't match what we asked for
		}
	case "failed", "cancelled", "expired":
		status = "failed"
	default:
		return false, nil // still pending
	}

	now := models.CustomTime{Time: time.Now().UTC()}
	updates := map[string]interface{}{
		"status":               status,
		"mpesa_reference":      result.Receipt,
		"result_description":   result.FailureReason,
		"callback_received_at": now,
	}
	if status == "success" {
		updates["paid_at"] = now
	}
	res := db.DB.Model(&models.Payment{}).Where("id = ? AND status = ?", payment.ID, "initiated").Updates(updates)
	if res.Error != nil || res.RowsAffected == 0 {
		return false, res.Error
	}
	if err := db.DB.First(payment, "id = ?", payment.ID).Error; err != nil {
		return true, err
	}
	if status == "success" {
		if err := ApplySuccessfulPayment(payment); err != nil {
			return true, err
		}
	}
	return true, nil
}

// RunDynapayReconciler asks Dynapay about M-Pesa payments still waiting on a result, in case a webhook
// never arrived. onSettled runs for each payment it settles (e.g. to send emails).
func RunDynapayReconciler(ctx context.Context, onSettled func(*models.Payment)) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		var waiting []models.Payment
		db.DB.Where("gateway = ? AND status = ? AND payment_id IS NOT NULL AND created_at < ? AND created_at > ?",
			"mpesa", "initiated", time.Now().Add(-2*time.Minute), time.Now().Add(-48*time.Hour)).
			Limit(50).Find(&waiting)
		for i := range waiting {
			result, err := Dynapay.GetPayment(*waiting[i].PaymentID)
			if err != nil {
				log.Printf("Dynapay check for payment %s failed: %v", waiting[i].ID, err)
				continue
			}
			changed, err := ApplyDynapayResult(&waiting[i], result)
			if err != nil {
				log.Printf("Applying Dynapay result to payment %s failed: %v", waiting[i].ID, err)
			}
			if changed && onSettled != nil {
				onSettled(&waiting[i])
			}
		}
	}
}
