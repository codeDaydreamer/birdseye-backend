package services

import (
	"birdseye-backend/pkg/models"
	"errors"
	"gorm.io/gorm"
)

// FlockFinancialService manages flock financial data
type FlockFinancialService struct {
	DB *gorm.DB
}

// NewFlockFinancialService initializes a new service instance
func NewFlockFinancialService(db *gorm.DB) *FlockFinancialService {
	return &FlockFinancialService{DB: db}
}

// GetFinancialDataByFarm retrieves financial records for a specific farm
func (s *FlockFinancialService) GetFinancialDataByFarm(farmID uint) ([]models.FlocksFinancialData, error) {
	var financialData []models.FlocksFinancialData
	err := s.DB.Where("farm_id = ?", farmID).Find(&financialData).Error
	return financialData, err
}

// GetFinancialDataByFlock retrieves financial data for a specific flock
func (s *FlockFinancialService) GetFinancialDataByFlock(flockID uint, farmID uint) (*models.FlocksFinancialData, error) {
	var financialData models.FlocksFinancialData
	err := s.DB.Where("flock_id = ? AND farm_id = ?", flockID, farmID).First(&financialData).Error
	if err != nil {
		return nil, err
	}
	return &financialData, nil
}

// AddOrUpdateFinancialData adds or updates financial data for a flock
func (s *FlockFinancialService) AddOrUpdateFinancialData(financialData *models.FlocksFinancialData) error {
	var existing models.FlocksFinancialData
	err := s.DB.Where("flock_id = ? AND farm_id = ? AND month = ? AND year = ?", 
		financialData.FlockID, financialData.FarmID, financialData.Month, financialData.Year).
		First(&existing).Error
	
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.DB.Create(financialData).Error
	}

	// Update existing record
	existing.Revenue = financialData.Revenue
	existing.EggSales = financialData.EggSales
	existing.Expenses = financialData.Expenses
	existing.NetRevenue = financialData.NetRevenue

	return s.DB.Save(&existing).Error
}

// DeleteFinancialData removes financial data for a flock
func (s *FlockFinancialService) DeleteFinancialData(flockID uint, farmID uint) error {
	return s.DB.Where("flock_id = ? AND farm_id = ?", flockID, farmID).Delete(&models.FlocksFinancialData{}).Error
}
