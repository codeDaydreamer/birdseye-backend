package services

import (
	"errors"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/broadcast"
	"gorm.io/gorm"
)

// ExpenseService provides methods to manage expenses
type ExpenseService struct {
	DB *gorm.DB
}

// NewExpenseService initializes a new service instance
func NewExpenseService(db *gorm.DB) *ExpenseService {
	return &ExpenseService{DB: db}
}

// GetExpensesByUser retrieves expenses for a specific user
func (s *ExpenseService) GetExpensesByUser(userID uint) ([]models.Expense, error) {
	var expenses []models.Expense
	err := s.DB.Where("user_id = ?", userID).Find(&expenses).Error
	return expenses, err
}

// AddExpense adds a new expense and sends a WebSocket update
func (s *ExpenseService) AddExpense(expense *models.Expense) error {
	if err := s.DB.Create(expense).Error; err != nil {
		return err
	}
	broadcast.SendExpenseUpdate("expense_added", *expense)
	return nil
}

// UpdateExpense updates an existing expense and sends a WebSocket update
func (s *ExpenseService) UpdateExpense(expense *models.Expense) error {
	if err := s.DB.Save(expense).Error; err != nil {
		return err
	}
	broadcast.SendExpenseUpdate("expense_updated", *expense)
	return nil
}

// DeleteExpense removes an expense by ID and sends a WebSocket update
func (s *ExpenseService) DeleteExpense(expenseID uint, userID uint) error {
	var expense models.Expense
	if err := s.DB.Where("id = ? AND user_id = ?", expenseID, userID).First(&expense).Error; err != nil {
		return errors.New("expense not found")
	}

	if err := s.DB.Delete(&expense).Error; err != nil {
		return err
	}

	broadcast.SendExpenseUpdate("expense_deleted", expenseID)
	return nil
}
