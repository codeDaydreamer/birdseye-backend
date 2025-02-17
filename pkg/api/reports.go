package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/services"
	"net/http"
	"time"
	"github.com/gin-gonic/gin"
	"birdseye-backend/pkg/middlewares"
	"github.com/sirupsen/logrus"
)

// ReportsHandler handles report-related requests
type ReportsHandler struct {
	ReportsService *services.ReportsService
}

// SetupReportsRoutes sets up the routes for report generation
func SetupReportsRoutes(r *gin.Engine) {
	// Initialize the ReportsService with the db connection
	handler := &ReportsHandler{
		ReportsService: services.NewReportsService(db.DB),
	}

	reportsRoutes := r.Group("/reports").Use(middlewares.AuthMiddleware()) // Use AuthMiddleware here
	{
		reportsRoutes.POST("/sales", handler.GenerateSalesReport)
		reportsRoutes.POST("/inventory", handler.GenerateInventoryReport)
	}
}

// GenerateSalesReport handles the report generation for sales
func (h *ReportsHandler) GenerateSalesReport(c *gin.Context) {
	// Get userID from the context (set by AuthMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		logrus.Error("User not authorized: user_id missing in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authorized"})
		return
	}

	// Ensure userID is uint
	authUserID, ok := userID.(uint)
	if !ok {
		logrus.Errorf("Invalid user ID: %v, unable to cast to uint", userID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get the report data from the request body
	var requestData struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
		UserID    uint   `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		logrus.Errorf("Failed to bind request data: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Verify that the user ID matches the authenticated user ID
	if requestData.UserID != authUserID {
		logrus.Warnf("User ID mismatch: expected %d, got %d", authUserID, requestData.UserID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID mismatch"})
		return
	}

	// Convert the string dates to time.Time
	startDate, err := time.Parse(time.RFC3339, requestData.StartDate)
	if err != nil {
		logrus.Errorf("Invalid start date format: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
		return
	}

	endDate, err := time.Parse(time.RFC3339, requestData.EndDate)
	if err != nil {
		logrus.Errorf("Invalid end date format: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format"})
		return
	}

	// Call the service method and pass the userID to ensure multi-tenancy
	report, err := h.ReportsService.GenerateSalesReport(startDate, endDate, authUserID)
	if err != nil {
		logrus.Errorf("Failed to generate sales report: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate sales report"})
		return
	}

	logrus.Infof("Successfully generated sales report for user ID %d", authUserID)
	c.JSON(http.StatusOK, report)
}

// GenerateInventoryReport handles the report generation for inventory
func (h *ReportsHandler) GenerateInventoryReport(c *gin.Context) {
	// Get userID from the context (set by AuthMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		logrus.Error("User not authorized: user_id missing in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authorized"})
		return
	}

	// Ensure userID is uint
	authUserID, ok := userID.(uint)
	if !ok {
		logrus.Errorf("Invalid user ID: %v, unable to cast to uint", userID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get user ID from request body (make sure the user ID matches the authenticated user)
	var requestData struct {
		UserID uint `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		logrus.Errorf("Failed to bind request data: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Verify that the user ID matches the authenticated user ID
	if requestData.UserID != authUserID {
		logrus.Warnf("User ID mismatch: expected %d, got %d", authUserID, requestData.UserID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID mismatch"})
		return
	}

	// Call the service method and pass the userID to ensure multi-tenancy
	report, err := h.ReportsService.GenerateInventoryReport(authUserID)
	if err != nil {
		logrus.Errorf("Failed to generate inventory report: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate inventory report"})
		return
	}

	logrus.Infof("Successfully generated inventory report for user ID %d", authUserID)
	c.JSON(http.StatusOK, report)
}
