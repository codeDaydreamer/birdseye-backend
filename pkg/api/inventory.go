package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/middlewares"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// InventoryHandler handles inventory-related requests
type InventoryHandler struct{}

// SetupInventoryRoutes sets up the inventory API routes with authentication middleware
func SetupInventoryRoutes(r *gin.Engine) {
	handler := &InventoryHandler{}

	inventoryRoutes := r.Group("/inventory").Use(middlewares.AuthMiddleware())
	{
		inventoryRoutes.GET("/", handler.GetInventory)
		inventoryRoutes.POST("/", handler.AddInventoryItem)
		inventoryRoutes.PUT("/:id", handler.UpdateInventoryItem)
		inventoryRoutes.DELETE("/:id", handler.DeleteInventoryItem)
	}
}
// GetInventory retrieves inventory records for the authenticated user, including flock names
func (h *InventoryHandler) GetInventory(c *gin.Context) {
	

	// Fetch user ID from the context
	userIDVal, exists := c.Get("user_id")
	if !exists {
		log.Println("GetInventory: User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		log.Println("GetInventory: Invalid user ID")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Fetch user details from the database using GetUserByID
	user, err := models.GetUserByID(userID)
	if err != nil {
		log.Println("GetInventory: User not found")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	var items []models.InventoryItem
	// Fetch inventory items and preload the Flock data (to get the flock name)
	if err := db.DB.Preload("Flock").Where("user_id = ?", user.ID).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve inventory items"})
		return
	}

	// Include flock name in the response
	for i := range items {
		// Ensure Flock is loaded and we can access the name
		if items[i].Flock != nil {
			// Optionally, you can add the flock name directly to the response if needed
			items[i].FlockName = items[i].Flock.Name
		}
	}

	c.JSON(http.StatusOK, items)
}

// AddInventoryItem adds a new inventory item for the authenticated user
func (h *InventoryHandler) AddInventoryItem(c *gin.Context) {
	// Fetch user ID from the context
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Fetch user details from the database using GetUserByID
	user, err := models.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	var item models.InventoryItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item.UserID = user.ID // Use user.ID as uint

	if err := db.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create inventory item"})
		return
	}

	c.JSON(http.StatusCreated, item)
}

// UpdateInventoryItem updates an existing inventory item for the authenticated user
func (h *InventoryHandler) UpdateInventoryItem(c *gin.Context) {
	// Fetch user ID from the context
	userIDVal, exists := c.Get("user_id")
	if !exists {
		log.Println("UpdateInventoryItem: User not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		log.Println("UpdateInventoryItem: Invalid user ID")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Fetch user details from the database using GetUserByID
	user, err := models.GetUserByID(userID)
	if err != nil {
		log.Println("UpdateInventoryItem: User not found")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	id := c.Param("id")

	var item models.InventoryItem
	// Convert user.ID and item ID to uint for the query
	if err := db.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Inventory item not found or unauthorized"})
		return
	}

	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := db.DB.Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update inventory item"})
		return
	}

	c.JSON(http.StatusOK, item)
}

// DeleteInventoryItem deletes an inventory item for the authenticated user
func (h *InventoryHandler) DeleteInventoryItem(c *gin.Context) {
	// Fetch user ID from the context
	userIDVal, exists := c.Get("user_id")
	if !exists {
		log.Println("DeleteInventoryItem: User not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		log.Println("DeleteInventoryItem: Invalid user ID")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Fetch user details from the database using GetUserByID
	user, err := models.GetUserByID(userID)
	if err != nil {
		log.Println("DeleteInventoryItem: User not found")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	id := c.Param("id")

	var item models.InventoryItem
	// Convert user.ID and item ID to uint for the query
	if err := db.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Inventory item not found or unauthorized"})
		return
	}

	if err := db.DB.Delete(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete inventory item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Inventory item deleted successfully"})
}
