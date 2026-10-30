package api

import (
	"birdseye-backend/pkg/broadcast"
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"log"
	"net/http"
	"strconv"
)

// BudgetHandler handles budget-related requests
type BudgetHandler struct{}

// SetupBudgetRoutes sets up the budget API routes with authentication middleware
func SetupBudgetRoutes(r *gin.Engine) {
	handler := &BudgetHandler{}

	budgetRoutes := r.Group("/budget").Use(middlewares.FarmAuth(), requireFeature("budgets"))
	{
		budgetRoutes.GET("/", handler.GetBudgets)
		budgetRoutes.GET("/flock/:flockID", handler.GetBudgetsByFlock)
		budgetRoutes.GET("/:month/:year", handler.GetBudgetByMonthYear)
		budgetRoutes.POST("/", handler.AddBudget)
		budgetRoutes.PUT("/:month/:year", handler.UpdateBudget)
		budgetRoutes.DELETE("/:month/:year", handler.DeleteBudget)
	}
}

// GetBudgets retrieves all budgets for the farm
func (h *BudgetHandler) GetBudgets(c *gin.Context) {
	log.Println("GET /budget called")

	farmID, _ := farmContext(c)

	var budgets []models.Budget
	if err := db.DB.Where("farm_id = ?", farmID).Find(&budgets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve budgets"})
		return
	}

	c.JSON(http.StatusOK, budgets)
}

// GetBudgetsByFlock retrieves all budgets for a specific flock
func (h *BudgetHandler) GetBudgetsByFlock(c *gin.Context) {
	log.Println("GET /budget/flock/:flockID called")

	farmID, _ := farmContext(c)

	flockID := c.Param("flockID")
	var budgets []models.Budget
	if err := db.DB.Where("flock_id = ? AND farm_id = ?", flockID, farmID).Find(&budgets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve budgets for flock"})
		return
	}

	c.JSON(http.StatusOK, budgets)
}

// GetBudgetByMonthYear retrieves the budget for a specific month and year
func (h *BudgetHandler) GetBudgetByMonthYear(c *gin.Context) {
	log.Println("GET /budget/:month/:year called")

	farmID, _ := farmContext(c)

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
	if err := db.DB.Where("farm_id = ? AND month = ? AND year = ?", farmID, month, year).First(&budget).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
		return
	}

	c.JSON(http.StatusOK, budget)
}

// AddBudget adds a new budget for the farm
func (h *BudgetHandler) AddBudget(c *gin.Context) {
	farmID, _ := farmContext(c)

	var budget models.Budget
	if err := c.ShouldBindJSON(&budget); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !flockInFarm(budget.FlockID, farmID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Flock not found"})
		return
	}

	budget.ID = 0
	budget.FarmID = farmID
	if err := db.DB.Omit("Flock").Create(&budget).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create budget"})
		return
	}
	broadcast.SendBudgetUpdate(farmID, "budget_added", budget)

	c.JSON(http.StatusCreated, budget)
}

// UpdateBudget updates an existing budget for the farm
func (h *BudgetHandler) UpdateBudget(c *gin.Context) {
	farmID, _ := farmContext(c)

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
	if err := budgetForMonth(c, farmID, month, year).First(&budget).Error; err != nil {
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
	if err := db.DB.Omit("Flock").Save(&budget).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update budget"})
		return
	}
	broadcast.SendBudgetUpdate(farmID, "budget_updated", budget)

	c.JSON(http.StatusOK, budget)
}

// DeleteBudget deletes a budget for the farm
func (h *BudgetHandler) DeleteBudget(c *gin.Context) {
	farmID, _ := farmContext(c)

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
	if err := budgetForMonth(c, farmID, month, year).First(&budget).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
		return
	}

	if err := db.DB.Delete(&budget).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete budget"})
		return
	}
	broadcast.SendBudgetUpdate(farmID, "budget_deleted", budget.ID)

	c.JSON(http.StatusOK, gin.H{"message": "Budget deleted successfully"})
}

// budgetForMonth finds the farm's budget for a month; a farm can budget each flock separately, so an
// optional ?flock_id= picks which flock's budget (without it, the first one for that month)
func budgetForMonth(c *gin.Context, farmID uint, month, year int) *gorm.DB {
	q := db.DB.Where("farm_id = ? AND month = ? AND year = ?", farmID, month, year)
	if flockID, err := strconv.Atoi(c.Query("flock_id")); err == nil {
		q = q.Where("flock_id = ?", flockID)
	}
	return q
}
