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

// GetSalesByFarm retrieves sales records for a specific farm
func (s *SalesService) GetSalesByFarm(farmID uint) ([]models.Sale, error) {
	var sales []models.Sale
	err := s.DB.Where("farm_id = ?", farmID).Find(&sales).Error
	return sales, err
}

// GetSalesByFlock retrieves sales related to a specific flock within a farm,
// including timestamps for dynamic filtering.
func (s *SalesService) GetSalesByFlock(flockID uint, farmID uint) ([]models.Sale, error) {
    var sales []models.Sale
    err := s.DB.Where("flock_id = ? AND farm_id = ?", flockID, farmID).
        Order("created_at DESC").Find(&sales).Error
    return sales, err
}

// GetSalesByFlockAndPeriod retrieves sales for a flock within a given time range
func (s *SalesService) GetSalesByFlockAndPeriod(flockID uint, farmID uint, start, end time.Time) ([]models.Sale, error) {
    var sales []models.Sale
    err := s.DB.Where("flock_id = ? AND farm_id = ? AND created_at BETWEEN ? AND ?", flockID, farmID, start, end).
        Order("created_at DESC").Find(&sales).Error
    return sales, err
}


// AddSale adds a new sale, sends a WebSocket update, and notifies the user
func (s *SalesService) AddSale(sale *models.Sale) error {
	if err := s.DB.Omit("Flock").Create(sale).Error; err != nil {
		return err
	}

	// Send WebSocket update
	broadcast.SendSaleUpdate(sale.FarmID, "sale_added", *sale)

	// Send Notification
	broadcast.SendFarmNotification(
		sale.FarmID,
		"New Sale Added",
		"Your sale record has been successfully added.",
		"/sales",
	)

	return nil
}

// UpdateSale updates an existing sale, sends a WebSocket update, and notifies the user
func (s *SalesService) UpdateSale(sale *models.Sale) error {
	if err := s.DB.Omit("Flock").Save(sale).Error; err != nil {
		return err
	}

	// Send WebSocket update
	broadcast.SendSaleUpdate(sale.FarmID, "sale_updated", *sale)

	// Send Notification
	broadcast.SendFarmNotification(
		sale.FarmID,
		"Sale Updated",
		"Your sale record has been successfully updated.",
		"/sales",
	)

	return nil
}

// DeleteSale removes a sale record by ID, sends a WebSocket update, and notifies the user
func (s *SalesService) DeleteSale(saleID uint, farmID uint) error {
	var sale models.Sale
	if err := s.DB.Where("id = ? AND farm_id = ?", saleID, farmID).First(&sale).Error; err != nil {
		return errors.New("sale not found")
	}

	if err := s.DB.Delete(&sale).Error; err != nil {
		return err
	}

	// Send WebSocket update
	broadcast.SendSaleUpdate(farmID, "sale_deleted", saleID)

	// Send Notification
	broadcast.SendFarmNotification(
		farmID,
		"Sale Deleted",
		"A sale record has been removed from your account.",
		"/sales",
	)

	return nil
}
