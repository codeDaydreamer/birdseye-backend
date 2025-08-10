package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/services"
	"net/http"
	
	"github.com/gin-gonic/gin"
	"birdseye-backend/pkg/middlewares"
	"github.com/sirupsen/logrus"
	"time"
	"birdseye-backend/pkg/reports"
	
)

// ReportsHandler handles report-related requests
type ReportsHandler struct {
	ReportsService *services.ReportsService
}

// SetupReportsRoutes sets up the routes for report generation
func SetupReportsRoutes(r *gin.Engine) {
	handler := &ReportsHandler{
		ReportsService: services.NewReportsService(db.DB),
	}

	reportsRoutes := r.Group("/reports").Use(middlewares.AuthMiddleware())
	{
		reportsRoutes.GET("/user/:userID", handler.GetUserReports)
		reportsRoutes.POST("/expenses", handler.GenerateExpenseReport)
		reportsRoutes.POST("/sales", handler.GenerateSalesReport)
		reportsRoutes.POST("/egg-production", handler.GenerateEggProductionReport)
		reportsRoutes.POST("/inventory", handler.GenerateInventoryReport)
		reportsRoutes.POST("/flock", handler.GenerateFlockReport)          // Existing route
		reportsRoutes.POST("/financial", handler.GenerateFinancialReport)  // New route
		reportsRoutes.DELETE("/:reportID", handler.DeleteReport)

	}
}






func (h *ReportsHandler) GetUserReports(c *gin.Context) {
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

    // Retrieve the reports for the authenticated user
    reports, err := h.ReportsService.GetUserReports(authUserID)
    if err != nil {
        logrus.Errorf("Failed to retrieve user reports: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user reports"})
        return
    }

    logrus.Infof("Successfully retrieved reports for user ID %d", authUserID)


    c.JSON(http.StatusOK, gin.H{
        "reports": reports,
    })
}

func (h *ReportsHandler) GenerateExpenseReport(c *gin.Context) {
	// Get userID from authentication middleware
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authorized"})
		return
	}

	// Convert userID to uint
	authUserID, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Parse request parameters
	var request struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
		UserID    uint   `json:"user_id"` // Ensure this matches the logged-in user
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	// Ensure the request user matches the authenticated user
	if request.UserID != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized user ID"})
		return
	}

	// Convert string dates from ISO 8601 to `time.Time`
	startDate, err := time.Parse(time.RFC3339, request.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
		return
	}

	endDate, err := time.Parse(time.RFC3339, request.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format"})
		return
	}

	// Generate the expense report
	pdfPath, err := reports.GenerateExpenseReport(db.DB, authUserID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report", "details": err.Error()})
		return
	}

	// Send the file as response
	c.File(pdfPath)
}

func (h *ReportsHandler) GenerateSalesReport(c *gin.Context) {
	// Get userID from authentication middleware
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authorized"})
		return
	}

	// Convert userID to uint
	authUserID, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Parse request parameters
	var request struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
		UserID    uint   `json:"user_id"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	// Ensure the request user matches the authenticated user
	if request.UserID != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized user ID"})
		return
	}

	// Convert string dates from ISO 8601 to `time.Time`
	startDate, err := time.Parse(time.RFC3339, request.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
		return
	}

	endDate, err := time.Parse(time.RFC3339, request.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format"})
		return
	}

	// Generate the sales report
	pdfPath, err := reports.GenerateSalesReport(db.DB, authUserID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report", "details": err.Error()})
		return
	}

	// Send the file as response
	c.File(pdfPath)
}

func (h *ReportsHandler) GenerateEggProductionReport(c *gin.Context) {
	logrus.Info("GenerateEggProductionReport called")

	// Get userID from authentication middleware
	userID, exists := c.Get("user_id")
	if !exists {
		logrus.Warn("user_id missing in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authorized"})
		return
	}
	logrus.Infof("Authenticated user ID from context: %v", userID)

	// Convert userID to uint
	authUserID, ok := userID.(uint)
	if !ok {
		logrus.Errorf("Invalid user ID type: %T", userID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Parse request parameters
	var request struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
		UserID    uint   `json:"user_id"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		logrus.Errorf("Failed to parse request JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}
	logrus.Infof("Request JSON parsed: %+v", request)

	// Ensure the request user matches the authenticated user
	if request.UserID != authUserID {
		logrus.Warnf("User ID mismatch: request.UserID=%d, authUserID=%d", request.UserID, authUserID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized user ID"})
		return
	}

	// Convert string dates from ISO 8601 to `time.Time`
	startDate, err := time.Parse(time.RFC3339, request.StartDate)
	if err != nil {
		logrus.Errorf("Invalid start date format: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
		return
	}

	endDate, err := time.Parse(time.RFC3339, request.EndDate)
	if err != nil {
		logrus.Errorf("Invalid end date format: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format"})
		return
	}
	logrus.Infof("Parsed dates - StartDate: %v, EndDate: %v", startDate, endDate)

	// Generate the egg production report
	pdfPath, err := reports.GenerateEggProductionReport(db.DB, authUserID, startDate, endDate)
	if err != nil {
		logrus.Errorf("Failed to generate egg production report: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report", "details": err.Error()})
		return
	}

	logrus.Infof("Egg production report generated at: %s", pdfPath)

	// Send the file as response
	c.File(pdfPath)
}


