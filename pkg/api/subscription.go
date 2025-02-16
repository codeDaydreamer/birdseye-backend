package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SubscriptionHandler handles subscription-related requests
type SubscriptionHandler struct{}

// SetupSubscriptionRoutes sets up subscription API routes
func SetupSubscriptionRoutes(r *gin.Engine) {
	handler := &SubscriptionHandler{}

	subRoutes := r.Group("/subscriptions").Use(middlewares.AuthMiddleware())
	{
		subRoutes.GET("/", handler.GetSubscription)
		subRoutes.POST("/", handler.AddSubscription)
		subRoutes.PUT("/", handler.UpdateSubscription)
		subRoutes.DELETE("/", handler.DeleteSubscription)
	}
}

// GetSubscription retrieves the subscription for the authenticated user
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := userVal.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user data"})
		return
	}

	var subscription models.Subscription
	if err := db.DB.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}

	c.JSON(http.StatusOK, subscription)
}

// AddSubscription creates a new subscription
func (h *SubscriptionHandler) AddSubscription(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDVal.(int)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	var subscription models.Subscription
	if err := c.ShouldBindJSON(&subscription); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	subscription.UserID = userID

	if err := db.DB.Create(&subscription).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subscription"})
		return
	}

	c.JSON(http.StatusCreated, subscription)
}

// UpdateSubscription modifies an existing subscription
func (h *SubscriptionHandler) UpdateSubscription(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := userVal.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user data"})
		return
	}

	var subscription models.Subscription
	if err := db.DB.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}

	if err := c.ShouldBindJSON(&subscription); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := db.DB.Save(&subscription).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subscription"})
		return
	}

	c.JSON(http.StatusOK, subscription)
}

// DeleteSubscription removes a subscription
func (h *SubscriptionHandler) DeleteSubscription(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := userVal.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user data"})
		return
	}

	if err := db.DB.Where("user_id = ?", user.ID).Delete(&models.Subscription{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Subscription deleted successfully"})
}
