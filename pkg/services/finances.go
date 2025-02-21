package services

import (
	"log"
	"time"
	"gorm.io/gorm"
	"birdseye-backend/pkg/models"
	"gorm.io/gorm/clause"
)

type FinanceService struct {
	db *gorm.DB
}

func NewFinanceService(db *gorm.DB) *FinanceService {
	return &FinanceService{db: db}
}

// GetTotalRevenue calculates total revenue from sales for the current period and user
func (s *FinanceService) GetTotalRevenue(startDate, endDate time.Time, userID uint) (map[uint]float64, error) {
	var totalRevenue []struct {
		FlockID uint    `json:"flock_id"`
		Amount  float64 `json:"amount"`
	}
	log.Printf("Getting total revenue for user %d from %s to %s", userID, startDate, endDate)

	err := s.db.Model(&models.Sale{}).
		Where("date BETWEEN ? AND ? AND flock_id IN (SELECT id FROM flocks WHERE user_id = ?)", startDate, endDate, userID).
		Select("flock_id, COALESCE(SUM(amount), 0) as amount").
		Group("flock_id").
		Scan(&totalRevenue).Error

	if err != nil {
		log.Printf("Error calculating total revenue: %v", err)
		return nil, err
	}

	revenueMap := make(map[uint]float64)
	for _, row := range totalRevenue {
		revenueMap[row.FlockID] = row.Amount
	}

	log.Printf("Total revenue: %v", revenueMap)
	return revenueMap, nil
}

// GetTotalExpenses calculates total expenses (including inventory) for the current period and user
func (s *FinanceService) GetTotalExpenses(startDate, endDate time.Time, userID uint) (map[uint]float64, error) {
	totalExpenses := make(map[uint]float64)
	log.Printf("Getting total expenses for user %d from %s to %s", userID, startDate, endDate)
	
	// Query total expenses
	var totalExpensesResult []struct {
		FlockID uint    `json:"flock_id"`
		Amount  float64 `json:"amount"`
	}
	err := s.db.Model(&models.Expense{}).
		Where("date BETWEEN ? AND ? AND flock_id IN (SELECT id FROM flocks WHERE user_id = ?)", startDate, endDate, userID).
		Select("flock_id, COALESCE(SUM(amount), 0) as amount").
		Group("flock_id").
		Scan(&totalExpensesResult).Error
	
	if err != nil {
		log.Printf("Error calculating total expenses: %v", err)
		return nil, err
	}

	// Convert the result into a map
	for _, row := range totalExpensesResult {
		totalExpenses[row.FlockID] = row.Amount
	}

	log.Printf("Total expenses (without inventory): %v", totalExpenses)

	// InventoryCostResult holds the result for the inventory cost calculation for each flock
type InventoryCostResult struct {
	FlockID            uint    `json:"flock_id"`
	TotalInventoryCost float64 `json:"total_inventory_cost"`
}


	// Include inventory costs as expenses
	var inventoryCosts []InventoryCostResult
	err = s.db.Model(&models.InventoryItem{}).
		Where("flock_id IN (SELECT id FROM flocks WHERE user_id = ?)", userID).
		Select("flock_id, COALESCE(SUM(quantity * cost_per_unit), 0) as total_inventory_cost").
		Group("flock_id").
		Scan(&inventoryCosts).Error

	if err != nil {
		log.Printf("Error calculating inventory costs: %v", err)
		return nil, err
	}

	// Convert inventoryCosts slice into a map for easy merging
	totalInventoryCost := make(map[uint]float64)
	for _, cost := range inventoryCosts {
		totalInventoryCost[cost.FlockID] = cost.TotalInventoryCost
	}

	log.Printf("Total inventory cost: %v", totalInventoryCost)

	// Combine expenses and inventory costs
	for flockID, inventoryCost := range totalInventoryCost {
		totalExpenses[flockID] += inventoryCost
	}

	return totalExpenses, nil
}

