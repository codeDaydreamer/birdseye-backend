package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/middlewares"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ExpenseHandler handles expense-related requests
type ExpenseHandler struct{}

// SetupExpenseRoutes sets up the expense API routes with authentication middleware
func SetupExpenseRoutes(r *gin.Engine) {
	handler := &ExpenseHandler{} // Create an instance of ExpenseHandler

	// Apply authentication middleware to all expense routes
	expenseRoutes := r.Group("/expenses").Use(middlewares.AuthMiddleware())
	{
		expenseRoutes.GET("/", handler.GetExpenses)
		expenseRoutes.POST("/", handler.AddExpense)
		expenseRoutes.PUT("/:id", handler.UpdateExpense)
		expenseRoutes.DELETE("/:id", handler.DeleteExpense)
	}
}

// GetExpenses retrieves expenses for the authenticated user
func (h *ExpenseHandler) GetExpenses(c *gin.Context) {
	log.Println("GET /expenses called")
	userVal, exists := c.Get("user")
	if !exists {
		log.Println("GetExpenses: User not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := userVal.(*models.User)
	if !ok {
		log.Println("GetExpenses: User data type mismatch")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user data"})
		return
	}

	var expenses []models.Expense
	if err := db.DB.Where("user_id = ?", user.ID).Find(&expenses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve expenses"})
		return
	}

	c.JSON(http.StatusOK, expenses)
}
// AddExpense adds a new expense for the authenticated user
func (h *ExpenseHandler) AddExpense(c *gin.Context) {
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

	var expense models.Expense
	if err := c.ShouldBindJSON(&expense); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Assign the authenticated user's ID to the expense (with proper type conversion)
	expense.UserID = uint(userID)

	if err := db.DB.Create(&expense).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create expense"})
		return
	}

	c.JSON(http.StatusCreated, expense)
}

// UpdateExpense updates an existing expense for the authenticated user
func (h *ExpenseHandler) UpdateExpense(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		log.Println("UpdateExpense: User not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := userVal.(*models.User)
	if !ok {
		log.Println("UpdateExpense: User data type mismatch")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user data"})
		return
	}

	id := c.Param("id")

	var expense models.Expense
	if err := db.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&expense).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Expense not found or unauthorized"})
		return
	}

	if err := c.ShouldBindJSON(&expense); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := db.DB.Save(&expense).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update expense"})
		return
	}

	c.JSON(http.StatusOK, expense)
}

// DeleteExpense deletes an expense for the authenticated user
func (h *ExpenseHandler) DeleteExpense(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		log.Println("DeleteExpense: User not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := userVal.(*models.User)
	if !ok {
		log.Println("DeleteExpense: User data type mismatch")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user data"})
		return
	}

	id := c.Param("id")

	var expense models.Expense
	if err := db.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&expense).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Expense not found or unauthorized"})
		return
	}

	if err := db.DB.Delete(&expense).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete expense"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Expense deleted successfully"})
}
