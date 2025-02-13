package services

import (
	"errors"
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

// AddEggProduction adds a new egg production record, calculates revenue, and sends a WebSocket update
func (s *EggProductionService) AddEggProduction(record *models.EggProduction) error {
	record.TotalRevenue = float64(record.EggsCollected) * record.PricePerUnit
	if err := s.DB.Create(record).Error; err != nil {
		return err
	}
	broadcast.SendEggProductionUpdate("egg_production_added", *record)
	return nil
}

// UpdateEggProduction updates an existing egg production record, recalculates revenue, and sends a WebSocket update
func (s *EggProductionService) UpdateEggProduction(record *models.EggProduction) error {
	record.TotalRevenue = float64(record.EggsCollected) * record.PricePerUnit
	if err := s.DB.Save(record).Error; err != nil {
		return err
	}
	broadcast.SendEggProductionUpdate("egg_production_updated", *record)
	return nil
}

// DeleteEggProduction removes an egg production record by ID and sends a WebSocket update
func (s *EggProductionService) DeleteEggProduction(recordID uint, userID uint) error {
	var record models.EggProduction
	if err := s.DB.Where("id = ? AND user_id = ?", recordID, userID).First(&record).Error; err != nil {
		return errors.New("record not found")
	}

	if err := s.DB.Delete(&record).Error; err != nil {
		return err
	}

	broadcast.SendEggProductionUpdate("egg_production_deleted", recordID)
	return nil
}
