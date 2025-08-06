package services

import (
	"errors"
	"fmt"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/broadcast"
	"gorm.io/gorm"
)

type EggAdjustmentService struct {
	DB *gorm.DB
}

// NewEggAdjustmentService initializes the service
func NewEggAdjustmentService(db *gorm.DB) *EggAdjustmentService {
	return &EggAdjustmentService{DB: db}
}

// GetAdjustmentsByUser retrieves all egg adjustments for a user
func (s *EggAdjustmentService) GetAdjustmentsByUser(userID uint) ([]models.EggAdjustment, error) {
	var adjustments []models.EggAdjustment
	err := s.DB.Where("user_id = ?", userID).Find(&adjustments).Error
	return adjustments, err
}

// AddAdjustment creates a new egg adjustment
func (s *EggAdjustmentService) AddAdjustment(adj *models.EggAdjustment) error {
	if err := s.DB.Create(adj).Error; err != nil {
		return err
	}

	// Broadcast update via WebSocket
	broadcast.SendEggAdjustmentUpdate(adj.UserID, "added", *adj)

	// Optional: Send notification
	notificationMessage := fmt.Sprintf("New egg adjustment: %s eggs (%d).", adj.Reason, adj.Quantity)
	broadcast.SendNotification(adj.UserID, "Egg Adjustment Added", notificationMessage, "/dashboard")

	return nil
}

// UpdateAdjustment updates an existing adjustment
func (s *EggAdjustmentService) UpdateAdjustment(adj *models.EggAdjustment) error {
	if err := s.DB.Save(adj).Error; err != nil {
		return err
	}

	broadcast.SendEggAdjustmentUpdate(adj.UserID, "updated", *adj)
	return nil
}

// DeleteAdjustment deletes an adjustment
func (s *EggAdjustmentService) DeleteAdjustment(id, userID uint) error {
	var adj models.EggAdjustment
	if err := s.DB.Where("id = ? AND user_id = ?", id, userID).First(&adj).Error; err != nil {
		return errors.New("adjustment not found")
	}

	if err := s.DB.Delete(&adj).Error; err != nil {
		return err
	}

	broadcast.SendEggAdjustmentUpdate(userID, "deleted", adj.ID)
	return nil
}
