package services

import (
	"errors"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/broadcast"
	"gorm.io/gorm"
	"log"
	"time"
	"fmt"
)

// ExpenseService provides methods to manage expenses
type ExpenseService struct {
	DB *gorm.DB
}

// NewExpenseService initializes a new service instance
func NewExpenseService(db *gorm.DB) *ExpenseService {
	return &ExpenseService{DB: db}
}

// GetExpensesByFarm retrieves expenses for a specific farm
func (s *ExpenseService) GetExpensesByFarm(farmID uint) ([]models.Expense, error) {
	var expenses []models.Expense
	err := s.DB.Where("farm_id = ?", farmID).Find(&expenses).Error
	return expenses, err
}

// GetExpensesByFlock retrieves expenses related to a specific flock within a farm,
// including timestamps for dynamic filtering.
func (s *ExpenseService) GetExpensesByFlock(flockID uint, farmID uint) ([]models.Expense, error) {
	var expenses []models.Expense
	err := s.DB.Where("flock_id = ? AND farm_id = ?", flockID, farmID).
		Order("created_at DESC").Find(&expenses).Error
	if err != nil {
		log.Printf("❌ Error fetching expenses for flock %d: %v", flockID, err)
		return nil, err
	}
	return expenses, nil
}

// GetExpensesByFlockAndPeriod retrieves expenses for a flock within a given time range
func (s *ExpenseService) GetExpensesByFlockAndPeriod(flockID uint, farmID uint, start, end time.Time) ([]models.Expense, error) {
	var expenses []models.Expense
	err := s.DB.Where("flock_id = ? AND farm_id = ? AND created_at BETWEEN ? AND ?", flockID, farmID, start, end).
		Order("created_at DESC").Find(&expenses).Error
	if err != nil {
		log.Printf("❌ Error fetching expenses for flock %d in period: %v", flockID, err)
		return nil, err
	}
	return expenses, nil
}

func (s *ExpenseService) AddExpense(expense *models.Expense) error {
	log.Println("ℹ️ Adding new expense...")

	if err := s.DB.Omit("Flock").Create(expense).Error; err != nil {
		log.Printf("❌ Error adding expense: %v\n", err)
		return err
	}
	log.Println("✅ Expense added successfully to the database.")

	// Send WebSocket update
	log.Println("📡 Sending WebSocket update for new expense...")
	broadcast.SendExpenseUpdate(expense.FarmID, "expense_added", *expense)
	log.Println("✅ WebSocket update sent.")

	// Send push notification
	message := fmt.Sprintf("A new expense of KES %.2f was added.", expense.Amount)
	log.Printf("🔔 Sending push notification: Title='New Expense', Message='%s'\n", message)
	broadcast.SendFarmNotification(expense.FarmID, "New Expense", message, "/expenses")
	log.Println("✅ Push notification sent.")

	return nil
}

// UpdateExpense updates an existing expense and sends a WebSocket update
func (s *ExpenseService) UpdateExpense(expense *models.Expense) error {
	if err := s.DB.Omit("Flock").Save(expense).Error; err != nil {
		return err
	}
	broadcast.SendExpenseUpdate(expense.FarmID, "expense_updated", *expense)
	return nil
}

// DeleteExpense removes an expense by ID and sends a WebSocket update
func (s *ExpenseService) DeleteExpense(expenseID uint, farmID uint) error {
	var expense models.Expense
	if err := s.DB.Where("id = ? AND farm_id = ?", expenseID, farmID).First(&expense).Error; err != nil {
		return errors.New("expense not found")
	}

	if err := s.DB.Delete(&expense).Error; err != nil {
		return err
	}

	broadcast.SendExpenseUpdate(farmID, "expense_deleted", expenseID)
	return nil
}
