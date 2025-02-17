package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/middlewares"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// EggProductionHandler handles egg production-related requests
type EggProductionHandler struct{}

// SetupEggProductionRoutes sets up the API routes with authentication middleware
func SetupEggProductionRoutes(r *gin.Engine) {
	handler := &EggProductionHandler{}

	routes := r.Group("/egg-productions").Use(middlewares.AuthMiddleware())
	{
		routes.GET("/", handler.GetEggProduction)
		routes.POST("/", handler.AddEggProduction)
		routes.PUT("/:id", handler.UpdateEggProduction)
		routes.DELETE("/:id", handler.DeleteEggProduction)
	}
}

// GetEggProduction retrieves egg production records for the authenticated user
func (h *EggProductionHandler) GetEggProduction(c *gin.Context) {
	userID, exists := c.Get("user_id")  // Get user ID from context
	if !exists {
		log.Println("GetEggProduction: User not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Fetch user data from the database
	user, err := models.GetUserByID(userID.(uint))
	if err != nil {
		log.Println("GetEggProduction: Error fetching user from DB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user data"})
		return
	}

	var records []models.EggProduction
	if err := db.DB.Table("egg_productions").
		Select("egg_productions.*, flocks.name AS flock_name").
		Joins("JOIN flocks ON flocks.id = egg_productions.flock_id").
		Where("egg_productions.user_id = ?", user.ID).
		Find(&records).Error; err != nil {
		log.Println("GetEggProduction: Error retrieving records:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve records"})
		return
	}

	c.JSON(http.StatusOK, records)
}

// AddEggProduction adds a new egg production record for the authenticated user
func (h *EggProductionHandler) AddEggProduction(c *gin.Context) {
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

	var record models.EggProduction
	if err := c.ShouldBindJSON(&record); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Assign authenticated user's ID to the record
	record.UserID = user.ID

	if err := db.DB.Create(&record).Error; err != nil {
		log.Println("AddEggProduction: Error creating record:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create record"})
		return
	}

	c.JSON(http.StatusCreated, record)
}

// UpdateEggProduction updates an existing record for the authenticated user
func (h *EggProductionHandler) UpdateEggProduction(c *gin.Context) {
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

	id := c.Param("id")

	var record models.EggProduction
	if err := db.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&record).Error; err != nil {
		log.Println("UpdateEggProduction: Record not found or unauthorized")
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found or unauthorized"})
		return
	}

	if err := c.ShouldBindJSON(&record); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := db.DB.Save(&record).Error; err != nil {
		log.Println("UpdateEggProduction: Error updating record:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update record"})
		return
	}

	c.JSON(http.StatusOK, record)
}

// DeleteEggProduction deletes a record for the authenticated user
func (h *EggProductionHandler) DeleteEggProduction(c *gin.Context) {
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

	id := c.Param("id")

	var record models.EggProduction
	if err := db.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&record).Error; err != nil {
		log.Println("DeleteEggProduction: Record not found or unauthorized")
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found or unauthorized"})
		return
	}

	if err := db.DB.Delete(&record).Error; err != nil {
		log.Println("DeleteEggProduction: Error deleting record:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}
