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

// GetInventoryByFarm retrieves inventory items for a specific farm
func (s *InventoryService) GetInventoryByFarm(farmID uint) ([]models.InventoryItem, error) {
	var items []models.InventoryItem
	err := s.DB.Where("farm_id = ?", farmID).Find(&items).Error
	return items, err
}

/// AddInventoryItem adds a new inventory item, sends a WebSocket update, and notifies the user
func (s *InventoryService) AddInventoryItem(item *models.InventoryItem) error {
	if err := s.DB.Create(item).Error; err != nil {
		return err
	}

	// Send real-time update to the farm
	broadcast.SendInventoryUpdate(item.FarmID, "inventory_added", *item)

	// Send notification to the farm
	broadcast.SendFarmNotification(item.FarmID, "New Inventory Item Added", 
		fmt.Sprintf("Item '%s' has been added to your inventory.", item.ItemName), 
		"/inventory")

	return nil
}

// UpdateInventoryItem updates an existing inventory item, sends a WebSocket update, and notifies the user
func (s *InventoryService) UpdateInventoryItem(item *models.InventoryItem) error {
	if err := s.DB.Save(item).Error; err != nil {
		return err
	}

	// Send real-time update to the farm
	broadcast.SendInventoryUpdate(item.FarmID, "inventory_updated", *item)

	// Send notification to the farm
	broadcast.SendFarmNotification(item.FarmID, "Inventory Item Updated", 
		fmt.Sprintf("Item '%s' has been updated in your inventory.", item.ItemName), 
		"/inventory")

	return nil
}

// DeleteInventoryItem removes an inventory item by ID, sends a WebSocket update, and notifies the user
func (s *InventoryService) DeleteInventoryItem(itemID uint, farmID uint) error {
	var item models.InventoryItem
	if err := s.DB.Where("id = ? AND farm_id = ?", itemID, farmID).First(&item).Error; err != nil {
		return errors.New("inventory item not found")
	}

	if err := s.DB.Delete(&item).Error; err != nil {
		return err
	}

	// Send real-time update to the farm
	broadcast.SendInventoryUpdate(farmID, "inventory_deleted", itemID)

	// Send notification to the farm
	broadcast.SendFarmNotification(farmID, "Inventory Item Deleted", 
		fmt.Sprintf("Item '%s' has been removed from your inventory.", item.ItemName), 
		"/inventory")

	return nil
}
