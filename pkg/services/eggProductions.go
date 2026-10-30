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

// GetEggProductionByFarm retrieves egg production records for a specific farm
func (s *EggProductionService) GetEggProductionByFarm(farmID uint) ([]models.EggProduction, error) {
	var records []models.EggProduction
	err := s.DB.Where("farm_id = ?", farmID).Find(&records).Error
	return records, err
}

// AddEggProduction adds a new egg production record, calculates revenue, and sends a WebSocket update and notification
func (s *EggProductionService) AddEggProduction(record *models.EggProduction) error {
	record.TotalRevenue = float64(record.EggsCollected) * record.PricePerUnit
	if err := s.DB.Create(record).Error; err != nil {
		return err
	}

	// Send WebSocket update
	broadcast.SendEggProductionUpdate(record.FarmID, "egg_production_added", *record)

	// Send notification
	notificationMessage := fmt.Sprintf("New egg production record added: %d eggs collected.", record.EggsCollected)
	broadcast.SendFarmNotification(record.FarmID, "Egg Production Added", notificationMessage, "/dashboard")

	return nil
}

// UpdateEggProduction updates an existing egg production record, recalculates revenue, and sends a WebSocket update and notification
func (s *EggProductionService) UpdateEggProduction(record *models.EggProduction) error {
	record.TotalRevenue = float64(record.EggsCollected) * record.PricePerUnit
	if err := s.DB.Save(record).Error; err != nil {
		return err
	}

	// Send WebSocket update
	broadcast.SendEggProductionUpdate(record.FarmID, "egg_production_updated", *record)

	// Send notification
	notificationMessage := fmt.Sprintf("Egg production record updated: %d eggs collected.", record.EggsCollected)
	broadcast.SendFarmNotification(record.FarmID, "Egg Production Updated", notificationMessage, "/dashboard")

	return nil
}

// DeleteEggProduction removes an egg production record by ID and sends a WebSocket update and notification
func (s *EggProductionService) DeleteEggProduction(recordID uint, farmID uint) error {
	var record models.EggProduction
	if err := s.DB.Where("id = ? AND farm_id = ?", recordID, farmID).First(&record).Error; err != nil {
		return errors.New("record not found")
	}

	if err := s.DB.Delete(&record).Error; err != nil {
		return err
	}

	// Send WebSocket update
	broadcast.SendEggProductionUpdate(farmID, "egg_production_deleted", recordID)

	// Send notification
	notificationMessage := "An egg production record was deleted."
	broadcast.SendFarmNotification(farmID, "Egg Production Deleted", notificationMessage, "/dashboard")

	return nil
}
