package services

import (
	"errors"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/broadcast"
	"gorm.io/gorm"
	"fmt"
)

// InventoryService provides methods to manage inventory items
type InventoryService struct {
	DB *gorm.DB
}

// NewInventoryService initializes a new service instance
func NewInventoryService(db *gorm.DB) *InventoryService {
	return &InventoryService{DB: db}
}

// GetInventoryByUser retrieves inventory items for a specific user
func (s *InventoryService) GetInventoryByUser(userID uint) ([]models.InventoryItem, error) {
	var items []models.InventoryItem
	err := s.DB.Where("user_id = ?", userID).Find(&items).Error
	return items, err
}

/// AddInventoryItem adds a new inventory item, sends a WebSocket update, and notifies the user
func (s *InventoryService) AddInventoryItem(item *models.InventoryItem) error {
	if err := s.DB.Create(item).Error; err != nil {
		return err
	}

	// Send real-time update with userID
	broadcast.SendInventoryUpdate(item.UserID, "inventory_added", *item)

	// Send notification to the user
	broadcast.SendNotification(item.UserID, "New Inventory Item Added", 
		fmt.Sprintf("Item '%s' has been added to your inventory.", item.ItemName), 
		"/inventory")

	return nil
}

// UpdateInventoryItem updates an existing inventory item, sends a WebSocket update, and notifies the user
func (s *InventoryService) UpdateInventoryItem(item *models.InventoryItem) error {
	if err := s.DB.Save(item).Error; err != nil {
		return err
	}

	// Send real-time update with userID
	broadcast.SendInventoryUpdate(item.UserID, "inventory_updated", *item)

	// Send notification to the user
	broadcast.SendNotification(item.UserID, "Inventory Item Updated", 
		fmt.Sprintf("Item '%s' has been updated in your inventory.", item.ItemName), 
		"/inventory")

	return nil
}

// DeleteInventoryItem removes an inventory item by ID, sends a WebSocket update, and notifies the user
func (s *InventoryService) DeleteInventoryItem(itemID uint, userID uint) error {
	var item models.InventoryItem
	if err := s.DB.Where("id = ? AND user_id = ?", itemID, userID).First(&item).Error; err != nil {
		return errors.New("inventory item not found")
	}

	if err := s.DB.Delete(&item).Error; err != nil {
		return err
	}

	// Send real-time update with userID
	broadcast.SendInventoryUpdate(userID, "inventory_deleted", itemID)

	// Send notification to the user
	broadcast.SendNotification(userID, "Inventory Item Deleted", 
		fmt.Sprintf("Item '%s' has been removed from your inventory.", item.ItemName), 
		"/inventory")

	return nil
}
