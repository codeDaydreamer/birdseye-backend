package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// BillingHandler handles the farm's billing details
type BillingHandler struct{}

// SetupBillingRoutes sets up billing API routes: owners and managers can read, only owners change them
func SetupBillingRoutes(r *gin.Engine) {
	handler := &BillingHandler{}
	ownerOnly := middlewares.RequireFarmRole(models.FarmRoleOwner)

	billingRoutes := r.Group("/billing").Use(middlewares.FarmAuth())
	{
		billingRoutes.GET("/", middlewares.RequireFarmRole(models.FarmRoleOwner, models.FarmRoleManager), handler.GetBillingInfo)
		billingRoutes.POST("/", ownerOnly, handler.SaveBillingInfo)
		billingRoutes.PUT("/", ownerOnly, handler.SaveBillingInfo)
		billingRoutes.DELETE("/", ownerOnly, handler.DeleteBillingInfo)
	}
}

// GetBillingInfo retrieves the farm's billing details
func (h *BillingHandler) GetBillingInfo(c *gin.Context) {
	farmID, _ := farmContext(c)

	var billing models.BillingInfo
	if err := db.DB.Where("farm_id = ?", farmID).First(&billing).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Billing info not found"})
		return
	}

	c.JSON(http.StatusOK, billing)
}

// SaveBillingInfo creates or updates the farm's billing details (one per farm)
func (h *BillingHandler) SaveBillingInfo(c *gin.Context) {
	farmID, _ := farmContext(c)

	var billing models.BillingInfo
	db.DB.Where("farm_id = ?", farmID).Limit(1).Find(&billing)
	id, token := billing.ID, billing.ProviderToken

	if err := c.ShouldBindJSON(&billing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Clients can't set the farm or the provider token
	billing.ID, billing.FarmID, billing.ProviderToken = id, farmID, token

	if err := db.DB.Save(&billing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save billing info"})
		return
	}

	c.JSON(http.StatusOK, billing)
}

// DeleteBillingInfo removes the farm's billing details
func (h *BillingHandler) DeleteBillingInfo(c *gin.Context) {
	farmID, _ := farmContext(c)

	if err := db.DB.Where("farm_id = ?", farmID).Delete(&models.BillingInfo{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete billing info"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Billing info deleted successfully"})
}
