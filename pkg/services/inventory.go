package services

import (
    "birdseye-backend/pkg/db"
    "birdseye-backend/pkg/models"
)

// FetchAllItems retrieves all inventory items for a specific user
func FetchAllItems(userID uint) ([]models.InventoryItem, error) {
    var items []models.InventoryItem
    result := db.DB.Where("user_id = ?", userID).Find(&items) // Get items for the authenticated user
    return items, result.Error
}

// AddItem adds a new inventory item for a specific user
func AddItem(userID uint, item *models.InventoryItem) (*models.InventoryItem, error) {
    item.UserID = userID // Associate the item with the authenticated user
    result := db.DB.Create(item) // Insert the new item into the database
    if result.Error != nil {
        return nil, result.Error
    }
    return item, nil // Return the created item with the generated ID
}

// EditItem updates an existing inventory item for a specific user
func EditItem(userID uint, item *models.InventoryItem) (*models.InventoryItem, error) {
    var existingItem models.InventoryItem
    result := db.DB.Where("user_id = ? AND id = ?", userID, item.ID).First(&existingItem)
    if result.Error != nil {
        return nil, result.Error // Item not found or error
    }
    // Update fields with the new values
    existingItem.ItemName = item.ItemName
    existingItem.Quantity = item.Quantity
    existingItem.ReorderLevel = item.ReorderLevel
    existingItem.CostPerUnit = item.CostPerUnit

    db.DB.Save(&existingItem) // Save the updated item
    return &existingItem, nil
}

// DeleteItem deletes an inventory item for a specific user
func DeleteItem(userID uint, itemID uint) error {
    result := db.DB.Where("user_id = ? AND id = ?", userID, itemID).Delete(&models.InventoryItem{}) // Delete item by user and item ID
    return result.Error
}

// UpdateItemQuantity updates the quantity of an inventory item for a specific user
func UpdateItemQuantity(userID uint, itemID uint, quantity int) (*models.InventoryItem, error) {
    var item models.InventoryItem
    result := db.DB.Where("user_id = ? AND id = ?", userID, itemID).First(&item) // Fetch the item for the user
    if result.Error != nil {
        return nil, result.Error // Item not found or error
    }
    item.Quantity += quantity // Adjust the quantity
    if item.Quantity < 0 {
        item.Quantity = 0 // Ensure quantity doesn't go negative
    }
    db.DB.Save(&item) // Save the updated item
    return &item, nil
}

// FetchItemByID retrieves a specific inventory item by ID for a given user
func FetchItemByID(userID uint, itemID uint) (*models.InventoryItem, error) {
    var item models.InventoryItem
    result := db.DB.Where("id = ? AND user_id = ?", itemID, userID).First(&item) // Fetch item by user and item ID
    if result.Error != nil {
        return nil, result.Error // Item not found or error
    }
    return &item, nil
}
