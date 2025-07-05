package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services"
	"birdseye-backend/pkg/services/email"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"time"
)

// PaymentHandler handles payment-related requests
type PaymentHandler struct{}

// RegisterPaymentRoutes sets up payment routes
func RegisterPaymentRoutes(r *gin.Engine) {
	handler := &PaymentHandler{}
	payments := r.Group("/payments").Use(middlewares.AuthMiddleware())
	{
		payments.POST("/initiate", handler.InitiatePayment)
		payments.GET("", handler.ListPayments)
	}
}

// PaymentRequest is the incoming payload from frontend
type PaymentRequest struct {
	Phone     string  `json:"PhoneNumber"`
	Amount    float64 `json:"amount"`
	Reference string  `json:"reference"`
}

// InitiatePayment sends payment request to Dynapay
func (h *PaymentHandler) InitiatePayment(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authorized"})
		return
	}

	var req PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	webhook := os.Getenv("DYNAPAY_WEBHOOK_URL")
	if webhook == "" {
		logrus.Error("Webhook URL not set in environment")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server misconfiguration"})
		return
	}

	result, err := services.Dynapay.SendSTKPush(req.Phone, req.Amount, req.Reference, webhook)
	if err != nil {
		logrus.Errorf("Dynapay STK Push failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to initiate payment", "details": err.Error()})
		return
	}

	logrus.Infof("STK Push successful for user %v, CheckoutID: %s", userID, result.CheckoutRequestID)

	c.JSON(http.StatusOK, gin.H{
		"message":             result.Message,
		"checkout_request_id": result.CheckoutRequestID,
		"merchant_request_id": result.MerchantRequestID,
		"payment_id":          result.PaymentID,
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

func HandleDynapayWebhook(c *gin.Context) {
	var payload DynapayCallbackPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		logrus.Errorf("❌ Invalid webhook payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	logrus.Infof("📩 Received payment webhook. Phone: %s | Status: %s", payload.PhoneNumber, payload.Status)

	// 🔍 Find user by phone number
	var user models.User
	if err := db.DB.Where("phone_number = ?", payload.PhoneNumber).First(&user).Error; err != nil {
		logrus.Warnf("⚠️ No user found with phone number %s", payload.PhoneNumber)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// 💾 Create new payment record
	payment := models.Payment{
		ID:                 payload.PaymentID,
		UserID:             user.ID,
		PhoneNumber:        payload.PhoneNumber,
		Amount:             payload.Amount,
		Status:             payload.Status,
		MpesaReference:     payload.MpesaReference,
		Reference:          payload.Reference,
		MerchantRequestID:  payload.MerchantRequestID,
		CheckoutRequestID:  payload.CheckoutRequestID,
		ResultDescription:  payload.ResultDescription,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := db.DB.Create(&payment).Error; err != nil {
		logrus.Errorf("❌ Failed to create payment record: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create payment"})
		return
	}

	// ✅ On success, update user status
	if payload.Status == "success" {
		user.Status = "active"
		user.UpdatedAt = time.Now()
		if err := db.DB.Save(&user).Error; err != nil {
			logrus.Errorf("❌ Failed to update user status: %v", err)
		} else {
			logrus.Infof("✅ User %d activated due to successful payment", user.ID)
			go func() {
				if err := email.SendPaymentSuccessEmail(user.Email, user.Username, payload.MpesaReference, payload.Amount); err != nil {
					logrus.Errorf("📧 Failed to send payment success email: %v", err)
				}
			}()
		}
	} else {
		// ❌ On failure, send failure email
		go func() {
			msg := payload.ResultDescription
			if msg == "" {
				msg = "Payment was not successful"
			}
			if err := email.SendPaymentFailureEmail(user.Email, user.Username, payload.Amount, msg); err != nil {
				logrus.Errorf("📧 Failed to send payment failure email: %v", err)
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{"message": "Webhook processed successfully"})
}

// ListPayments returns the authenticated user's payment history
func (h *PaymentHandler) ListPayments(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authorized"})
		return
	}

	var payments []models.Payment
	if err := db.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&payments).Error; err != nil {
		logrus.Errorf("❌ Failed to fetch payments: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load payments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"payments": payments})
}
