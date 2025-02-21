package services

import (
	"log"
	"time"
	"errors"
	"birdseye-backend/pkg/models"
	"gorm.io/gorm"
)

// FlockFinancialService handles flock financial calculations
type FlockFinancialService struct {
	db             *gorm.DB
	financeService *FinanceService // Injected instance
}

// NewFlockFinancialService initializes the service with dependencies
func NewFlockFinancialService(db *gorm.DB, financeService *FinanceService) *FlockFinancialService {
	return &FlockFinancialService{
		db:             db,
		financeService: financeService,
	}
}
// GetFlockFinancialData fetches financial data for the given user and period
func (s *FlockFinancialService) GetFlockFinancialData(userID uint, periodStart, periodEnd time.Time) ([]models.FlockFinancialData, error) {
	log.Printf("Triggering financial data calculation for user %d from %s to %s", userID, periodStart, periodEnd)

	// Ensure financeService is initialized
	if s.financeService == nil {
		log.Println("FinanceService is not initialized")
		return nil, errors.New("finance service is not available")
	}

	// Trigger financial data calculation before fetching
	_, err := s.financeService.getFinanceDataForPeriod(periodStart, periodEnd, userID)
	if err != nil {
		log.Printf("Error calculating financial data: %v", err)
		return nil, err
	}

	// Fetch financial data from the database
	var financialData []models.FlockFinancialData
	log.Printf("Fetching financial data for user %d from %s to %s", userID, periodStart, periodEnd)

	err = s.db.Model(&models.FlockFinancialData{}).
		Where("user_id = ? AND period_start BETWEEN ? AND ?", userID, periodStart, periodEnd).
		Find(&financialData).Error

	if err != nil {
		log.Printf("Error fetching flock financial data: %v", err)
		return nil, err
	}

	return financialData, nil
}