func (h *ReportsHandler) GenerateInventoryReport(c *gin.Context) {
	// Get userID from authentication middleware
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authorized"})
		return
	}

	// Convert userID to uint
	authUserID, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Parse request parameters
	var request struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
		UserID    uint   `json:"user_id"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	// Ensure the request user matches the authenticated user
	if request.UserID != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized user ID"})
		return
	}

	// Convert string dates from ISO 8601 to `time.Time`
	startDate, err := time.Parse(time.RFC3339, request.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
		return
	}

	endDate, err := time.Parse(time.RFC3339, request.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format"})
		return
	}

	// Generate the inventory report
	pdfPath, err := reports.GenerateInventoryReport(db.DB, authUserID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report", "details": err.Error()})
		return
	}

	// Send the file as response
	c.File(pdfPath)
}

func (h *ReportsHandler) GenerateFlockReport(c *gin.Context) {
	// Get userID from authentication middleware
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authorized"})
		return
	}

	// Convert userID to uint
	authUserID, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Parse request parameters
	var request struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
		UserID    uint   `json:"user_id"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	// Ensure the request user matches the authenticated user
	if request.UserID != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized user ID"})
		return
	}

	// Convert string dates from ISO 8601 to `time.Time`
	startDate, err := time.Parse(time.RFC3339, request.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
		return
	}

	endDate, err := time.Parse(time.RFC3339, request.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format"})
		return
	}

	// Generate the flock report
	pdfPath, err := reports.GenerateFlockReport(db.DB, authUserID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report", "details": err.Error()})
		return
	}

	// Send the file as response
	c.File(pdfPath)
}

func (h *ReportsHandler) GenerateFinancialReport(c *gin.Context) {
	// Get userID from authentication middleware
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authorized"})
		return
	}

	// Convert userID to uint
	authUserID, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Generate the financial report for all flocks of the user
	pdfPath, err := reports.GenerateFinancialReport(db.DB, authUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report", "details": err.Error()})
		return
	}

	// Send the file as response
	c.File(pdfPath)
}
func (h *ReportsHandler) DeleteReport(c *gin.Context) {
    // Get userID from authentication middleware
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authorized"})
        return
    }

    // Convert userID to uint
    authUserID, ok := userID.(uint)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
        return
    }

    // Get report ID from URL parameter
    reportID := c.Param("reportID")

    // Call the service to delete the report
    err := h.ReportsService.DeleteReport(authUserID, reportID)
    if err != nil {
        logrus.Errorf("Failed to delete report %s: %v", reportID, err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete report"})
        return
    }

    logrus.Infof("Report %s deleted successfully by user %d", reportID, authUserID)
    c.JSON(http.StatusOK, gin.H{"message": "Report deleted successfully"})
}
