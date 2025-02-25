package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/middlewares"
	"log"
	"net/http"
	"strconv"
	"errors"
	"github.com/gin-gonic/gin"
)

// BudgetHandler handles budget-related requests
type BudgetHandler struct{}

// SetupBudgetRoutes sets up the budget API routes with authentication middleware
func SetupBudgetRoutes(r *gin.Engine) {
	handler := &BudgetHandler{}

	budgetRoutes := r.Group("/budget").Use(middlewares.AuthMiddleware())
	{
		budgetRoutes.GET("/", handler.GetBudgets)
		budgetRoutes.GET("/flock/:flockID", handler.GetBudgetsByFlock)
		budgetRoutes.GET("/:month/:year", handler.GetBudgetByMonthYear)
		budgetRoutes.POST("/", handler.AddBudget)
		budgetRoutes.PUT("/:month/:year", handler.UpdateBudget)
		budgetRoutes.DELETE("/:month/:year", handler.DeleteBudget)
	}
}

// GetBudgets retrieves all budgets for the authenticated user
func (h *BudgetHandler) GetBudgets(c *gin.Context) {
	log.Println("GET /budget called")

	user, err := getUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var budgets []models.Budget
	if err := db.DB.Where("user_id = ?", user.ID).Find(&budgets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve budgets"})
		return
	}

	c.JSON(http.StatusOK, budgets)
}

// GetBudgetsByFlock retrieves all budgets for a specific flock
func (h *BudgetHandler) GetBudgetsByFlock(c *gin.Context) {
	log.Println("GET /budget/flock/:flockID called")

	user, err := getUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	flockID := c.Param("flockID")
	var budgets []models.Budget
	if err := db.DB.Where("flock_id = ? AND user_id = ?", flockID, user.ID).Find(&budgets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve budgets for flock"})
		return
	}

	c.JSON(http.StatusOK, budgets)
}

// GetBudgetByMonthYear retrieves the budget for a specific month and year
func (h *BudgetHandler) GetBudgetByMonthYear(c *gin.Context) {
	log.Println("GET /budget/:month/:year called")

	user, err := getUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	month, err := strconv.Atoi(c.Param("month"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid month"})
		return
	}

	year, err := strconv.Atoi(c.Param("year"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid year"})
		return
	}

	var budget models.Budget
	if err := db.DB.Where("user_id = ? AND month = ? AND year = ?", user.ID, month, year).First(&budget).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
		return
	}

	c.JSON(http.StatusOK, budget)
}

// AddBudget adds a new budget for the authenticated user
func (h *BudgetHandler) AddBudget(c *gin.Context) {
	user, err := getUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var budget models.Budget
	if err := c.ShouldBindJSON(&budget); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	budget.UserID = int(user.ID)
	if err := db.DB.Create(&budget).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create budget"})
		return
	}

	c.JSON(http.StatusCreated, budget)
}

// UpdateBudget updates an existing budget for the authenticated user
func (h *BudgetHandler) UpdateBudget(c *gin.Context) {
	user, err := getUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	month, err := strconv.Atoi(c.Param("month"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid month"})
		return
	}

	year, err := strconv.Atoi(c.Param("year"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid year"})
		return
	}

	var budget models.Budget
	if err := db.DB.Where("user_id = ? AND month = ? AND year = ?", user.ID, month, year).First(&budget).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
		return
	}

	var input struct {
		Amount float64 `json:"amount" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	budget.Amount = input.Amount
	if err := db.DB.Save(&budget).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update budget"})
		return
	}

	c.JSON(http.StatusOK, budget)
}

// DeleteBudget deletes a budget for the authenticated user
func (h *BudgetHandler) DeleteBudget(c *gin.Context) {
	user, err := getUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	month, err := strconv.Atoi(c.Param("month"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid month"})
		return
	}

	year, err := strconv.Atoi(c.Param("year"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid year"})
		return
	}

	var budget models.Budget
	if err := db.DB.Where("user_id = ? AND month = ? AND year = ?", user.ID, month, year).First(&budget).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
		return
	}

	if err := db.DB.Delete(&budget).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete budget"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Budget deleted successfully"})
}

// Helper function to get the authenticated user
func getUserFromContext(c *gin.Context) (*models.User, error) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return nil, errors.New("Unauthorized")
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		return nil, errors.New("invalid user id")
	}

	user, err := models.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	return user, nil
}
