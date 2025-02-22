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

// GetFinancialDataByUser retrieves financial records for a specific user
func (s *FlockFinancialService) GetFinancialDataByUser(userID uint) ([]models.FlocksFinancialData, error) {
	var financialData []models.FlocksFinancialData
	err := s.DB.Where("user_id = ?", userID).Find(&financialData).Error
	return financialData, err
}

// GetFinancialDataByFlock retrieves financial data for a specific flock
func (s *FlockFinancialService) GetFinancialDataByFlock(flockID uint, userID uint) (*models.FlocksFinancialData, error) {
	var financialData models.FlocksFinancialData
	err := s.DB.Where("flock_id = ? AND user_id = ?", flockID, userID).First(&financialData).Error
	if err != nil {
		return nil, err
	}
	return &financialData, nil
}

// AddOrUpdateFinancialData adds or updates financial data for a flock
func (s *FlockFinancialService) AddOrUpdateFinancialData(financialData *models.FlocksFinancialData) error {
	var existing models.FlocksFinancialData
	err := s.DB.Where("flock_id = ? AND user_id = ? AND month = ? AND year = ?", 
		financialData.FlockID, financialData.UserID, financialData.Month, financialData.Year).
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
func (s *FlockFinancialService) DeleteFinancialData(flockID uint, userID uint) error {
	return s.DB.Where("flock_id = ? AND user_id = ?", flockID, userID).Delete(&models.FlocksFinancialData{}).Error
}
