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
	handler := &EggProductionHandler{} // Create an instance of EggProductionHandler

	// Apply authentication middleware to all routes
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
	log.Println("GET /egg-productions called")
	userVal, exists := c.Get("user")
	if !exists {
		log.Println("GetEggProduction: User not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := userVal.(*models.User)
	if !ok {
		log.Println("GetEggProduction: User data type mismatch")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user data"})
		return
	}

	var records []models.EggProduction
	if err := db.DB.Where("user_id = ?", user.ID).Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve records"})
		return
	}

	c.JSON(http.StatusOK, records)
}

// AddEggProduction adds a new egg production record for the authenticated user
func (h *EggProductionHandler) AddEggProduction(c *gin.Context) {
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

	var record models.EggProduction
	if err := c.ShouldBindJSON(&record); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Assign the authenticated user's ID to the record
	record.UserID = uint(userID)

	if err := db.DB.Create(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create record"})
		return
	}

	c.JSON(http.StatusCreated, record)
}

// UpdateEggProduction updates an existing record for the authenticated user
func (h *EggProductionHandler) UpdateEggProduction(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		log.Println("UpdateEggProduction: User not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := userVal.(*models.User)
	if !ok {
		log.Println("UpdateEggProduction: User data type mismatch")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user data"})
		return
	}

	id := c.Param("id")

	var record models.EggProduction
	if err := db.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&record).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found or unauthorized"})
		return
	}

	if err := c.ShouldBindJSON(&record); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := db.DB.Save(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update record"})
		return
	}

	c.JSON(http.StatusOK, record)
}

// DeleteEggProduction deletes a record for the authenticated user
func (h *EggProductionHandler) DeleteEggProduction(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		log.Println("DeleteEggProduction: User not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := userVal.(*models.User)
	if !ok {
		log.Println("DeleteEggProduction: User data type mismatch")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user data"})
		return
	}

	id := c.Param("id")

	var record models.EggProduction
	if err := db.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&record).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found or unauthorized"})
		return
	}

	if err := db.DB.Delete(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}
