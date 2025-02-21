package api

import (
	"log"
	"net/http"
	"time"
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services"
	"github.com/gin-gonic/gin"
)

// FinancialHandler handles financial data requests
type FinancialHandler struct {
	FinanceService *services.FlockFinancialService
}

// SetupFinancialRoutes initializes financial API endpoints
func SetupFinancialRoutes(r *gin.Engine) {
	// Ensure FinanceService is initialized properly
	financeService := services.NewFinanceService(db.DB) // Initialize FinanceService first
	flockFinancialService := services.NewFlockFinancialService(db.DB, financeService)

	handler := &FinancialHandler{
		FinanceService: flockFinancialService,
	}

	financialRoutes := r.Group("/financials").Use(middlewares.AuthMiddleware())
	{
		financialRoutes.GET("/week", handler.GetCurrentWeekFinancialData)
		financialRoutes.GET("/month", handler.GetCurrentMonthFinancialData)
		financialRoutes.GET("/year", handler.GetCurrentYearFinancialData)
		financialRoutes.GET("/day", handler.GetCurrentDayFinancialData)
	}
}

// GetCurrentWeekFinancialData retrieves weekly financial data
func (h *FinancialHandler) GetCurrentWeekFinancialData(c *gin.Context) {
	log.Println("GET /financials/week called")
	h.handleFinancialRequest(c, models.GetCurrentWeekPeriod)
}

// GetCurrentMonthFinancialData retrieves monthly financial data
func (h *FinancialHandler) GetCurrentMonthFinancialData(c *gin.Context) {
	log.Println("GET /financials/month called")
	h.handleFinancialRequest(c, models.GetCurrentMonthPeriod)
}

// GetCurrentYearFinancialData retrieves yearly financial data
func (h *FinancialHandler) GetCurrentYearFinancialData(c *gin.Context) {
	log.Println("GET /financials/year called")
	h.handleFinancialRequest(c, models.GetCurrentYearPeriod)
}

// GetCurrentDayFinancialData retrieves daily financial data
func (h *FinancialHandler) GetCurrentDayFinancialData(c *gin.Context) {
	log.Println("GET /financials/day called")
	h.handleFinancialRequest(c, models.GetCurrentDayPeriod)
}

// handleFinancialRequest is a helper to reduce repeated code
func (h *FinancialHandler) handleFinancialRequest(c *gin.Context, periodFunc func() (time.Time, time.Time)) {
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

	// Calculate the period
	periodStart, periodEnd := periodFunc()

	// Fetch financial data
	financeData, err := h.FinanceService.GetFlockFinancialData(userID, periodStart, periodEnd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve financial data"})
		return
	}

	c.JSON(http.StatusOK, financeData)
}
