package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/middlewares"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// FlockHandler handles flock-related requests
type FlockHandler struct{}

// SetupFlockRoutes sets up the flock API routes with authentication middleware
func SetupFlockRoutes(r *gin.Engine) {
	handler := &FlockHandler{} // Create an instance of FlockHandler

	// Apply authentication middleware to all flock routes
	flockRoutes := r.Group("/flocks").Use(middlewares.AuthMiddleware())
	{
		flockRoutes.GET("/", handler.GetFlocks)
		flockRoutes.GET("/:id", handler.GetFlock)
		flockRoutes.POST("/", handler.AddFlock)
		flockRoutes.PUT("/:id", handler.UpdateFlock)
		flockRoutes.DELETE("/:id", handler.DeleteFlock)
	}
}

// GetFlocks retrieves all flocks for the authenticated user, including related data
func (h *FlockHandler) GetFlocks(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := userVal.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user data"})
		return
	}

	var flocks []models.Flock
	if err := db.DB.
		Preload("EggProductions").
		Preload("VaccinationSchedule").
		Where("user_id = ?", user.ID).
		Find(&flocks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve flocks"})
		return
	}

	c.JSON(http.StatusOK, flocks)
}


// GetFlock retrieves a single flock with related data
func (h *FlockHandler) GetFlock(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := userVal.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user data"})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var flock models.Flock
	if err := db.DB.
		Preload("EggProductions").
		Preload("VaccinationSchedule").
		Where("id = ? AND user_id = ?", id, user.ID).
		First(&flock).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Flock not found"})
		return
	}

	c.JSON(http.StatusOK, flock)
}


// AddFlock adds a new flock for the authenticated user
func (h *FlockHandler) AddFlock(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := userVal.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user data"})
		return
	}

	var flock models.Flock
	if err := c.ShouldBindJSON(&flock); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Assign authenticated user's ID
	flock.UserID = uint(user.ID)

	// Initialize empty slices if not provided
	if flock.EggProduction == nil {
		flock.EggProduction = []int{}
	}
	if flock.MortalityRateData == nil {
		flock.MortalityRateData = []int{}
	}
	if flock.FeedConsumption == nil {
		flock.FeedConsumption = []int{}
	}
	if flock.SalesData == nil {
		flock.SalesData = []int{}
	}
	if flock.FeedQualityList == nil {
		flock.FeedQualityList = []models.FeedQuality{}
	}
	if flock.VaccinationSchedule == nil {
		flock.VaccinationSchedule = []models.Vaccination{}
	}

	if err := db.DB.Create(&flock).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create flock"})
		return
	}

	c.JSON(http.StatusCreated, flock)
}

// UpdateFlock updates an existing flock for the authenticated user
func (h *FlockHandler) UpdateFlock(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := userVal.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user data"})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var flock models.Flock
	if err := db.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&flock).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Flock not found"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Only update the fields that exist in updateData
	if err := db.DB.Model(&flock).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update flock"})
		return
	}

	c.JSON(http.StatusOK, flock)
}

// DeleteFlock deletes a flock for the authenticated user
func (h *FlockHandler) DeleteFlock(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := userVal.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user data"})
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
