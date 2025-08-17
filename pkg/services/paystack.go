package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	

	"birdseye-backend/pkg/models"
	"gorm.io/gorm"
)

// PaystackService handles Paystack payment operations
type PaystackService struct {
	SecretKey string
	DB        *gorm.DB
}

func NewPaystackService(db *gorm.DB) *PaystackService {
	return &PaystackService{
		SecretKey: os.Getenv("PAYSTACK_SECRET_KEY"),
		DB:        db,
	}
}

// InitTransactionRequest is the payload for initiating a transaction
type InitTransactionRequest struct {
	Email     string `json:"email"`
	Amount    int    `json:"amount"`    // In kobo
	Reference string `json:"reference"` // Sent from frontend
	CallbackURL string `json:"callback_url"`
}

type InitTransactionResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		AuthorizationURL string `json:"authorization_url"`
		AccessCode       string `json:"access_code"`
		Reference        string `json:"reference"`
	} `json:"data"`
}

func (s *PaystackService) InitializeTransaction(req InitTransactionRequest) (*InitTransactionResponse, error) {
	if req.Email == "" || req.Amount <= 0 || req.Reference == "" {
		return nil, errors.New("email, amount, and reference are required")
	}

	// Prevent duplicate references
	var count int64
	if err := s.DB.Model(&models.Payment{}).Where("reference = ?", req.Reference).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("payment with reference '%s' already exists", req.Reference)
	}

	// Hardcode the callback URL here
	req.CallbackURL = "https://www.app.birdseye-poultry.com/paystack-callback"
	cancelURL := req.CallbackURL // use same for cancel

	// Create payload with metadata
	payload := map[string]interface{}{
		"email":        req.Email,
		"amount":       req.Amount,
		"reference":    req.Reference,
		"callback_url": req.CallbackURL,
		"metadata": map[string]string{
			"cancel_action": cancelURL,
		},
	}

	payloadBytes, _ := json.Marshal(payload)

	url := "https://api.paystack.co/transaction/initialize"
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+s.SecretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result InitTransactionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if !result.Status {
		return nil, fmt.Errorf("paystack error: %s", result.Message)
	}

	return &result, nil
}

// SaveInitiatedPayment creates a record of the initiated payment
func (s *PaystackService) SaveInitiatedPayment(userID uint, reference string, amount int) error {
	if reference == "" {
		return errors.New("reference is required")
	}

	// Prevent duplicate references
	var count int64
	s.DB.Model(&models.Payment{}).Where("reference = ?", reference).Count(&count)
	if count > 0 {
		return fmt.Errorf("payment with reference '%s' already exists", reference)
	}

	payment := models.Payment{
		UserID:    userID,
		Amount:    float64(amount) / 100,
		Gateway:   "paystack",
		TxRef:     &reference,
		Reference: reference,
		Status:    "initiated",
	
	}

	return s.DB.Create(&payment).Error
}

// VerifyPaystackSignature checks webhook authenticity
func (s *PaystackService) VerifyPaystackSignature(body []byte, signature string) bool {
	mac := hmac.New(sha512.New, []byte(s.SecretKey))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)
	expectedSignature := hex.EncodeToString(expectedMAC)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// VerifyTransaction checks payment status via Paystack API
func (s *PaystackService) VerifyTransaction(reference string) (map[string]interface{}, error) {
	if reference == "" {
		return nil, errors.New("reference is required")
	}

	url := "https://api.paystack.co/transaction/verify/" + reference
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.SecretKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	status, ok := result["status"].(bool)
	if !ok || !status {
		msg := "verification failed"
		if m, ok := result["message"].(string); ok {
			msg = m
		}
		return nil, errors.New("Paystack verification failed: " + msg)
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid response format from Paystack")
	}

	return data, nil
}

// SaveSuccessfulPayment stores the verified payment & updates the user
func (s *PaystackService) SaveSuccessfulPayment(userID uint, txData map[string]interface{}) error {
	// Extract reference
	ref, ok := txData["reference"].(string)
	if !ok || ref == "" {
		return errors.New("invalid reference in Paystack data")
	}

	// Extract amount
	amountFloat, ok := txData["amount"].(float64)
	if !ok {
		return errors.New("invalid amount format")
	}
	amount := float64(int(amountFloat) / 100)

	// Extract transaction ID
	idVal := txData["id"]
	var idStr string
	switch v := idVal.(type) {
	case float64:
		idStr = fmt.Sprintf("%.0f", v)
	case string:
		idStr = v
	default:
		return errors.New("invalid transaction ID format")
	}

	// Try to find existing payment by reference
	var existing models.Payment
	err := s.DB.Where("reference = ?", ref).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Record doesn't exist — insert new
			payment := models.Payment{
				UserID:    userID,
				Amount:    amount,
				Gateway:   "paystack",
				TxRef:     &ref,
				Reference: ref,
				Status:    "success",
				PaymentID: &idStr,
			}

			if err := s.DB.Create(&payment).Error; err != nil {
				return err
			}
		} else {
			// DB error
			return err
		}
	} else {
		// Record exists — update it
		existing.Status = "success"
		existing.PaymentID = &idStr
		existing.Amount = amount

		if err := s.DB.Save(&existing).Error; err != nil {
			return err
		}
	}

	// Update user payment status
	return s.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"is_trial_active": false,
		"payment_status":  "paid",
	}).Error
}
