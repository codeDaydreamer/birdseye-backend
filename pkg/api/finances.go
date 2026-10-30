package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// FlockFinancialHandler handles flock financial data requests
type FlockFinancialHandler struct{}

// SetupFlockFinancialRoutes sets up the financial data API routes
func SetupFlockFinancialRoutes(r *gin.Engine) {
	handler := &FlockFinancialHandler{}

	financialRoutes := r.Group("/flock-financial").Use(middlewares.FarmAuth())
	{
		financialRoutes.GET("/", handler.GetFinancialData)
		financialRoutes.GET("/flock/:flockID", handler.GetFinancialDataByFlock)
		financialRoutes.POST("/", handler.AddOrUpdateFinancialData)
		financialRoutes.PUT("/", handler.UpdateFinancialData)
		financialRoutes.DELETE("/flock/:flockID", handler.DeleteFinancialData)
	}
}

// GetFinancialData retrieves all financial data for the farm
func (h *FlockFinancialHandler) GetFinancialData(c *gin.Context) {
	log.Println("GET /flock-financial called")

	farmID, _ := farmContext(c)

	var financialData []models.FlocksFinancialData
	if err := db.DB.Where("farm_id = ?", farmID).Find(&financialData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve financial data"})
		return
	}

	c.JSON(http.StatusOK, financialData)
}

// GetFinancialDataByFlock retrieves financial data for a specific flock
func (h *FlockFinancialHandler) GetFinancialDataByFlock(c *gin.Context) {
	log.Println("GET /flock-financial/flock/:flockID called")

	farmID, _ := farmContext(c)

	flockID, err := strconv.Atoi(c.Param("flockID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid flock ID"})
		return
	}

	var financialData models.FlocksFinancialData
	if err := db.DB.Where("flock_id = ? AND farm_id = ?", flockID, farmID).First(&financialData).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Financial data not found"})
		return
	}

	c.JSON(http.StatusOK, financialData)
}

// AddOrUpdateFinancialData adds or updates financial data for a flock
func (h *FlockFinancialHandler) AddOrUpdateFinancialData(c *gin.Context) {
	farmID, _ := farmContext(c)

	var financialData models.FlocksFinancialData
	if err := c.ShouldBindJSON(&financialData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !flockInFarm(financialData.FlockID, farmID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Flock not found"})
		return
	}

	financialData.ID = 0
	financialData.FarmID = farmID
	if err := db.DB.Where("flock_id = ? AND farm_id = ? AND month = ? AND year = ?",
		financialData.FlockID, financialData.FarmID, financialData.Month, financialData.Year).
		FirstOrCreate(&financialData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store financial data"})
		return
	}

	c.JSON(http.StatusOK, financialData)
}

// UpdateFinancialData updates existing financial data
func (h *FlockFinancialHandler) UpdateFinancialData(c *gin.Context) {
	farmID, _ := farmContext(c)

	var updatedData models.FlocksFinancialData
	if err := c.ShouldBindJSON(&updatedData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existingData models.FlocksFinancialData
	if err := db.DB.Where("flock_id = ? AND farm_id = ? AND month = ? AND year = ?",
		updatedData.FlockID, farmID, updatedData.Month, updatedData.Year).
		First(&existingData).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Financial data not found"})
		return
	}

	// The body can't move the record to another record or farm
	updatedData.ID, updatedData.FarmID = 0, 0
	if err := db.DB.Model(&existingData).Updates(updatedData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update financial data"})
		return
	}

	c.JSON(http.StatusOK, existingData)
}

// DeleteFinancialData removes financial data for a flock
func (h *FlockFinancialHandler) DeleteFinancialData(c *gin.Context) {
	farmID, _ := farmContext(c)

	flockID, err := strconv.Atoi(c.Param("flockID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid flock ID"})
		return
	}

	if err := db.DB.Where("flock_id = ? AND farm_id = ?", flockID, farmID).Delete(&models.FlocksFinancialData{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete financial data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Financial data deleted successfully"})
}