// GetNetProfit calculates total revenue - total expenses for each flock of a user
func (s *FinanceService) GetNetProfit(startDate, endDate time.Time, userID uint) (map[uint]float64, error) {
	revenue, err := s.GetTotalRevenue(startDate, endDate, userID)
	if err != nil {
		log.Printf("Error fetching total revenue: %v", err)
		return nil, err
	}

	expenses, err := s.GetTotalExpenses(startDate, endDate, userID)
	if err != nil {
		log.Printf("Error fetching total expenses: %v", err)
		return nil, err
	}

	netProfit := make(map[uint]float64)
	for flockID, flockRevenue := range revenue {
		flockExpenses, exists := expenses[flockID]
		if !exists {
			flockExpenses = 0
		}
		netProfit[flockID] = flockRevenue - flockExpenses
	}

	log.Printf("Net profit for each flock: %v", netProfit)
	return netProfit, nil
}

// GetProfitMargin calculates (Net Profit / Revenue) * 100 for each flock
func (s *FinanceService) GetProfitMargin(startDate, endDate time.Time, userID uint) (map[uint]float64, error) {
	netProfit, err := s.GetNetProfit(startDate, endDate, userID)
	if err != nil {
		log.Printf("Error fetching net profit: %v", err)
		return nil, err
	}

	revenue, err := s.GetTotalRevenue(startDate, endDate, userID)
	if err != nil {
		log.Printf("Error fetching total revenue: %v", err)
		return nil, err
	}

	profitMargin := make(map[uint]float64)
	for flockID := range netProfit {
		if revenue[flockID] != 0 {
			profitMargin[flockID] = (netProfit[flockID] / revenue[flockID]) * 100
		}
	}

	log.Printf("Profit margin for each flock: %v", profitMargin)
	return profitMargin, nil
}

// GetCostPerUnitSold calculates total expenses divided by total quantity sold for each flock
func (s *FinanceService) GetCostPerUnitSold(startDate, endDate time.Time, userID uint) (map[uint]float64, error) {
	totalExpenses, err := s.GetTotalExpenses(startDate, endDate, userID)
	if err != nil {
		log.Printf("Error fetching total expenses: %v", err)
		return nil, err
	}

	// Temporary struct to hold the query result
	var totalQuantityResults []struct {
		FlockID  uint `json:"flock_id"`
		Quantity int  `json:"quantity"`
	}

	// Query total quantity sold per flock
	err = s.db.Model(&models.Sale{}).
		Where("date BETWEEN ? AND ? AND flock_id IN (SELECT id FROM flocks WHERE user_id = ?)", startDate, endDate, userID).
		Select("flock_id, COALESCE(SUM(quantity), 0) as quantity").
		Group("flock_id").
		Scan(&totalQuantityResults).Error

	if err != nil {
		log.Printf("Error fetching total quantity sold: %v", err)
		return nil, err
	}

	// Convert the result into a map
	totalQuantity := make(map[uint]int)
	for _, row := range totalQuantityResults {
		totalQuantity[row.FlockID] = row.Quantity
	}

	// Compute cost per unit sold
	costPerUnit := make(map[uint]float64)
	for flockID, flockQuantity := range totalQuantity {
		if flockQuantity != 0 {
			costPerUnit[flockID] = totalExpenses[flockID] / float64(flockQuantity)
		}
	}

	log.Printf("Cost per unit sold for each flock: %v", costPerUnit)
	return costPerUnit, nil
}

// GetExpenseToRevenueRatio calculates (Total Expenses / Total Revenue) * 100 for each flock
func (s *FinanceService) GetExpenseToRevenueRatio(startDate, endDate time.Time, userID uint) (map[uint]float64, error) {
	expenses, err := s.GetTotalExpenses(startDate, endDate, userID)
	if err != nil {
		log.Printf("Error fetching total expenses: %v", err)
		return nil, err
	}

	revenue, err := s.GetTotalRevenue(startDate, endDate, userID)
	if err != nil {
		log.Printf("Error fetching total revenue: %v", err)
		return nil, err
	}

	expenseToRevenueRatio := make(map[uint]float64)
	for flockID := range expenses {
		if revenue[flockID] != 0 {
			expenseToRevenueRatio[flockID] = (expenses[flockID] / revenue[flockID]) * 100
		}
	}

	log.Printf("Expense to revenue ratio for each flock: %v", expenseToRevenueRatio)
	return expenseToRevenueRatio, nil
}

