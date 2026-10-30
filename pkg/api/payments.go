package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services"
	"birdseye-backend/pkg/services/email"
	"encoding/json"
	"errors"
	"io"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
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

	result, err := services.Dynapay.CreatePayment(req.Phone, quote.Amount, req.Reference,
		fmt.Sprintf("Birdseye %s %s", quote.PlanName, quote.BillingPeriod),
		map[string]interface{}{"farm_id": farmID, "user_id": userID, "plan_id": quote.PlanID})
	if err != nil {
		logrus.Errorf("Dynapay payment request failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to initiate payment", "details": err.Error()})
		return
	}

	// Record the payment now, so the webhook can only complete payments we started
	payment.Status = "initiated"
	payment.Gateway = "mpesa"
	payment.PaymentID = &result.ID
	payment.MerchantRequestID = result.ProviderMerchantID
	payment.CheckoutRequestID = result.ProviderRequestID
	if err := db.DB.Create(&payment).Error; err != nil {
		logrus.Errorf("Failed to record initiated payment for farm %d: %v", farmID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record payment"})
		return
	}

	logrus.Infof("M-Pesa payment %s started for farm %d by user %d", result.ID, farmID, userID)

	c.JSON(http.StatusOK, gin.H{
		"message":             "Check your phone to complete the M-Pesa payment",
		"checkout_request_id": result.ProviderRequestID,
		"merchant_request_id": result.ProviderMerchantID,
		"reference":           req.Reference,
		"amount":              quote.Amount,
		"quote":               quote,
	})
}

func RegisterWebhookRoutes(r *gin.Engine) {
	r.POST("/webhooks/dynapay-payment", HandleDynapayWebhook)
}

// HandleDynapayWebhook applies a payment outcome Dynapay reports. The body is signed with the
// webhook secret Dynapay issued for Birdseye (DYNAPAY_WEBHOOK_SECRET).
func HandleDynapayWebhook(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid body"})
		return
	}
	if err := services.VerifyDynapaySignature(c.GetHeader("Dynapay-Signature"), body); err != nil {
		logrus.Warnf("Rejected Dynapay webhook: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	var event struct {
		ID   string                  `json:"id"`
		Type string                  `json:"type"`
		Data services.DynapayPayment `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil || event.Data.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	var payment models.Payment
	if err := db.DB.Where("payment_id = ?", event.Data.ID).First(&payment).Error; err != nil {
		// Not one of ours (or already cleaned up): acknowledge so Dynapay stops retrying
		logrus.Warnf("Dynapay webhook %s for unknown payment %s", event.ID, event.Data.ID)
		c.JSON(http.StatusOK, gin.H{"message": "Ignored"})
		return
	}

	changed, err := services.ApplyDynapayResult(&payment, &event.Data)
	if err != nil {
		logrus.Errorf("Failed to apply Dynapay result to payment %s: %v", payment.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update payment"})
		return
	}
	if changed {
		NotifyPaymentOutcome(&payment)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Webhook processed successfully"})
}

// NotifyPaymentOutcome emails the result of a settled M-Pesa payment: success to the farm's owners and
// managers, failure to whoever paid
func NotifyPaymentOutcome(payment *models.Payment) {
	if payment.Status == "success" {
		logrus.Infof("Farm %d paid via payment %s", payment.FarmID, payment.ID)
		managers, _ := services.GetFarmManagers(payment.FarmID)
		for _, user := range managers {
			go func(u models.User) {
				if err := email.SendPaymentSuccessEmail(u.Email, u.Username, payment.MpesaReference, payment.Amount); err != nil {
					logrus.Errorf("Failed to send payment success email: %v", err)
				}
			}(user)
		}
		return
	}

	var payer models.User
	if err := db.DB.First(&payer, payment.UserID).Error; err != nil {
		return
	}
	msg := payment.ResultDescription
	if msg == "" {
		msg = "Payment was not successful"
	}
	go func() {
		if err := email.SendPaymentFailureEmail(payer.Email, payer.Username, payment.Amount, msg); err != nil {
			logrus.Errorf("Failed to send payment failure email: %v", err)
		}
	}()
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
