package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SubscriptionRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256DH string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func HandlePushSubscription(c *gin.Context) {
	userID := c.GetUint("user_id") // set by UserAuth

	var req SubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription data"})
		return
	}

	sub := models.PushSubscription{
		UserID:   userID,
		Endpoint: req.Endpoint,
		P256DH:   req.Keys.P256DH,
		Auth:     req.Keys.Auth,
	}

	// Upsert: if subscription with same endpoint exists, update it, else create new
	err := db.DB.
		Where("user_id = ? AND endpoint = ?", userID, req.Endpoint).
		Assign(sub).
		FirstOrCreate(&sub).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Push subscription saved"})
}

// HandleRegisterDevice saves the mobile app's Firebase token so the phone gets push notifications
func HandleRegisterDevice(c *gin.Context) {
	var req struct {
		Token    string `json:"token" binding:"required"`
		Platform string `json:"platform"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Token) > 255 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device token"})
		return
	}
	if err := services.SaveDeviceToken(db.DB, c.GetUint("user_id"), req.Token, req.Platform); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save device"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Device registered"})
}

// HandleUnregisterDevice stops push notifications to the phone (called when the user signs out)
func HandleUnregisterDevice(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device token"})
		return
	}
	if err := services.DeleteDeviceToken(db.DB, c.GetUint("user_id"), req.Token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove device"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Device removed"})
}
