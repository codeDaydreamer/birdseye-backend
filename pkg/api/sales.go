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
		salesRoutes.GET("/flock/:flockID", handler.GetSalesByFlock) // New route ✅
		salesRoutes.POST("/", handler.AddSale)
		salesRoutes.PUT("/:id", handler.UpdateSale)
		salesRoutes.DELETE("/:id", handler.DeleteSale)
	}
}

// GetSales retrieves sales records for the authenticated user
func (h *SalesHandler) GetSales(c *gin.Context) {
	log.Println("GET /sales called")
	userVal, exists := c.Get("user")
	if !exists {
		log.Println("GetSales: User not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := userVal.(*models.User)
	if !ok {
		log.Println("GetSales: User data type mismatch")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user data"})
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

	userVal, exists := c.Get("user")
	if !exists {
		log.Println("GetSalesByFlock: User not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := userVal.(*models.User)
	if !ok {
		log.Println("GetSalesByFlock: User data type mismatch")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user data"})
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

	var sale models.Sale
	if err := c.ShouldBindJSON(&sale); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Assign the authenticated user's ID to the sale (with proper type conversion)
	sale.UserID = uint(userID)

	if err := db.DB.Create(&sale).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sale"})
		return
	}

	c.JSON(http.StatusCreated, sale)
}

// UpdateSale updates an existing sale for the authenticated user
func (h *SalesHandler) UpdateSale(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		log.Println("UpdateSale: User not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := userVal.(*models.User)
	if !ok {
		log.Println("UpdateSale: User data type mismatch")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user data"})
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
	userVal, exists := c.Get("user")
	if !exists {
		log.Println("DeleteSale: User not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := userVal.(*models.User)
	if !ok {
		log.Println("DeleteSale: User data type mismatch")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user data"})
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
