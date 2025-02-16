package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
)

// BillingHandler handles billing-related requests
type BillingHandler struct{}

// SetupBillingRoutes sets up billing API routes
func SetupBillingRoutes(r *gin.Engine) {
	handler := &BillingHandler{}

	billingRoutes := r.Group("/billing").Use(middlewares.AuthMiddleware())
	{
		billingRoutes.GET("/", handler.GetBillingInfo)
		billingRoutes.POST("/", handler.AddBillingInfo)
		billingRoutes.PUT("/", handler.UpdateBillingInfo)
		billingRoutes.DELETE("/", handler.DeleteBillingInfo)
	}
}

// GetBillingInfo retrieves billing information for the authenticated user
func (h *BillingHandler) GetBillingInfo(c *gin.Context) {
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

	var billing models.BillingInfo
	if err := db.DB.Where("user_id = ?", user.ID).First(&billing).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Billing info not found"})
		return
	}

	c.JSON(http.StatusOK, billing)
}

// AddBillingInfo creates billing info
func (h *BillingHandler) AddBillingInfo(c *gin.Context) {
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

	var billing models.BillingInfo
	if err := c.ShouldBindJSON(&billing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	billing.UserID = userID

	if err := db.DB.Create(&billing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create billing info"})
		return
	}

	c.JSON(http.StatusCreated, billing)
}

// UpdateBillingInfo updates billing details
func (h *BillingHandler) UpdateBillingInfo(c *gin.Context) {
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

	var billing models.BillingInfo
	if err := db.DB.Where("user_id = ?", user.ID).First(&billing).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Billing info not found"})
		return
	}

	if err := c.ShouldBindJSON(&billing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := db.DB.Save(&billing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update billing info"})
		return
	}

	c.JSON(http.StatusOK, billing)
}

// DeleteBillingInfo removes billing info
func (h *BillingHandler) DeleteBillingInfo(c *gin.Context) {
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

	if err := db.DB.Where("user_id = ?", user.ID).Delete(&models.BillingInfo{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete billing info"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Billing info deleted successfully"})
}
