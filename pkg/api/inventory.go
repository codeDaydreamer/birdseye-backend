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

// InventoryHandler handles inventory-related requests
type InventoryHandler struct {
	Service *services.InventoryService
}

// SetupInventoryRoutes sets up the inventory API routes with authentication middleware
func SetupInventoryRoutes(r *gin.Engine) {
	handler := &InventoryHandler{Service: services.NewInventoryService(db.DB)}

	inventoryRoutes := r.Group("/inventory").Use(middlewares.FarmAuth())
	{
		inventoryRoutes.GET("/", handler.GetInventory)
		inventoryRoutes.POST("/", handler.AddInventoryItem)
		inventoryRoutes.PUT("/:id", handler.UpdateInventoryItem)
		inventoryRoutes.DELETE("/:id", handler.DeleteInventoryItem)
	}
}
// GetInventory retrieves the farm's inventory records, including flock names
func (h *InventoryHandler) GetInventory(c *gin.Context) {
	farmID, _ := farmContext(c)

	var items []models.InventoryItem
	// Fetch inventory items and preload the Flock data (to get the flock name)
	if err := db.DB.Preload("Flock").Where("farm_id = ?", farmID).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve inventory items"})
		return
	}

	// Include flock name in the response
	for i := range items {
		if items[i].Flock != nil {
			items[i].FlockName = items[i].Flock.Name
		}
	}

	c.JSON(http.StatusOK, items)
}

// AddInventoryItem adds a new inventory item to the farm
func (h *InventoryHandler) AddInventoryItem(c *gin.Context) {
	farmID, _ := farmContext(c)

	var item models.InventoryItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !flockInFarm(item.FlockID, farmID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Flock not found"})
		return
	}

	item.ID = 0
	item.FarmID = farmID

	if err := h.Service.AddInventoryItem(&item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create inventory item"})
		return
	}

	c.JSON(http.StatusCreated, item)
}

// UpdateInventoryItem updates one of the farm's inventory items
func (h *InventoryHandler) UpdateInventoryItem(c *gin.Context) {
	farmID, _ := farmContext(c)
	id := c.Param("id")

	var item models.InventoryItem
	if err := db.DB.Where("id = ? AND farm_id = ?", id, farmID).First(&item).Error; err != nil {
		log.Println("UpdateInventoryItem: Item not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "Inventory item not found or unauthorized"})
		return
	}
	original := item

	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// The body can't move the item to another record or farm
	item.ID, item.FarmID = original.ID, original.FarmID
	if item.FlockID != original.FlockID && !flockInFarm(item.FlockID, farmID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Flock not found"})
		return
	}

	if err := h.Service.UpdateInventoryItem(&item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update inventory item"})
		return
	}

	c.JSON(http.StatusOK, item)
}

// DeleteInventoryItem deletes one of the farm's inventory items
func (h *InventoryHandler) DeleteInventoryItem(c *gin.Context) {
	farmID, _ := farmContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := h.Service.DeleteInventoryItem(uint(id), farmID); err != nil {
		log.Println("DeleteInventoryItem: Item not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "Inventory item not found or unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Inventory item deleted successfully"})
}
