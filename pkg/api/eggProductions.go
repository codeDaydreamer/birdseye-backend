package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/middlewares"
	"log"
	"net/http"
	"strconv"

	"birdseye-backend/pkg/services"

	"github.com/gin-gonic/gin"
)

// EggProductionHandler handles egg production-related requests
type EggProductionHandler struct {
	Service *services.EggProductionService
}

// SetupEggProductionRoutes sets up the API routes with authentication middleware
func SetupEggProductionRoutes(r *gin.Engine) {
	handler := &EggProductionHandler{Service: services.NewEggProductionService(db.DB)}

	routes := r.Group("/egg-productions").Use(middlewares.FarmAuth())
	{
		routes.GET("/", handler.GetEggProduction)
		routes.POST("/", handler.AddEggProduction)
		routes.PUT("/:id", handler.UpdateEggProduction)
		routes.DELETE("/:id", handler.DeleteEggProduction)
	}
}

// GetEggProduction retrieves the farm's egg production records
func (h *EggProductionHandler) GetEggProduction(c *gin.Context) {
	farmID, _ := farmContext(c)

	var records []models.EggProduction
	if err := db.DB.Table("egg_productions").
		Select("egg_productions.*, flocks.name AS flock_name").
		Joins("JOIN flocks ON flocks.id = egg_productions.flock_id").
		Where("egg_productions.farm_id = ?", farmID).
		Find(&records).Error; err != nil {
		log.Println("GetEggProduction: Error retrieving records:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve records"})
		return
	}

	c.JSON(http.StatusOK, records)
}

// AddEggProduction adds a new egg production record to the farm
func (h *EggProductionHandler) AddEggProduction(c *gin.Context) {
	farmID, _ := farmContext(c)

	var record models.EggProduction
	if err := c.ShouldBindJSON(&record); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !flockInFarm(record.FlockID, farmID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Flock not found"})
		return
	}

	record.ID = 0
	record.FarmID = farmID

	if err := h.Service.AddEggProduction(&record); err != nil {
		log.Println("AddEggProduction: Error creating record:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create record"})
		return
	}

	c.JSON(http.StatusCreated, record)
}

// UpdateEggProduction updates one of the farm's records
func (h *EggProductionHandler) UpdateEggProduction(c *gin.Context) {
	farmID, _ := farmContext(c)
	id := c.Param("id")

	var record models.EggProduction
	if err := db.DB.Where("id = ? AND farm_id = ?", id, farmID).First(&record).Error; err != nil {
		log.Println("UpdateEggProduction: Record not found or unauthorized")
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found or unauthorized"})
		return
	}
	original := record

	if err := c.ShouldBindJSON(&record); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// The body can't move the record to another record or farm
	record.ID, record.FarmID, record.CreatedAt = original.ID, original.FarmID, original.CreatedAt
	if record.FlockID != original.FlockID && !flockInFarm(record.FlockID, farmID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Flock not found"})
		return
	}

	if err := h.Service.UpdateEggProduction(&record); err != nil {
		log.Println("UpdateEggProduction: Error updating record:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update record"})
		return
	}

	c.JSON(http.StatusOK, record)
}

// DeleteEggProduction deletes one of the farm's records
func (h *EggProductionHandler) DeleteEggProduction(c *gin.Context) {
	farmID, _ := farmContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := h.Service.DeleteEggProduction(uint(id), farmID); err != nil {
		log.Println("DeleteEggProduction: Record not found or unauthorized")
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found or unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}
