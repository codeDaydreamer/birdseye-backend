package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services"
	"birdseye-backend/pkg/services/email"
	"crypto/subtle"
	"errors"
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
		payments.POST("/quote", handler.QuotePayment)
		payments.POST("/initiate", handler.InitiatePayment)
		payments.GET("", handler.ListPayments)
	}
}

// PaymentRequest is the incoming payload from frontend. The amount is never taken from it: it's
// worked out from the plan, billing period and promo code (services.QuotePlan).
type PaymentRequest struct {
	Phone         string `json:"PhoneNumber"`
	Reference     string `json:"reference"`
	PlanID        uint   `json:"plan_id"`
	BillingPeriod string `json:"billing_period"`
	PromoCode     string `json:"promo_code"`
}

// quoteForRequest prices what the request is for; with no plan given it renews the farm's
// current plan and billing period
func quoteForRequest(farmID uint, req PaymentRequest) (*services.Quote, error) {
	if req.PlanID == 0 || req.BillingPeriod == "" {
		var farm models.Farm
		if err := db.DB.Omit("Subscription", "BillingInfo").First(&farm, farmID).Error; err != nil {
			return nil, err
		}
		if req.PlanID == 0 {
			if farm.PlanID == nil {
				return nil, services.ErrPlanNotFound
			}
			req.PlanID = *farm.PlanID
		}
		if req.BillingPeriod == "" {
			req.BillingPeriod = farm.BillingPeriod
		}
	}
	return services.QuotePlan(req.PlanID, req.BillingPeriod, req.PromoCode)
}

// respondQuoteError answers pricing errors (unknown plan, period not offered, bad promo) with 400
func respondQuoteError(c *gin.Context, err error) {
	if errors.Is(err, services.ErrPlanNotFound) || errors.Is(err, services.ErrPeriodNotOffered) || errors.Is(err, services.ErrPromoInvalid) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	logrus.Errorf("Failed to price payment: %v", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not determine price"})
}

// QuotePayment shows what a plan + billing period (+ promo code) will cost before paying
func (h *PaymentHandler) QuotePayment(c *gin.Context) {
	farmID, _ := farmContext(c)
	var req PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}
	quote, err := quoteForRequest(farmID, req)
	if err != nil {
		respondQuoteError(c, err)
		return
	}
	c.JSON(http.StatusOK, quote)
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

// InitiatePayment pays for a plan: an M-Pesa STK push via Dynapay, or, when a promo code brings
// the price to zero, activates the plan straight away
func (h *PaymentHandler) InitiatePayment(c *gin.Context) {
	farmID, userID := farmContext(c)

	var req PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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

	quote, err := quoteForRequest(farmID, req)
	if err != nil {
		respondQuoteError(c, err)
		return
	}

	payment := models.Payment{
		UserID:         userID,
		FarmID:         farmID,
		Amount:         quote.Amount,
		PhoneNumber:    req.Phone,
		Reference:      req.Reference,
		PlanID:         &quote.PlanID,
		BillingPeriod:  quote.BillingPeriod,
		ListPrice:      quote.ListPrice,
		PromoCodeID:    quote.PromoCodeID,
		DiscountAmount: quote.DiscountAmount,
	}

	// Fully discounted: nothing to collect, so activate now
	if quote.Amount <= 0 {
		payment.Status = "success"
		payment.Gateway = "promo"
		if err := db.DB.Create(&payment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record payment"})
			return
		}
		if err := services.ApplySuccessfulPayment(&payment); err != nil {
			logrus.Errorf("Failed to apply free payment for farm %d: %v", farmID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to activate plan"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Plan activated", "reference": req.Reference, "amount": 0, "status": "success", "quote": quote})
		return
	}

	if strings.TrimSpace(req.Phone) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phone number is required"})
		return
	}

	webhook, err := dynapayWebhookURL()
	if err != nil {
		logrus.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server misconfiguration"})
		return
	}

	result, err := services.Dynapay.SendSTKPush(req.Phone, quote.Amount, req.Reference, webhook)
	if err != nil {
		logrus.Errorf("Dynapay STK Push failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to initiate payment", "details": err.Error()})
		return
	}

	// Record the payment now, so the webhook can only complete payments we started
	payment.Status = "initiated"
	payment.Gateway = "mpesa"
	payment.PaymentID = &result.PaymentID
	payment.MerchantRequestID = result.MerchantRequestID
	payment.CheckoutRequestID = result.CheckoutRequestID
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
		"amount":              quote.Amount,
		"quote":               quote,
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
		if err := services.ApplySuccessfulPayment(&payment); err != nil {
			logrus.Errorf("❌ Failed to apply payment %s to farm %d: %v", payment.ID, payment.FarmID, err)
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
