package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services"
	"birdseye-backend/pkg/services/email"
	"crypto/subtle"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// PaymentHandler handles payment-related requests
type PaymentHandler struct{}

// RegisterPaymentRoutes sets up payment routes: only owners and managers handle the farm's payments
func RegisterPaymentRoutes(r *gin.Engine) {
	handler := &PaymentHandler{}
	payments := r.Group("/payments").Use(
		middlewares.FarmAuth(),
		middlewares.RequireFarmRole(models.FarmRoleOwner, models.FarmRoleManager),
	)
	{
		payments.POST("/initiate", handler.InitiatePayment)
		payments.GET("", handler.ListPayments)
	}
}

// PaymentRequest is the incoming payload from frontend. Any amount it sends is ignored: the price
// comes from services.SubscriptionPrice.
type PaymentRequest struct {
	Phone     string `json:"PhoneNumber"`
	Reference string `json:"reference"`
}

// dynapayWebhookURL is where Dynapay posts the payment result. Dynapay doesn't sign webhooks, so the
// URL carries a shared secret that HandleDynapayWebhook checks.
func dynapayWebhookURL() (string, error) {
	base := os.Getenv("DYNAPAY_WEBHOOK_URL")
	secret := os.Getenv("DYNAPAY_WEBHOOK_SECRET")
	if base == "" || secret == "" {
		return "", fmt.Errorf("DYNAPAY_WEBHOOK_URL and DYNAPAY_WEBHOOK_SECRET must be set")
	}
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + "token=" + url.QueryEscape(secret), nil
}

// InitiatePayment sends an M-Pesa STK push for the farm's subscription via Dynapay
func (h *PaymentHandler) InitiatePayment(c *gin.Context) {
	farmID, userID := farmContext(c)

	var req PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Phone) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}
	if req.Reference == "" {
		req.Reference = "BE-" + uuid.NewString()[:8]
	}
	if len(req.Reference) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reference is too long"})
		return
	}

	var existing int64
	db.DB.Model(&models.Payment{}).Where("reference = ?", req.Reference).Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "A payment with this reference already exists"})
		return
	}

	amount, err := services.SubscriptionPrice(farmID)
	if err != nil {
		logrus.Errorf("Failed to work out subscription price for farm %d: %v", farmID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not determine price"})
		return
	}

	webhook, err := dynapayWebhookURL()
	if err != nil {
		logrus.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server misconfiguration"})
		return
	}

	result, err := services.Dynapay.SendSTKPush(req.Phone, amount, req.Reference, webhook)
	if err != nil {
		logrus.Errorf("Dynapay STK Push failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to initiate payment", "details": err.Error()})
		return
	}

	// Record the payment now, so the webhook can only complete payments we started
	payment := models.Payment{
		UserID:            userID,
		FarmID:            farmID,
		Amount:            amount,
		PhoneNumber:       req.Phone,
		Status:            "initiated",
		Reference:         req.Reference,
		PaymentID:         &result.PaymentID,
		MerchantRequestID: result.MerchantRequestID,
		CheckoutRequestID: result.CheckoutRequestID,
		Gateway:           "mpesa",
	}
	if err := db.DB.Create(&payment).Error; err != nil {
		logrus.Errorf("Failed to record initiated payment for farm %d: %v", farmID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record payment"})
		return
	}

	logrus.Infof("STK Push sent for farm %d by user %d, CheckoutID: %s", farmID, userID, result.CheckoutRequestID)

	c.JSON(http.StatusOK, gin.H{
		"message":             result.Message,
		"checkout_request_id": result.CheckoutRequestID,
		"merchant_request_id": result.MerchantRequestID,
		"reference":           req.Reference,
		"amount":              amount,
	})
}

