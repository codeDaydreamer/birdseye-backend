package services

import (
	"errors"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/broadcast"
	"gorm.io/gorm"
)

type FlockService struct {
	DB                  *gorm.DB
	EggProductionService *EggProductionService
	SalesService        *SalesService
}


func NewFlockService(db *gorm.DB, eggService *EggProductionService, salesService *SalesService) *FlockService {
	return &FlockService{
		DB:                  db,
		EggProductionService: eggService,
		SalesService:        salesService,
	}
}


// GetFlocksByUser retrieves flock records for a specific user
func (s *FlockService) GetFlocksByUser(userID uint) ([]models.Flock, error) {
	var flocks []models.Flock
	err := s.DB.Where("user_id = ?", userID).Find(&flocks).Error
	return flocks, err
}

// GetFlockByID retrieves a single flock by ID and user
func (s *FlockService) GetFlockByID(flockID, userID uint) (*models.Flock, error) {
	var flock models.Flock
	err := s.DB.Where("id = ? AND user_id = ?", flockID, userID).First(&flock).Error
	if err != nil {
		return nil, errors.New("flock not found")
	}
	return &flock, nil
}

// AddFlock adds a new flock, calculates required fields, and sends a WebSocket update
func (s *FlockService) AddFlock(flock *models.Flock) error {
	s.CalculateFlockMetrics(flock) // Compute before inserting
	if err := s.DB.Create(flock).Error; err != nil {
		return err
	}
	broadcast.SendFlockUpdate("flock_added", *flock)
	return nil
}

// UpdateFlock updates an existing flock, recalculates fields, and sends a WebSocket update
func (s *FlockService) UpdateFlock(flock *models.Flock) error {
	s.CalculateFlockMetrics(flock) // Compute before updating
	if err := s.DB.Save(flock).Error; err != nil {
		return err
	}
	broadcast.SendFlockUpdate("flock_updated", *flock)
	return nil
}

// DeleteFlock removes a flock record by ID and sends a WebSocket update
func (s *FlockService) DeleteFlock(flockID, userID uint) error {
	var flock models.Flock
	if err := s.DB.Where("id = ? AND user_id = ?", flockID, userID).First(&flock).Error; err != nil {
		return errors.New("flock not found")
	}

	if err := s.DB.Delete(&flock).Error; err != nil {
		return err
	}

	broadcast.SendFlockUpdate("flock_deleted", flockID)
	return nil
}

// ------------------------ 🔹 Calculation Functions 🔹 ------------------------

// CalculateFlockMetrics computes all derived fields before saving to DB
func (s *FlockService) CalculateFlockMetrics(flock *models.Flock) {
	s.CalculateMortalityRate(flock)
	s.CalculateFeedIntake(flock)
	s.CalculateRevenueAndExpenses(flock)
	s.AggregateEggProduction(flock)
}

// CalculateMortalityRate updates the mortality rate based on historical data
func (s *FlockService) CalculateMortalityRate(flock *models.Flock) {
	totalDeaths := 0
	for _, deaths := range flock.MortalityRateData {
		totalDeaths += int(deaths)
	}
	if flock.BirdCount > 0 {
		flock.MortalityRate = (float64(totalDeaths) / float64(flock.BirdCount)) * 100
	} else {
		flock.MortalityRate = 0
	}
}

// CalculateFeedIntake estimates daily feed intake based on past consumption
func (s *FlockService) CalculateFeedIntake(flock *models.Flock) {
	totalFeed := 0
	days := len(flock.FeedConsumption)
	if days == 0 {
		flock.FeedIntake = 0
		return
	}

	for _, feed := range flock.FeedConsumption {
		totalFeed += feed
	}

	flock.FeedIntake = float64(totalFeed) / float64(days) // Average feed intake per day
}

// CalculateRevenueAndExpenses computes revenue from sales and subtracts expenses
func (s *FlockService) CalculateRevenueAndExpenses(flock *models.Flock) {
	totalRevenue := 0.0
	totalExpenses := flock.Expenses // Already stored expenses

	// Fetch related egg sales data
	eggSales, _ := s.EggProductionService.GetEggProductionByUser(flock.UserID)
	for _, sale := range eggSales {
		if sale.FlockID == flock.ID {
			totalRevenue += float64(sale.EggsCollected) * sale.PricePerUnit
		}
	}

	// Fetch related flock sales data
	sales, _ := s.SalesService.GetSalesByFlock(flock.ID)
	for _, sale := range sales {
		totalRevenue += sale.Amount
	}

	flock.Revenue = totalRevenue - totalExpenses
}
// AggregateEggProduction summarizes egg production for reports
func (s *FlockService) AggregateEggProduction(flock *models.Flock) {
	eggProductions, _ := s.EggProductionService.GetEggProductionByUser(flock.UserID)

	// Filter by flock
	filteredEggs := []int{}
	for _, egg := range eggProductions {
		if egg.FlockID == flock.ID {
			filteredEggs = append(filteredEggs, egg.EggsCollected)
		}
	}

	// Compute last 7 days
	last7Days := 7
	if len(filteredEggs) < 7 {
		last7Days = len(filteredEggs)
	}
	flock.EggProduction7Days = filteredEggs[len(filteredEggs)-last7Days:]

	// Compute last 4 weeks (28 days)
	last28Days := 28
	if len(filteredEggs) < 28 {
		last28Days = len(filteredEggs)
	}
	flock.EggProduction4Weeks = filteredEggs[len(filteredEggs)-last28Days:]
}
