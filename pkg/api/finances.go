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

	financialRoutes := r.Group("/flock-financial").Use(middlewares.AuthMiddleware())
	{
		financialRoutes.GET("/", handler.GetFinancialData)
		financialRoutes.GET("/flock/:flockID", handler.GetFinancialDataByFlock)
		financialRoutes.POST("/", handler.AddOrUpdateFinancialData)
		financialRoutes.DELETE("/flock/:flockID", handler.DeleteFinancialData)
	}
}

// GetFinancialData retrieves all financial data for the authenticated user
func (h *FlockFinancialHandler) GetFinancialData(c *gin.Context) {
	log.Println("GET /flock-financial called")

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var financialData []models.FlocksFinancialData
	if err := db.DB.Where("user_id = ?", userID).Find(&financialData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve financial data"})
		return
	}

	c.JSON(http.StatusOK, financialData)
}

// GetFinancialDataByFlock retrieves financial data for a specific flock
func (h *FlockFinancialHandler) GetFinancialDataByFlock(c *gin.Context) {
	log.Println("GET /flock-financial/flock/:flockID called")

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	flockID, err := strconv.Atoi(c.Param("flockID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid flock ID"})
		return
	}

	var financialData models.FlocksFinancialData
	if err := db.DB.Where("flock_id = ? AND user_id = ?", flockID, userID).First(&financialData).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Financial data not found"})
		return
	}

	c.JSON(http.StatusOK, financialData)
}

// AddOrUpdateFinancialData adds or updates financial data for a flock
func (h *FlockFinancialHandler) AddOrUpdateFinancialData(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var financialData models.FlocksFinancialData
	if err := c.ShouldBindJSON(&financialData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	financialData.UserID = userID
	if err := db.DB.Where("flock_id = ? AND user_id = ? AND month = ? AND year = ?",
		financialData.FlockID, financialData.UserID, financialData.Month, financialData.Year).
		FirstOrCreate(&financialData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store financial data"})
		return
	}

	c.JSON(http.StatusOK, financialData)
}

// DeleteFinancialData removes financial data for a flock
func (h *FlockFinancialHandler) DeleteFinancialData(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	flockID, err := strconv.Atoi(c.Param("flockID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid flock ID"})
		return
	}

	if err := db.DB.Where("flock_id = ? AND user_id = ?", flockID, userID).Delete(&models.FlocksFinancialData{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete financial data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Financial data deleted successfully"})
}
