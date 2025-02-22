package services

import (
	"errors"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/broadcast"
	"gorm.io/gorm"
	"time"
)

// SalesService provides methods to manage sales
type SalesService struct {
	DB *gorm.DB
}

// NewSalesService initializes a new service instance
func NewSalesService(db *gorm.DB) *SalesService {
	return &SalesService{DB: db}
}

// GetSalesByUser retrieves sales records for a specific user
func (s *SalesService) GetSalesByUser(userID uint) ([]models.Sale, error) {
	var sales []models.Sale
	err := s.DB.Where("user_id = ?", userID).Find(&sales).Error
	return sales, err
}

// GetSalesByFlock retrieves sales related to a specific flock for an authenticated user,
// including timestamps for dynamic filtering.
func (s *SalesService) GetSalesByFlock(flockID uint, userID uint) ([]models.Sale, error) {
    var sales []models.Sale
    err := s.DB.Where("flock_id = ? AND user_id = ?", flockID, userID).
        Order("created_at DESC").Find(&sales).Error
    return sales, err
}

// GetSalesByFlockAndPeriod retrieves sales for a flock within a given time range
func (s *SalesService) GetSalesByFlockAndPeriod(flockID uint, userID uint, start, end time.Time) ([]models.Sale, error) {
    var sales []models.Sale
    err := s.DB.Where("flock_id = ? AND user_id = ? AND created_at BETWEEN ? AND ?", flockID, userID, start, end).
        Order("created_at DESC").Find(&sales).Error
    return sales, err
}


// AddSale adds a new sale and sends a WebSocket update
func (s *SalesService) AddSale(sale *models.Sale) error {
	if err := s.DB.Create(sale).Error; err != nil {
		return err
	}
	broadcast.SendSaleUpdate("sale_added", *sale)
	return nil
}

// UpdateSale updates an existing sale and sends a WebSocket update
func (s *SalesService) UpdateSale(sale *models.Sale) error {
	if err := s.DB.Save(sale).Error; err != nil {
		return err
	}
	broadcast.SendSaleUpdate("sale_updated", *sale)
	return nil
}

// DeleteSale removes a sale record by ID and sends a WebSocket update
func (s *SalesService) DeleteSale(saleID uint, userID uint) error {
	var sale models.Sale
	if err := s.DB.Where("id = ? AND user_id = ?", saleID, userID).First(&sale).Error; err != nil {
		return errors.New("sale not found")
	}

	if err := s.DB.Delete(&sale).Error; err != nil {
		return err
	}

	broadcast.SendSaleUpdate("sale_deleted", saleID)
	return nil
}
