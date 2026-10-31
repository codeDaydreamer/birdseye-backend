package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"log"
	"net/http"
	"strconv"
"birdseye-backend/pkg/middlewares"
	"github.com/gin-gonic/gin"
	"birdseye-backend/pkg/services"
	

)

// ExpenseHandler handles expense-related requests
type ExpenseHandler struct {
	Service *services.ExpenseService // ✅ Inject ExpenseService
}

func SetupExpenseRoutes(r *gin.Engine, expenseService *services.ExpenseService) {
	handler := &ExpenseHandler{Service: expenseService} // Inject the service

	expenseRoutes := r.Group("/expenses").Use(middlewares.FarmAuth())
	{
		expenseRoutes.GET("/", handler.GetExpenses)
		expenseRoutes.POST("/", handler.AddExpense)
		expenseRoutes.PUT("/:id", handler.UpdateExpense)
		expenseRoutes.DELETE("/:id", handler.DeleteExpense)

		// Budget-related routes
		expenseRoutes.GET("/budget", handler.GetTotalBudget)
		expenseRoutes.PUT("/budget", handler.UpdateBudget)
	}
}



// GetExpenses retrieves the farm's expenses
func (h *ExpenseHandler) GetExpenses(c *gin.Context) {
	log.Println("GET /expenses called")
	farmID, _ := farmContext(c)

	var expenses []models.Expense
	if err := db.DB.Where("farm_id = ?", farmID).Find(&expenses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve expenses"})
		return
	}

	c.JSON(http.StatusOK, expenses)
}

// AddExpense adds a new expense to the farm
func (h *ExpenseHandler) AddExpense(c *gin.Context) {
	farmID, userID := farmContext(c)

	var expense models.Expense
	if err := c.ShouldBindJSON(&expense); err != nil {
		log.Printf("❌ Invalid request data: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !optionalFlockInFarm(&expense.FlockID, farmID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Flock not found"})
		return
	}

	expense.ID = 0
	expense.FarmID = farmID
	expense.CreatedByUserID = userID

	// ✅ Call ExpenseService to handle logic
	log.Println("📌 Calling ExpenseService to add expense...")
	if err := h.Service.AddExpense(&expense); err != nil {
		log.Printf("❌ Error adding expense: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create expense"})
		return
	}

	log.Println("✅ Expense successfully added!")
	c.JSON(http.StatusCreated, expense)
}

// UpdateExpense updates one of the farm's expenses
func (h *ExpenseHandler) UpdateExpense(c *gin.Context) {
	farmID, _ := farmContext(c)
	id := c.Param("id")

	var expense models.Expense
	if err := db.DB.Where("id = ? AND farm_id = ?", id, farmID).First(&expense).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Expense not found or unauthorized"})
		return
	}
	original := expense

	if err := c.ShouldBindJSON(&expense); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// The body can't move the expense to another record, farm or author
	expense.ID, expense.FarmID = original.ID, original.FarmID
	expense.CreatedByUserID, expense.CreatedAt = original.CreatedByUserID, original.CreatedAt
	if !optionalFlockInFarm(&expense.FlockID, farmID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Flock not found"})
		return
	}

	if err := h.Service.UpdateExpense(&expense); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update expense"})
		return
	}

	c.JSON(http.StatusOK, expense)
}

// DeleteExpense deletes one of the farm's expenses
func (h *ExpenseHandler) DeleteExpense(c *gin.Context) {
	farmID, _ := farmContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := h.Service.DeleteExpense(uint(id), farmID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Expense not found or unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Expense deleted successfully"})
}


func (h *ExpenseHandler) GetTotalBudget(c *gin.Context) {
	log.Println("GET /expenses/budget called")
	farmID, _ := farmContext(c)

	var totalBudget float64
	err := db.DB.Model(&models.Expense{}).
		Where("farm_id = ?", farmID).
		Select("COALESCE(SUM(budget), 0)").
		Scan(&totalBudget).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch total budget"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total_budget": totalBudget})
}
func (h *ExpenseHandler) UpdateBudget(c *gin.Context) {
	log.Println("PUT /expenses/budget called")
	farmID, _ := farmContext(c)

	var budgetUpdate struct {
		FlockID uint    `json:"flock_id"`
		Budget  float64 `json:"budget"`
	}

	if err := c.ShouldBindJSON(&budgetUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	err := db.DB.Model(&models.Expense{}).
		Where("flock_id = ? AND farm_id = ?", budgetUpdate.FlockID, farmID).
		Update("budget", budgetUpdate.Budget).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update budget"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Budget updated successfully"})
}
