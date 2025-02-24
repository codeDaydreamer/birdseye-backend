package services

import (
	"errors"
	"fmt"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/broadcast"
	"gorm.io/gorm"
)

// EggProductionService provides methods to manage egg production records
type EggProductionService struct {
	DB *gorm.DB
}

// NewEggProductionService initializes a new service instance
func NewEggProductionService(db *gorm.DB) *EggProductionService {
	return &EggProductionService{DB: db}
}

// GetEggProductionByUser retrieves egg production records for a specific user
func (s *EggProductionService) GetEggProductionByUser(userID uint) ([]models.EggProduction, error) {
	var records []models.EggProduction
	err := s.DB.Where("user_id = ?", userID).Find(&records).Error
	return records, err
}

// AddEggProduction adds a new egg production record, calculates revenue, and sends a WebSocket update and notification
func (s *EggProductionService) AddEggProduction(record *models.EggProduction) error {
	record.TotalRevenue = float64(record.EggsCollected) * record.PricePerUnit
	if err := s.DB.Create(record).Error; err != nil {
		return err
	}

	// Send WebSocket update
	broadcast.SendEggProductionUpdate(record.UserID, "egg_production_added", *record)

	// Send notification
	notificationMessage := fmt.Sprintf("New egg production record added: %d eggs collected.", record.EggsCollected)
	broadcast.SendNotification(record.UserID, "Egg Production Added", notificationMessage, "/dashboard")

	return nil
}

// UpdateEggProduction updates an existing egg production record, recalculates revenue, and sends a WebSocket update and notification
func (s *EggProductionService) UpdateEggProduction(record *models.EggProduction) error {
	record.TotalRevenue = float64(record.EggsCollected) * record.PricePerUnit
	if err := s.DB.Save(record).Error; err != nil {
		return err
	}

	// Send WebSocket update
	broadcast.SendEggProductionUpdate(record.UserID, "egg_production_updated", *record)

	// Send notification
	notificationMessage := fmt.Sprintf("Egg production record updated: %d eggs collected.", record.EggsCollected)
	broadcast.SendNotification(record.UserID, "Egg Production Updated", notificationMessage, "/dashboard")

	return nil
}

// DeleteEggProduction removes an egg production record by ID and sends a WebSocket update and notification
func (s *EggProductionService) DeleteEggProduction(recordID uint, userID uint) error {
	var record models.EggProduction
	if err := s.DB.Where("id = ? AND user_id = ?", recordID, userID).First(&record).Error; err != nil {
		return errors.New("record not found")
	}

	if err := s.DB.Delete(&record).Error; err != nil {
		return err
	}

	// Send WebSocket update
	broadcast.SendEggProductionUpdate(userID, "egg_production_deleted", recordID)

	// Send notification
	notificationMessage := "An egg production record was deleted."
	broadcast.SendNotification(userID, "Egg Production Deleted", notificationMessage, "/dashboard")

	return nil
}
