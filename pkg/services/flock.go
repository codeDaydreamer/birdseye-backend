package services

import (
	"errors"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/broadcast"
	"gorm.io/gorm"
	"fmt"
	"sort"
	"log"
	"encoding/json"
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

    // Compute initial metrics before saving
    s.CalculateFlockMetrics(flock, flock.UserID) // ✅ Pass userID
    fmt.Printf("Metrics after calculation: %+v\n", flock)

    // Save to database
    if err := s.DB.Create(flock).Error; err != nil {
        fmt.Printf("Error creating flock: %v\n", err)
        return err
    }

    fmt.Printf("Flock successfully added with ID %d\n", flock.ID)
    
    // Send real-time update
    broadcast.SendFlockUpdate("flock_added", *flock)
    return nil
}



func (s *FlockService) UpdateFlock(flock *models.Flock) error {
    log.Printf("📢 UpdateFlock called for Flock ID: %d\n", flock.ID)
    
    s.CalculateFlockMetrics(flock, flock.UserID) // ✅ Pass userID

    if err := s.DB.Save(flock).Error; err != nil {
        log.Printf("❌ Error saving flock ID %d: %v\n", flock.ID, err)
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

func (s *FlockService) CalculateFlockMetrics(flock *models.Flock, userID uint) {
    log.Printf("Calculating metrics for Flock ID %d...", flock.ID)

    s.CalculateMortalityRate(flock)
    s.CalculateFeedIntake(flock)
    s.CalculateRevenueAndExpenses(flock, userID) // ✅ Now userID is passed correctly
    s.AggregateEggProduction(flock)

    log.Printf("Metrics before saving: %+v\n", flock)

    err := s.DB.Model(&models.Flock{}).Where("id = ?", flock.ID).
        Select("*").Updates(map[string]interface{}{
        "mortality_rate":         flock.MortalityRate,
        "feed_intake":            flock.FeedIntake,
        "revenue":                flock.Revenue,
        "expenses":               flock.Expenses,
        "health":                 flock.Health,
        "egg_production_7_days":  flock.EggProduction7Days,
        "egg_production_4_weeks": flock.EggProduction4Weeks,
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



// 🔹 Calculate daily feed intake (Stores in DB)
func (s *FlockService) CalculateFeedIntake(flock *models.Flock) {
	totalFeed := 0
	days := len(flock.FeedConsumption)

	if days > 0 {
		for _, feed := range flock.FeedConsumption {
			totalFeed += int(feed)
		}
		flock.FeedIntake = float64(totalFeed) / float64(days) // Average intake per day
	} else {
		flock.FeedIntake = 0
	}

	fmt.Printf("Flock ID %d - Feed Intake Calculated: %.2f (Total Feed: %d, Days: %d)\n",
		flock.ID, flock.FeedIntake, totalFeed, days)
}
func (s *FlockService) CalculateRevenueAndExpenses(flock *models.Flock, userID uint) {
    var totalRevenue, totalEggSales, totalExpenses float64

    // Fetch all sales related to the flock for the specific user
    sales, err := s.SalesService.GetSalesByFlock(flock.ID, userID)
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

    // Fetch all expenses related to the flock (without userID)
    expenses, err := s.ExpenseService.GetExpensesByFlock(flock.ID)
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

    // Update the flock record
    flock.Revenue = netRevenue
    flock.Expenses = totalExpenses

    // Save updates to the database
    if err := s.DB.Save(&flock).Error; err != nil {
        fmt.Println("Error updating flock revenue and expenses:", err)
    }

    fmt.Printf("Flock ID %d - Revenue: %.2f, Egg Sales: %.2f, Expenses: %.2f, Net Revenue: %.2f\n",
        flock.ID, totalRevenue, totalEggSales, totalExpenses, netRevenue)
}

// 🔹 Aggregate Egg Production for 7 Days and 4 Weeks (Stores in DB)
func (s *FlockService) AggregateEggProduction(flock *models.Flock) {
	eggProductions, err := s.EggProductionService.GetEggProductionByUser(flock.UserID)
	if err != nil {
		fmt.Println("❌ Error fetching egg production data:", err)
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

	// Ensure eggCounts is not nil (though in Go, slices are never nil after initialization)
	if eggCounts == nil {
		eggCounts = []int{}
	}

	// Compute last 7 days
	var jsonData7 []byte
	if len(eggCounts) >= 7 {
		jsonData7, err = json.Marshal(eggCounts[:7])
	} else {
		jsonData7, err = json.Marshal(eggCounts)
	}
	if err != nil {
		fmt.Printf("❌ Error marshaling egg production (7 days) for Flock ID %d: %v\n", flock.ID, err)
		jsonData7 = []byte("[]") // Fallback to an empty array
	}
	flock.EggProduction7Days = jsonData7

	// Compute last 4 weeks (28 days)
	var jsonData28 []byte
	if len(eggCounts) >= 28 {
		jsonData28, err = json.Marshal(eggCounts[:28])
	} else {
		jsonData28, err = json.Marshal(eggCounts)
	}
	if err != nil {
		fmt.Printf("❌ Error marshaling egg production (28 days) for Flock ID %d: %v\n", flock.ID, err)
		jsonData28 = []byte("[]") // Fallback to an empty array
	}
	flock.EggProduction4Weeks = jsonData28

	fmt.Printf("✅ Egg production data updated for Flock ID %d\n", flock.ID)
}