// GetTotalInventoryCost calculates total cost of inventory items for each flock
func (s *FinanceService) GetTotalInventoryCost(userID uint) (map[uint]float64, error) {
	log.Printf("Getting total inventory cost for user %d", userID)

	// Initialize the map to store the results
	totalInventoryCost := make(map[uint]float64)

	// Query the database
	rows, err := s.db.Raw(`
		SELECT flock_id, COALESCE(SUM(quantity * cost_per_unit), 0) 
		FROM inventory_items 
		WHERE flock_id IN (SELECT id FROM flocks WHERE user_id = ?) 
		GROUP BY flock_id`, userID).Rows()
	if err != nil {
		log.Printf("Error executing inventory cost query: %v", err)
		return nil, err
	}
	defer rows.Close()

	// Read the results
	var flockID uint
	var totalCost float64
	for rows.Next() {
		// Scan the results into variables
		if err := rows.Scan(&flockID, &totalCost); err != nil {
			log.Printf("Error scanning inventory cost row: %v", err)
			return nil, err
		}
		totalInventoryCost[flockID] = totalCost
	}

	log.Printf("Total inventory cost for each flock: %v", totalInventoryCost)
	return totalInventoryCost, nil
}



// Helper function to get financial data for the current period (e.g., current week, current month, current year)
func (s *FinanceService) getFinanceDataForPeriod(startDate, endDate time.Time, userID uint) (map[uint]map[string]float64, error) {
	log.Printf("Fetching financial data for user %d from %s to %s", userID, startDate, endDate)

	// Get total revenue
	totalRevenue, err := s.GetTotalRevenue(startDate, endDate, userID)
	if err != nil {
		log.Printf("Error fetching total revenue: %v", err)
		return nil, err
	}
	log.Printf("Total Revenue fetched: %v", totalRevenue)

	// Get total expenses
	totalExpenses, err := s.GetTotalExpenses(startDate, endDate, userID)
	if err != nil {
		log.Printf("Error fetching total expenses: %v", err)
		return nil, err
	}
	log.Printf("Total Expenses fetched: %v", totalExpenses)

	// Get net profit
	netProfit, err := s.GetNetProfit(startDate, endDate, userID)
	if err != nil {
		log.Printf("Error fetching net profit: %v", err)
		return nil, err
	}
	log.Printf("Net Profit fetched: %v", netProfit)

	// Get profit margin
	profitMargin, err := s.GetProfitMargin(startDate, endDate, userID)
	if err != nil {
		log.Printf("Error fetching profit margin: %v", err)
		return nil, err
	}
	log.Printf("Profit Margin fetched: %v", profitMargin)

	// Get cost per unit sold
	costPerUnit, err := s.GetCostPerUnitSold(startDate, endDate, userID)
	if err != nil {
		log.Printf("Error fetching cost per unit sold: %v", err)
		return nil, err
	}
	log.Printf("Cost Per Unit fetched: %v", costPerUnit)

	// Get expense to revenue ratio
	expenseRatio, err := s.GetExpenseToRevenueRatio(startDate, endDate, userID)
	if err != nil {
		log.Printf("Error fetching expense to revenue ratio: %v", err)
		return nil, err
	}
	log.Printf("Expense to Revenue Ratio fetched: %v", expenseRatio)

	// Prepare the result map for each flock
	result := make(map[uint]map[string]float64)
	for flockID := range totalRevenue {
		result[flockID] = map[string]float64{
			"totalRevenue":  totalRevenue[flockID],
			"totalExpenses": totalExpenses[flockID],
			"netProfit":     netProfit[flockID],
			"profitMargin":  profitMargin[flockID],
			"costPerUnit":   costPerUnit[flockID],
			"expenseRatio":  expenseRatio[flockID],
		}

		// Save financial data for each flock
		err := s.SaveFlockFinancialData(flockID, userID, startDate, endDate)
		if err != nil {
			log.Printf("Error saving financial data for flock %d: %v", flockID, err)
		} else {
			log.Printf("Successfully saved financial data for flock %d", flockID)
		}
	}

	log.Printf("Compiled and saved financial data for each flock: %v", result)
	return result, nil
}


