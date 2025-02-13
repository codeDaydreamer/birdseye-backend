package services

import (
	"errors"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/broadcast"
	"gorm.io/gorm"
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

// AddInventoryItem adds a new inventory item and sends a WebSocket update
func (s *InventoryService) AddInventoryItem(item *models.InventoryItem) error {
	if err := s.DB.Create(item).Error; err != nil {
		return err
	}
	broadcast.SendInventoryUpdate("inventory_added", *item)
	return nil
}

// UpdateInventoryItem updates an existing inventory item and sends a WebSocket update
func (s *InventoryService) UpdateInventoryItem(item *models.InventoryItem) error {
	if err := s.DB.Save(item).Error; err != nil {
		return err
	}
	broadcast.SendInventoryUpdate("inventory_updated", *item)
	return nil
}

// DeleteInventoryItem removes an inventory item by ID and sends a WebSocket update
func (s *InventoryService) DeleteInventoryItem(itemID uint, userID uint) error {
	var item models.InventoryItem
	if err := s.DB.Where("id = ? AND user_id = ?", itemID, userID).First(&item).Error; err != nil {
		return errors.New("inventory item not found")
	}

	if err := s.DB.Delete(&item).Error; err != nil {
		return err
	}

	broadcast.SendInventoryUpdate("inventory_deleted", itemID)
	return nil
}
