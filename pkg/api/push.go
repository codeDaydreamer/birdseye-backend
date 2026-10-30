package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
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
