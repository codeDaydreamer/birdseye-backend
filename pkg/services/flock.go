package services

import (
	"errors"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/broadcast"
	"gorm.io/gorm"
	"fmt"

	"log"

	"time"
)

type FlockService struct {
	DB                  *gorm.DB
	EggProductionService *EggProductionService
	SalesService        *SalesService
	ExpenseService      *ExpenseService // ✅ Fixed Naming
}

// NewFlockService initializes a new service instance
func NewFlockService(db *gorm.DB, eggService *EggProductionService, salesService *SalesService, expenseService *ExpenseService) *FlockService {
	return &FlockService{
		DB:                  db,
		EggProductionService: eggService,
		SalesService:        salesService,
		ExpenseService:      expenseService, // ✅ Fixed Naming
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
func (s *FlockService) AddFlock(flock *models.Flock) error {
    fmt.Printf("Adding new flock: %+v\n", flock)

    // Define date range for financial calculations (current month)
    start := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
    end := start.AddDate(0, 1, -1) // Last day of the month

    // Compute initial metrics before saving
    s.CalculateFlockMetrics(flock, flock.UserID, start, end)
    fmt.Printf("Metrics after calculation: %+v\n", flock)

    // Save to database
    if err := s.DB.Create(flock).Error; err != nil {
        fmt.Printf("Error creating flock: %v\n", err)
        return err
    }

    fmt.Printf("Flock successfully added with ID %d\n", flock.ID)
    
    // Send real-time update with userID
    broadcast.SendFlockUpdate(flock.UserID, "flock_added", *flock)
    return nil
}

func (s *FlockService) UpdateFlock(flock *models.Flock) error {
    log.Printf("📢 UpdateFlock called for Flock ID: %d\n", flock.ID)

    // Define date range for financial calculations (current month)
    start := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
    end := start.AddDate(0, 1, -1) // Last day of the month

    s.CalculateFlockMetrics(flock, flock.UserID, start, end)

    if err := s.DB.Save(flock).Error; err != nil {
        log.Printf("❌ Error saving flock ID %d: %v\n", flock.ID, err)
        return err
    }
    
    // Send real-time update with userID
    broadcast.SendFlockUpdate(flock.UserID, "flock_updated", *flock)
    return nil
}

func (s *FlockService) DeleteFlock(flockID, userID uint) error {
    var flock models.Flock
    if err := s.DB.Where("id = ? AND user_id = ?", flockID, userID).First(&flock).Error; err != nil {
        return errors.New("flock not found")
    }

    if err := s.DB.Delete(&flock).Error; err != nil {
        return err
    }

    // Send real-time update with userID
    broadcast.SendFlockUpdate(userID, "flock_deleted", flockID)
    return nil
}


func (s *FlockService) CalculateFlockMetrics(flock *models.Flock, userID uint, start, end time.Time) {
    log.Printf("Calculating metrics for Flock ID %d...", flock.ID)

    s.CalculateMortalityRate(flock)
    s.CalculateRevenueAndExpenses(flock, userID, start, end) // ✅ Now includes date range
 

    log.Printf("Metrics before saving: %+v\n", flock)

    err := s.DB.Model(&models.Flock{}).Where("id = ?", flock.ID).
        Select("*").Updates(map[string]interface{}{
        "mortality_rate":         flock.MortalityRate,
        "feed_intake":            flock.FeedIntake,
        "revenue":                flock.Revenue,
        "expenses":               flock.Expenses,
        "health":                 flock.Health,
        
    }).Error

    if err != nil {
        log.Printf("❌ Error updating flock metrics for flock ID %d: %v\n", flock.ID, err)
    } else {
        log.Printf("✅ Successfully updated flock metrics for Flock ID %d\n", flock.ID)
    }
}




// 🔹 Mortality rate calculation (Stores in DB)
func (s *FlockService) CalculateMortalityRate(flock *models.Flock) {
	// Ensure InitialBirdCount is greater than zero to avoid division by zero
	if flock.InitialBirdCount > 0 {
		// Calculate total deaths using the difference between InitialBirdCount and current BirdCount
		totalDeaths := flock.InitialBirdCount - flock.BirdCount
		flock.MortalityRate = (float64(totalDeaths) / float64(flock.InitialBirdCount)) * 100
	} else {
		flock.MortalityRate = 0
	}

	fmt.Printf("Flock ID %d - Mortality Rate Calculated: %.2f%% (Initial Count: %d, Current Count: %d)\n",
		flock.ID, flock.MortalityRate, flock.InitialBirdCount, flock.BirdCount)

	// 🔥 Save updated mortality rate in the database
	err := s.DB.Model(&models.Flock{}).
		Where("id = ?", flock.ID).
		Update("mortality_rate", flock.MortalityRate).Error

	if err != nil {
		fmt.Printf("❌ Error updating mortality rate for flock ID %d: %v\n", flock.ID, err)
	} else {
		fmt.Printf("✅ Successfully stored mortality rate for flock ID %d\n", flock.ID)
	}
}




func (s *FlockService) CalculateRevenueAndExpenses(flock *models.Flock, userID uint, start, end time.Time) {
    var totalRevenue, totalEggSales, totalExpenses float64

    // Fetch sales within the specified date range
    sales, err := s.SalesService.GetSalesByFlockAndPeriod(flock.ID, userID, start, end)
    if err != nil {
        fmt.Println("Error fetching flock sales:", err)
        return
    }

    // Filter and sum revenue from egg sales
    for _, sale := range sales {
        if sale.Category == "Egg Sales" {
            totalEggSales += sale.Amount
        }
        totalRevenue += sale.Amount
    }

    // Fetch expenses within the specified date range
    expenses, err := s.ExpenseService.GetExpensesByFlockAndPeriod(flock.ID, userID, start, end)
    if err != nil {
        fmt.Println("Error fetching flock expenses:", err)
        return
    }

    // Sum up all expenses
    for _, expense := range expenses {
        totalExpenses += expense.Amount
    }

    // Compute net revenue (profit/loss)
    netRevenue := totalRevenue - totalExpenses

    // Store financial data in a separate table
	financialData := models.FlocksFinancialData{
		FlockID:    flock.ID,
		UserID:     userID,
		Month:      int(start.Month()),  // Convert time.Month to int
		Year:       start.Year(),
		Revenue:    totalRevenue,
		EggSales:   totalEggSales,
		Expenses:   totalExpenses,
		NetRevenue: netRevenue,
	}
	

    // Upsert into the database
    if err := s.DB.Where("flock_id = ? AND user_id = ? AND month = ? AND year = ?", flock.ID, userID, start.Month(), start.Year()).
        Assign(financialData).FirstOrCreate(&financialData).Error; err != nil {
        fmt.Println("Error updating flock financial data:", err)
    }

    fmt.Printf("Flock ID %d - Revenue: %.2f, Egg Sales: %.2f, Expenses: %.2f, Net Revenue: %.2f for %s %d\n",
        flock.ID, totalRevenue, totalEggSales, totalExpenses, netRevenue, start.Month().String(), start.Year())
}

