package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/services"
	"net/http"
	"github.com/gin-gonic/gin"
	"fmt"
)

type EggAdjustmentHandler struct {
	Service *services.EggAdjustmentService
}

func SetupEggAdjustmentRoutes(r *gin.Engine) {
	handler := &EggAdjustmentHandler{
		Service: services.NewEggAdjustmentService(db.DB),
	}

	routes := r.Group("/egg-adjustments").Use(middlewares.FarmAuth())
	{
		routes.GET("/", handler.GetAdjustments)
		routes.POST("/", handler.AddAdjustment)
		routes.PUT("/:id", handler.UpdateAdjustment)
		routes.DELETE("/:id", handler.DeleteAdjustment)
	}
}

func (h *EggAdjustmentHandler) GetAdjustments(c *gin.Context) {
	farmID, _ := farmContext(c)
	adjustments, err := h.Service.GetAdjustmentsByFarm(farmID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch adjustments"})
		return
	}
	c.JSON(http.StatusOK, adjustments)
}

func (h *EggAdjustmentHandler) AddAdjustment(c *gin.Context) {
	farmID, _ := farmContext(c)
	var adj models.EggAdjustment
	if err := c.ShouldBindJSON(&adj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !flockInFarm(adj.FlockID, farmID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Flock not found"})
		return
	}
	adj.ID = 0
	adj.FarmID = farmID

	if err := h.Service.AddAdjustment(&adj); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add adjustment"})
		return
	}
	c.JSON(http.StatusCreated, adj)
}

func (h *EggAdjustmentHandler) UpdateAdjustment(c *gin.Context) {
	farmID, _ := farmContext(c)
	id := parseUint(c.Param("id"))

	var existing models.EggAdjustment
	if err := db.DB.Where("id = ? AND farm_id = ?", id, farmID).First(&existing).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "adjustment not found"})
		return
	}

	var adj models.EggAdjustment
	if err := c.ShouldBindJSON(&adj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if adj.FlockID != existing.FlockID && !flockInFarm(adj.FlockID, farmID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Flock not found"})
		return
	}
	adj.ID = existing.ID
	adj.FarmID = farmID
	adj.CreatedAt = existing.CreatedAt

	if err := h.Service.UpdateAdjustment(&adj); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update adjustment"})
		return
	}
	c.JSON(http.StatusOK, adj)
}

func (h *EggAdjustmentHandler) DeleteAdjustment(c *gin.Context) {
	farmID, _ := farmContext(c)
	id := parseUint(c.Param("id"))

	if err := h.Service.DeleteAdjustment(id, farmID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Adjustment deleted successfully"})
}

// Helper to convert string ID to uint
func parseUint(s string) uint {
	var id uint
	fmt.Sscanf(s, "%d", &id)
	return id
}
