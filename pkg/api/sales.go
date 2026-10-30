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

	salesRoutes := r.Group("/sales").Use(middlewares.FarmAuth())
	{
		salesRoutes.GET("/", handler.GetSales)
		salesRoutes.GET("/flock/:flockID", handler.GetSalesByFlock)
		salesRoutes.POST("/", handler.AddSale)
		salesRoutes.PUT("/:id", handler.UpdateSale)
		salesRoutes.DELETE("/:id", handler.DeleteSale)
	}
}

// GetSales retrieves the farm's sales records
func (h *SalesHandler) GetSales(c *gin.Context) {
	log.Println("GET /sales called")
	farmID, _ := farmContext(c)

	var sales []models.Sale
	if err := db.DB.Where("farm_id = ?", farmID).Find(&sales).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve sales"})
		return
	}

	c.JSON(http.StatusOK, sales)
}

// GetSalesByFlock retrieves sales records for one of the farm's flocks
func (h *SalesHandler) GetSalesByFlock(c *gin.Context) {
	log.Println("GET /sales/flock/:flockID called")
	farmID, _ := farmContext(c)

	flockID := c.Param("flockID")
	var sales []models.Sale
	if err := db.DB.Where("flock_id = ? AND farm_id = ?", flockID, farmID).Find(&sales).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve sales for flock"})
		return
	}

	c.JSON(http.StatusOK, sales)
}

// AddSale adds a new sale to the farm
func (h *SalesHandler) AddSale(c *gin.Context) {
	farmID, userID := farmContext(c)

	var sale models.Sale
	if err := c.ShouldBindJSON(&sale); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !flockInFarm(sale.FlockID, farmID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Flock not found"})
		return
	}

	sale.ID = 0
	sale.FarmID = farmID
	sale.CreatedByUserID = userID
	if err := db.DB.Omit("Flock").Create(&sale).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sale"})
		return
	}

	c.JSON(http.StatusCreated, sale)
}

// UpdateSale updates one of the farm's sales
func (h *SalesHandler) UpdateSale(c *gin.Context) {
	farmID, _ := farmContext(c)

	id := c.Param("id")
	var sale models.Sale
	if err := db.DB.Where("id = ? AND farm_id = ?", id, farmID).First(&sale).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sale not found or unauthorized"})
		return
	}
	original := sale

	if err := c.ShouldBindJSON(&sale); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// The body can't move the sale to another record, farm or author
	sale.ID, sale.FarmID = original.ID, original.FarmID
	sale.CreatedByUserID = original.CreatedByUserID
	sale.RefNo, sale.CreatedAt = original.RefNo, original.CreatedAt

	if sale.FlockID != original.FlockID && !flockInFarm(sale.FlockID, farmID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Flock not found"})
		return
	}

	if err := db.DB.Omit("Flock").Save(&sale).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sale"})
		return
	}

	c.JSON(http.StatusOK, sale)
}

// DeleteSale deletes one of the farm's sales
func (h *SalesHandler) DeleteSale(c *gin.Context) {
	farmID, _ := farmContext(c)

	id := c.Param("id")
	var sale models.Sale
	if err := db.DB.Where("id = ? AND farm_id = ?", id, farmID).First(&sale).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sale not found or unauthorized"})
		return
	}

	if err := db.DB.Delete(&sale).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete sale"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sale deleted successfully"})
}
