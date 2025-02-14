package services

import (
	"errors"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/broadcast"
	"gorm.io/gorm"
	"fmt"
	"sort"
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
	s.CalculateFlockMetrics(flock)
	if err := s.DB.Create(flock).Error; err != nil {
		return err
	}
	broadcast.SendFlockUpdate("flock_added", *flock)
	return nil
}

// UpdateFlock updates an existing flock, recalculates fields, and sends a WebSocket update
func (s *FlockService) UpdateFlock(flock *models.Flock) error {
	s.CalculateFlockMetrics(flock)
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
// CalculateFlockMetrics computes all derived fields and saves them to the DB
func (s *FlockService) CalculateFlockMetrics(flock *models.Flock) {
	fmt.Printf("Calculating metrics for Flock ID %d...\n", flock.ID)

	s.CalculateMortalityRate(flock)
	s.CalculateFeedIntake(flock)
	s.CalculateRevenueAndExpenses(flock)
	s.AggregateEggProduction(flock)
	s.CalculateHealth(flock)

	// Persist updated fields to the database
	err := s.DB.Model(&flock).Updates(map[string]interface{}{
		"mortality_rate":         flock.MortalityRate,
		"feed_intake":            flock.FeedIntake,
		"revenue":                flock.Revenue,
		"expenses":               flock.Expenses,
		"health":                 flock.Health,
		"egg_production_7_days":  flock.EggProduction7Days,
		"egg_production_4_weeks": flock.EggProduction4Weeks,
	}).Error

	if err != nil {
		fmt.Printf("Error updating flock metrics for flock ID %d: %v\n", flock.ID, err)
	} else {
		fmt.Printf("Successfully updated flock metrics for Flock ID %d\n", flock.ID)
	}
}

// 🔹 Mortality rate calculation (Stores in DB)
func (s *FlockService) CalculateMortalityRate(flock *models.Flock) {
	totalDeaths := 0
	for _, deaths := range flock.MortalityRateData {
		totalDeaths += deaths
	}

	if flock.InitialBirdCount > 0 {
		flock.MortalityRate = (float64(totalDeaths) / float64(flock.InitialBirdCount)) * 100
	} else {
		flock.MortalityRate = 0
	}

	fmt.Printf("Flock ID %d - Mortality Rate Calculated: %.2f%% (Total Deaths: %d, Initial Count: %d)\n",
		flock.ID, flock.MortalityRate, totalDeaths, flock.InitialBirdCount)
}

// 🔹 Calculate daily feed intake (Stores in DB)
func (s *FlockService) CalculateFeedIntake(flock *models.Flock) {
	totalFeed := 0
	days := len(flock.FeedConsumption)

	if days > 0 {
		for _, feed := range flock.FeedConsumption {
			totalFeed += feed
		}
		flock.FeedIntake = float64(totalFeed) / float64(days) // Average intake per day
	} else {
		flock.FeedIntake = 0
	}

	fmt.Printf("Flock ID %d - Feed Intake Calculated: %.2f (Total Feed: %d, Days: %d)\n",
		flock.ID, flock.FeedIntake, totalFeed, days)
}

// 🔹 Compute revenue and expenses (Stores in DB)
func (s *FlockService) CalculateRevenueAndExpenses(flock *models.Flock) {
	totalRevenue := 0.0
	totalExpenses := flock.Expenses

	// Fetch related egg sales data
	eggSales, err := s.EggProductionService.GetEggProductionByUser(flock.UserID)
	if err != nil {
		fmt.Println("Error fetching egg sales:", err)
		return
	}

	for _, sale := range eggSales {
		if sale.FlockID == flock.ID {
			totalRevenue += float64(sale.EggsCollected) * sale.PricePerUnit
		}
	}

	// Fetch related flock sales data
	sales, err := s.SalesService.GetSalesByFlock(flock.ID)
	if err != nil {
		fmt.Println("Error fetching flock sales:", err)
		return
	}

	for _, sale := range sales {
		totalRevenue += sale.Amount
	}

	flock.Revenue = totalRevenue - totalExpenses

	fmt.Printf("Flock ID %d - Revenue & Expenses Calculated: Revenue: %.2f, Expenses: %.2f, Net Revenue: %.2f\n",
		flock.ID, totalRevenue, totalExpenses, flock.Revenue)
}

// 🔹 Aggregate Egg Production for 7 Days and 4 Weeks (Stores in DB)
func (s *FlockService) AggregateEggProduction(flock *models.Flock) {
	eggProductions, err := s.EggProductionService.GetEggProductionByUser(flock.UserID)
	if err != nil {
		fmt.Println("Error fetching egg production data:", err)
		return
	}

	// Filter by flock and sort by date
	filteredEggs := []models.EggProduction{}
	for _, egg := range eggProductions {
		if egg.FlockID == flock.ID {
			filteredEggs = append(filteredEggs, egg)
		}
	}

	sort.Slice(filteredEggs, func(i, j int) bool {
		return filteredEggs[i].DateProduced.After(filteredEggs[j].DateProduced)
	})

	eggCounts := []int{}
	for _, egg := range filteredEggs {
		eggCounts = append(eggCounts, egg.EggsCollected)
	}

	// Compute last 7 days
	if len(eggCounts) >= 7 {
		flock.EggProduction7Days = eggCounts[:7]
	} else {
		flock.EggProduction7Days = eggCounts
	}

	// Compute last 4 weeks (28 days)
	if len(eggCounts) >= 28 {
		flock.EggProduction4Weeks = eggCounts[:28]
	} else {
		flock.EggProduction4Weeks = eggCounts
	}

	fmt.Printf("Flock ID %d - Egg Production Calculated: Last 7 Days: %v, Last 4 Weeks: %v\n",
		flock.ID, flock.EggProduction7Days, flock.EggProduction4Weeks)
}

// 🔹 Health Calculation Based on Mortality and Feed Intake (Stores in DB)
func (s *FlockService) CalculateHealth(flock *models.Flock) {
	health := 100.0

	// Mortality impact
	health -= flock.MortalityRate * 2 

	// Feed intake impact
	averageIntake := 100.0 
	if flock.FeedIntake < averageIntake {
		health -= (averageIntake - flock.FeedIntake) * 0.5
	}

	if health < 0 {
		health = 0
	} else if health > 100 {
		health = 100
	}

	flock.Health = health

	fmt.Printf("Flock ID %d - Health Calculated: %.2f (Mortality Rate: %.2f, Feed Intake: %.2f)\n",
		flock.ID, flock.Health, flock.MortalityRate, flock.FeedIntake)
}
