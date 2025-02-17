package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services"
	"birdseye-backend/pkg/middlewares"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// FlockHandler handles flock-related requests
type FlockHandler struct {
    Service *services.FlockService
}

func SetupFlockRoutes(r *gin.Engine) {
    eggService := services.NewEggProductionService(db.DB) // Initialize EggProductionService
    salesService := services.NewSalesService(db.DB)       // Initialize SalesService
    expenseService := services.NewExpenseService(db.DB)   // ✅ Initialize ExpenseService

    flockService := services.NewFlockService(db.DB, eggService, salesService, expenseService) // ✅ Pass ExpenseService
    handler := &FlockHandler{Service: flockService}

    flockRoutes := r.Group("/flocks").Use(middlewares.AuthMiddleware())
    {
        flockRoutes.GET("/", handler.GetFlocks)
        flockRoutes.GET("/:id", handler.GetFlock)
        flockRoutes.POST("/", handler.AddFlock)
        flockRoutes.PUT("/:id", handler.UpdateFlock)
        flockRoutes.DELETE("/:id", handler.DeleteFlock)
    }
}

func (h *FlockHandler) GetFlocks(c *gin.Context) {
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

    // Use the user ID to fetch the flocks for the user
    flocks, err := h.Service.GetFlocksByUser(user.ID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve flocks"})
        return
    }

    // Recalculate and update metrics before returning the data
    for i := range flocks {
        h.Service.CalculateFlockMetrics(&flocks[i], user.ID)
    }

    c.JSON(http.StatusOK, flocks)
}

func (h *FlockHandler) GetFlock(c *gin.Context) {
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

    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
        return
    }

    flock, err := h.Service.GetFlockByID(uint(id), user.ID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Flock not found"})
        return
    }

    // Recalculate and update metrics before returning the data
    h.Service.CalculateFlockMetrics(flock, user.ID)

    c.JSON(http.StatusOK, flock)
}

// AddFlock adds a new flock
func (h *FlockHandler) AddFlock(c *gin.Context) {
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

    var flock models.Flock
    if err := c.ShouldBindJSON(&flock); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    flock.UserID = user.ID // Use user.ID as uint

    if err := db.DB.Create(&flock).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create flock"})
        return
    }

    c.JSON(http.StatusCreated, flock)
}

func (h *FlockHandler) UpdateFlock(c *gin.Context) {
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

    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
        return
    }

    flock, err := h.Service.GetFlockByID(uint(id), user.ID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Flock not found"})
        return
    }

    // Bind JSON data to the existing flock
    if err := c.ShouldBindJSON(&flock); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if err := h.Service.UpdateFlock(flock); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update flock"})
        return
    }

    c.JSON(http.StatusOK, flock)
}

// DeleteFlock deletes a flock for the authenticated user
func (h *FlockHandler) DeleteFlock(c *gin.Context) {
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

    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
        return
    }

    if err := db.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.Flock{}).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete flock"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Flock deleted successfully"})
}
