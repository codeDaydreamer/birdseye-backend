package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"log"
	"net/http"
"birdseye-backend/pkg/middlewares"
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
	userID, exists := c.Get("user_id") // Get user ID from context
	if !exists {
		log.Println("GetExpenses: User not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Fetch user data from the database
	user, err := models.GetUserByID(userID.(uint))
	if err != nil {
		log.Println("GetExpenses: Error fetching user from DB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user data"})
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
	userID, exists := c.Get("user_id") // Get user ID from context
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Fetch user data from the database
	user, err := models.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user data"})
		return
	}

	var expense models.Expense
	if err := c.ShouldBindJSON(&expense); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Assign the authenticated user's ID to the expense
	expense.UserID = user.ID

	if err := db.DB.Create(&expense).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create expense"})
		return
	}

	c.JSON(http.StatusCreated, expense)
}

// UpdateExpense updates an existing expense for the authenticated user
func (h *ExpenseHandler) UpdateExpense(c *gin.Context) {
	userID, exists := c.Get("user_id") // Get user ID from context
	if !exists {
		log.Println("UpdateExpense: User not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Fetch user data from the database
	user, err := models.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user data"})
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
	userID, exists := c.Get("user_id") // Get user ID from context
	if !exists {
		log.Println("DeleteExpense: User not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Fetch user data from the database
	user, err := models.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user data"})
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