// GetCurrentWeekFinanceData calculates financial data for the current week
func (s *FinanceService) GetCurrentWeekFinanceData(userID uint) (map[uint]map[string]float64, error) {
	// Get the current time
	now := time.Now()

	// Find the most recent Sunday (start of the week)
	startOfWeek := now.Truncate(24 * time.Hour)
	for startOfWeek.Weekday() != time.Sunday {
		startOfWeek = startOfWeek.AddDate(0, 0, -1) // Go back one day until Sunday
	}

	// Set the end of the week (Saturday at 23:59:59)
	endOfWeek := startOfWeek.AddDate(0, 0, 6).Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	log.Printf("Fetching weekly financial data for user %d from %s to %s", userID, startOfWeek, endOfWeek)

	// Fetch the financial data
	return s.getFinanceDataForPeriod(startOfWeek, endOfWeek, userID)
}



// GetCurrentMonthFinanceData calculates financial data for the current month
func (s *FinanceService) GetCurrentMonthFinanceData(userID uint) (map[uint]map[string]float64, error) {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)

	// Correct end of month: last second of the last day
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Second)

	log.Printf("Fetching monthly financial data for user %d from %s to %s", userID, startOfMonth, endOfMonth)
	return s.getFinanceDataForPeriod(startOfMonth, endOfMonth, userID)
}

// GetCurrentYearFinanceData calculates financial data for the current year
func (s *FinanceService) GetCurrentYearFinanceData(userID uint) (map[uint]map[string]float64, error) {
	now := time.Now()
	startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.Local)

	// Correct end of year: last second of December 31
	endOfYear := startOfYear.AddDate(1, 0, 0).Add(-time.Second)

	log.Printf("Fetching yearly financial data for user %d from %s to %s", userID, startOfYear, endOfYear)
	return s.getFinanceDataForPeriod(startOfYear, endOfYear, userID)
}

// SaveFlockFinancialData stores or updates calculated financial data for a flock in the database
func (s *FinanceService) SaveFlockFinancialData(flockID uint, userID uint, startDate, endDate time.Time) error {
    // Fetch calculated financial data for the flock within the specified time period
    totalRevenue, err := s.GetTotalRevenue(startDate, endDate, userID)
    if err != nil {
        return err
    }

    totalExpenses, err := s.GetTotalExpenses(startDate, endDate, userID)
    if err != nil {
        return err
    }

    netProfit, err := s.GetNetProfit(startDate, endDate, userID)
    if err != nil {
        return err
    }

    profitMargin, err := s.GetProfitMargin(startDate, endDate, userID)
    if err != nil {
        return err
    }

    costPerUnit, err := s.GetCostPerUnitSold(startDate, endDate, userID)
    if err != nil {
        return err
    }

    expenseRatio, err := s.GetExpenseToRevenueRatio(startDate, endDate, userID)
    if err != nil {
        return err
    }

    inventoryCost, err := s.GetTotalInventoryCost(userID)
    if err != nil {
        return err
    }

    // Create financial data struct
    financialData := models.FlockFinancialData{
        FlockID:       flockID,
        UserID:        userID,
        PeriodStart:   startDate,
        PeriodEnd:     endDate,
        TotalRevenue:  totalRevenue[flockID],
        TotalExpenses: totalExpenses[flockID],
        NetProfit:     netProfit[flockID],
        ProfitMargin:  profitMargin[flockID],
        CostPerUnit:   costPerUnit[flockID],
        ExpenseRatio:  expenseRatio[flockID],
        InventoryCost: inventoryCost[flockID],
        UpdatedAt:     time.Now(),
    }

    // Upsert: Insert if new, update if exists
    err = s.db.Clauses(clause.OnConflict{
        Columns:   []clause.Column{{Name: "flock_id"}, {Name: "user_id"}, {Name: "period_start"}},
        DoUpdates: clause.AssignmentColumns([]string{
            "total_revenue", "total_expenses", "net_profit", "profit_margin",
            "cost_per_unit", "expense_ratio", "inventory_cost", "updated_at",
        }),
    }).Create(&financialData).Error

    if err != nil {
        return err
    }

    log.Printf("Saved or updated financial data for flock %d from %s to %s", flockID, startDate, endDate)
    return nil
}
