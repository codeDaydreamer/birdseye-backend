package api

import (
    "birdseye-backend/pkg/db"
    "birdseye-backend/pkg/models"
    "github.com/gin-gonic/gin"
    "net/http"
)

func SetupInventoryRoutes(router *gin.Engine) {
    inventoryGroup := router.Group("/inventory")
    inventoryGroup.GET("/", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"message": "Inventory route works"})
    })
    inventoryGroup.POST("/", addItem)
    inventoryGroup.PUT("/", editItem)
    inventoryGroup.DELETE("/", deleteItem)
    inventoryGroup.POST("/fetch", fetchItems)
    inventoryGroup.PUT("/quantity", updateItemQuantity)
}


// Add a new inventory item
func addItem(c *gin.Context) {
    var item models.InventoryItem
    if err := c.ShouldBindJSON(&item); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
        return
    }

    // Add the item to the database
    if err := db.DB.Create(&item).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add item"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"item": item})
}

// Edit an existing inventory item
func editItem(c *gin.Context) {
    var updatedItem models.InventoryItem
    if err := c.ShouldBindJSON(&updatedItem); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
        return
    }

    if err := db.DB.Save(&updatedItem).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to edit item"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"item": updatedItem})
}

// Delete an inventory item
func deleteItem(c *gin.Context) {
    var itemToDelete models.InventoryItem
    if err := c.ShouldBindJSON(&itemToDelete); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
        return
    }

    if err := db.DB.Delete(&itemToDelete).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete item"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Item deleted"})
}

// Fetch all inventory items for a user
func fetchItems(c *gin.Context) {
    var request struct {
        UserID uint `json:"user_id"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
        return
    }

    var items []models.InventoryItem
    if err := db.DB.Where("user_id = ?", request.UserID).Find(&items).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch items"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"items": items})
}

// Update item quantity
func updateItemQuantity(c *gin.Context) {
    var request struct {
        ID       uint `json:"id"`
        Quantity int  `json:"quantity"`
        UserID   uint `json:"user_id"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
        return
    }

    var item models.InventoryItem
    if err := db.DB.First(&item, request.ID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
        return
    }

    item.Quantity = request.Quantity
    if err := db.DB.Save(&item).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update quantity"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"item": item})
}
