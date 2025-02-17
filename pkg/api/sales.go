package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/middlewares"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SalesHandler handles sales-related requests
type SalesHandler struct{}

// SetupSalesRoutes sets up the sales API routes with authentication middleware
func SetupSalesRoutes(r *gin.Engine) {
	handler := &SalesHandler{}

	salesRoutes := r.Group("/sales").Use(middlewares.AuthMiddleware())
	{
		salesRoutes.GET("/", handler.GetSales)
		salesRoutes.GET("/flock/:flockID", handler.GetSalesByFlock)
		salesRoutes.POST("/", handler.AddSale)
		salesRoutes.PUT("/:id", handler.UpdateSale)
		salesRoutes.DELETE("/:id", handler.DeleteSale)
	}
}

// GetSales retrieves sales records for the authenticated user
func (h *SalesHandler) GetSales(c *gin.Context) {
	log.Println("GET /sales called")

	// Fetch user ID from the context
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Fetch user details from the database using the GetUserByID function
	user, err := models.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var sales []models.Sale
	if err := db.DB.Where("user_id = ?", user.ID).Find(&sales).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve sales"})
		return
	}

	c.JSON(http.StatusOK, sales)
}

// GetSalesByFlock retrieves sales records for a specific flock
func (h *SalesHandler) GetSalesByFlock(c *gin.Context) {
	log.Println("GET /sales/flock/:flockID called")

	// Fetch user ID from the context
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Fetch user details from the database using the GetUserByID function
	user, err := models.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	flockID := c.Param("flockID")
	var sales []models.Sale
	if err := db.DB.Where("flock_id = ? AND user_id = ?", flockID, user.ID).Find(&sales).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve sales for flock"})
		return
	}

	c.JSON(http.StatusOK, sales)
}

// AddSale adds a new sale for the authenticated user
func (h *SalesHandler) AddSale(c *gin.Context) {
	// Fetch user ID from the context
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Fetch user details from the database using the GetUserByID function
	user, err := models.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var sale models.Sale
	if err := c.ShouldBindJSON(&sale); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sale.UserID = user.ID
	if err := db.DB.Create(&sale).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sale"})
		return
	}

	c.JSON(http.StatusCreated, sale)
}

// UpdateSale updates an existing sale for the authenticated user
func (h *SalesHandler) UpdateSale(c *gin.Context) {
	// Fetch user ID from the context
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Fetch user details from the database using the GetUserByID function
	user, err := models.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("id")
	var sale models.Sale
	if err := db.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&sale).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sale not found or unauthorized"})
		return
	}

	if err := c.ShouldBindJSON(&sale); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := db.DB.Save(&sale).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sale"})
		return
	}

	c.JSON(http.StatusOK, sale)
}

// DeleteSale deletes a sale for the authenticated user
func (h *SalesHandler) DeleteSale(c *gin.Context) {
	// Fetch user ID from the context
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Fetch user details from the database using the GetUserByID function
	user, err := models.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("id")
	var sale models.Sale
	if err := db.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&sale).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sale not found or unauthorized"})
		return
	}

	if err := db.DB.Delete(&sale).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete sale"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sale deleted successfully"})
}
