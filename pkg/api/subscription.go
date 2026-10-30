package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SubscriptionHandler handles the farm's subscription record
type SubscriptionHandler struct{}

// SetupSubscriptionRoutes sets up subscription API routes: any member can read, only owners change it
func SetupSubscriptionRoutes(r *gin.Engine) {
	handler := &SubscriptionHandler{}
	ownerOnly := middlewares.RequireFarmRole(models.FarmRoleOwner)

	subRoutes := r.Group("/subscriptions").Use(middlewares.FarmAuth())
	{
		subRoutes.GET("/", handler.GetSubscription)
		subRoutes.POST("/", ownerOnly, handler.SaveSubscription)
		subRoutes.PUT("/", ownerOnly, handler.SaveSubscription)
		subRoutes.DELETE("/", ownerOnly, handler.DeleteSubscription)
	}
}

// GetSubscription retrieves the farm's subscription
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	farmID, _ := farmContext(c)

	var subscription models.Subscription
	if err := db.DB.Where("farm_id = ?", farmID).First(&subscription).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}

	c.JSON(http.StatusOK, subscription)
}

// SaveSubscription creates or updates the farm's subscription (one per farm)
func (h *SubscriptionHandler) SaveSubscription(c *gin.Context) {
	farmID, _ := farmContext(c)

	var subscription models.Subscription
	db.DB.Where("farm_id = ?", farmID).Limit(1).Find(&subscription)
	id := subscription.ID

	if err := c.ShouldBindJSON(&subscription); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	subscription.ID, subscription.FarmID = id, farmID

	if err := db.DB.Save(&subscription).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save subscription"})
		return
	}

	c.JSON(http.StatusOK, subscription)
}

// DeleteSubscription removes the farm's subscription
func (h *SubscriptionHandler) DeleteSubscription(c *gin.Context) {
	farmID, _ := farmContext(c)

	if err := db.DB.Where("farm_id = ?", farmID).Delete(&models.Subscription{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Subscription deleted successfully"})
}
