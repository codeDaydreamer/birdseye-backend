package api

import (
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PaystackHandler handles payment routes using Paystack
type PaystackHandler struct {
	Service *services.PaystackService
	DB      *gorm.DB
}

// SetupPaystackRoutes registers Paystack routes
func SetupPaystackRoutes(r *gin.Engine, db *gorm.DB) {
	handler := &PaystackHandler{
		Service: services.NewPaystackService(db),
		DB:      db,
	}

	paystackGroup := r.Group("/paystack").Use(middlewares.AuthMiddleware())
	{
		paystackGroup.POST("/initiate", handler.InitiateTransaction)
		paystackGroup.GET("/status/:reference", handler.GetPaymentStatusByReference) 
	}

	// Webhook should NOT be behind auth
	r.POST("/webhook/paystack", handler.HandlePaystackWebhook)
}

type paystackInitPayload struct {
	Email     string `json:"email"`
	Amount    int    `json:"amount"`    // in kobo
	Reference string `json:"reference"` // add this to receive from frontend
}


// InitiateTransaction handles Paystack transaction initialization
func (h *PaystackHandler) InitiateTransaction(c *gin.Context) {
	var payload paystackInitPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	tx, err := h.Service.InitializeTransaction(services.InitTransactionRequest{
		Email:     payload.Email,
		Amount:    payload.Amount,
		Reference: payload.Reference,  // pass reference from frontend
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.DB.Where("email = ?", payload.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Save initiated payment with the frontend reference, not the one from Paystack
	if err := h.Service.SaveInitiatedPayment(user.ID, payload.Reference, payload.Amount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save initiated payment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Transaction initialized",
		"data":    tx.Data,
	})
}

// HandlePaystackWebhook handles webhook callbacks from Paystack
func (h *PaystackHandler) HandlePaystackWebhook(c *gin.Context) {
	// Read body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Println("Webhook read error:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid body"})
		return
	}

	// Verify HMAC signature
	signature := c.GetHeader("x-paystack-signature")
	if !h.Service.VerifyPaystackSignature(body, signature) {
		log.Println("Invalid Paystack signature")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	// Parse JSON payload
	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		log.Println("Webhook JSON parse error:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	eventType, ok := event["event"].(string)
	if !ok {
		log.Println("Missing event type in webhook")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing event type"})
		return
	}

	// Only handle charge.success
	if eventType == "charge.success" {
		data, ok := event["data"].(map[string]interface{})
		if !ok {
			log.Println("Missing data block in event")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data structure"})
			return
		}

		// Safely extract email
		customer, ok := data["customer"].(map[string]interface{})
		if !ok {
			log.Println("Missing customer in webhook")
			c.JSON(http.StatusOK, gin.H{"status": "ignored – no customer"})
			return
		}
		email, ok := customer["email"].(string)
		if !ok || email == "" {
			log.Println("Invalid or missing customer email")
			c.JSON(http.StatusOK, gin.H{"status": "ignored – no email"})
			return
		}

		// Look up user by email
		var user models.User
		if err := h.DB.Where("email = ?", email).First(&user).Error; err != nil {
			log.Printf("User not found for email: %s\n", email)
			c.JSON(http.StatusOK, gin.H{"status": "user not found, skipping"})
			return
		}

		// Save payment and update status
		if err := h.Service.SaveSuccessfulPayment(user.ID, data); err != nil {
			log.Println("Failed to save payment:", err)
			c.JSON(http.StatusOK, gin.H{"status": "payment duplicate or error"}) // prevent retries
			return
		}

		log.Println("Payment confirmed for user:", user.Email)

		// TODO: send notification or email if needed

		c.JSON(http.StatusOK, gin.H{"status": "payment handled"})
		return
	}

	// Unhandled event
	log.Println("Unhandled Paystack event type:", eventType)
	c.JSON(http.StatusOK, gin.H{"status": "event ignored"})
}


// GetPaymentStatusByReference returns payment status by its reference
func (h *PaystackHandler) GetPaymentStatusByReference(c *gin.Context) {
	reference := c.Param("reference")

	var payment models.Payment
	if err := h.DB.Where("reference = ?", reference).First(&payment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"payment": payment,
	})
}