// DynapayCallbackPayload matches the webhook payload
type DynapayCallbackPayload struct {
	PaymentID          string  `json:"id"`
	TenantID           string  `json:"tenant_id"`
	Status             string  `json:"status"`
	Amount             float64 `json:"amount"`
	PhoneNumber        string  `json:"phone_number"`
	MpesaReference     string  `json:"mpesa_reference"`
	Reference          string  `json:"reference"`
	MerchantRequestID  string  `json:"merchant_request_id"`
	CheckoutRequestID  string  `json:"checkout_request_id"`
	ResultDescription  string  `json:"result_description"`
	CreatedAt          string  `json:"created_at"`
	CallbackReceivedAt string  `json:"callback_received_at"`
	WebhookURL         string  `json:"webhook_url"`
}

func RegisterWebhookRoutes(r *gin.Engine) {
	r.POST("/webhooks/dynapay-payment", HandleDynapayWebhook)
}

// HandleDynapayWebhook completes a payment we initiated with the result Dynapay reports
func HandleDynapayWebhook(c *gin.Context) {
	secret := os.Getenv("DYNAPAY_WEBHOOK_SECRET")
	token := c.Query("token")
	if secret == "" || subtle.ConstantTimeCompare([]byte(token), []byte(secret)) != 1 {
		logrus.Warn("⚠️ Rejected Dynapay webhook with a missing or wrong token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var payload DynapayCallbackPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		logrus.Errorf("❌ Invalid webhook payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	logrus.Infof("📩 Received payment webhook. Payment: %s | Status: %s", payload.PaymentID, payload.Status)

	var payment models.Payment
	if err := db.DB.Where("payment_id = ?", payload.PaymentID).First(&payment).Error; err != nil {
		logrus.Warnf("⚠️ Webhook for unknown payment %s", payload.PaymentID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	// Dynapay retries webhooks; a payment that already succeeded stays that way
	if payment.Status == "success" {
		c.JSON(http.StatusOK, gin.H{"message": "Webhook already processed"})
		return
	}

	now := models.CustomTime{Time: time.Now().UTC()}
	payment.Status = payload.Status
	payment.MpesaReference = payload.MpesaReference
	payment.ResultDescription = payload.ResultDescription
	payment.CallbackReceivedAt = &now
	if payload.Status == "success" {
		payment.PaidAt = now
		if payload.Amount > 0 && payload.Amount < payment.Amount {
			logrus.Warnf("⚠️ Payment %s: paid %.2f but %.2f was due", payment.ID, payload.Amount, payment.Amount)
			payment.Status = "underpaid"
		}
	}

	if err := db.DB.Save(&payment).Error; err != nil {
		logrus.Errorf("❌ Failed to update payment %s: %v", payment.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update payment"})
		return
	}

	var payer models.User
	db.DB.First(&payer, payment.UserID)

	if payment.Status == "success" {
		if _, err := services.GetFarmBillingStatus(payment.FarmID); err != nil {
			logrus.Errorf("❌ Failed to refresh billing status for farm %d: %v", payment.FarmID, err)
		}
		logrus.Infof("✅ Farm %d paid via payment %s", payment.FarmID, payment.ID)

		managers, _ := services.GetFarmManagers(payment.FarmID)
		for _, user := range managers {
			go func(u models.User) {
				if err := email.SendPaymentSuccessEmail(u.Email, u.Username, payload.MpesaReference, payment.Amount); err != nil {
					logrus.Errorf("📧 Failed to send payment success email: %v", err)
				}
			}(user)
		}
	} else if payer.ID != 0 {
		go func() {
			msg := payload.ResultDescription
			if msg == "" {
				msg = "Payment was not successful"
			}
			if err := email.SendPaymentFailureEmail(payer.Email, payer.Username, payment.Amount, msg); err != nil {
				logrus.Errorf("📧 Failed to send payment failure email: %v", err)
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{"message": "Webhook processed successfully"})
}

// ListPayments returns the farm's payment history
func (h *PaymentHandler) ListPayments(c *gin.Context) {
	farmID, _ := farmContext(c)

	var payments []models.Payment
	if err := db.DB.Where("farm_id = ?", farmID).Order("created_at DESC").Find(&payments).Error; err != nil {
		logrus.Errorf("❌ Failed to fetch payments: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load payments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"payments": payments})
}
