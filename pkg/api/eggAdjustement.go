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

	routes := r.Group("/egg-adjustments").Use(middlewares.AuthMiddleware())
	{
		routes.GET("/", handler.GetAdjustments)
		routes.POST("/", handler.AddAdjustment)
		routes.PUT("/:id", handler.UpdateAdjustment)
		routes.DELETE("/:id", handler.DeleteAdjustment)
	}
}

func (h *EggAdjustmentHandler) GetAdjustments(c *gin.Context) {
	userID := c.GetUint("user_id")
	adjustments, err := h.Service.GetAdjustmentsByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch adjustments"})
		return
	}
	c.JSON(http.StatusOK, adjustments)
}

func (h *EggAdjustmentHandler) AddAdjustment(c *gin.Context) {
	userID := c.GetUint("user_id")
	var adj models.EggAdjustment
	if err := c.ShouldBindJSON(&adj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	adj.UserID = userID

	if err := h.Service.AddAdjustment(&adj); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add adjustment"})
		return
	}
	c.JSON(http.StatusCreated, adj)
}

func (h *EggAdjustmentHandler) UpdateAdjustment(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var adj models.EggAdjustment
	if err := c.ShouldBindJSON(&adj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	adj.ID = parseUint(id)
	adj.UserID = userID

	if err := h.Service.UpdateAdjustment(&adj); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update adjustment"})
		return
	}
	c.JSON(http.StatusOK, adj)
}

func (h *EggAdjustmentHandler) DeleteAdjustment(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := parseUint(c.Param("id"))

	if err := h.Service.DeleteAdjustment(id, userID); err != nil {
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
