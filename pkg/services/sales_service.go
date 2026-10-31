package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"birdseye-backend/pkg/broadcast"
	"birdseye-backend/pkg/models"

	"gorm.io/gorm"
)

var (
	ErrSaleNotFound     = errors.New("sale not found")
	ErrSaleProduct      = errors.New("product must be eggs, hatching_eggs, chicks, live_birds, spent_hens, meat, manure, feed or other")
	ErrSaleUnit         = errors.New("that unit doesn't fit the product")
	ErrSaleQuantity     = errors.New("enter a quantity of at least 1")
	ErrSalePrice        = errors.New("the price can't be negative")
	ErrSaleNeedsFlock   = errors.New("choose the batch the birds or eggs came from")
	ErrSaleMeatWeight   = errors.New("enter the weight sold in kg")
	ErrSaleMeatBirds    = errors.New("enter how many birds were slaughtered")
	ErrSaleFlockMissing = errors.New("flock not found")
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

// GetSalesByFlock retrieves sales related to a specific flock within a farm
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

// prepare checks the sale's product, unit and numbers, and fills in its category, product name and amount
func prepare(sale *models.Sale) error {
	product, ok := models.SaleProducts[sale.ProductType]
	if !ok {
		return ErrSaleProduct
	}
	if sale.Unit == "" {
		sale.Unit = product.Units[0]
	}
	unitOK := false
	for _, u := range product.Units {
		unitOK = unitOK || u == sale.Unit
	}
	switch {
	case !unitOK:
		return ErrSaleUnit
	case sale.UnitPrice < 0:
		return ErrSalePrice
	case product.Stock != "" && sale.FlockID == nil:
		return ErrSaleNeedsFlock
	}
	if product.Stock == "meat" {
		if sale.WeightKg <= 0 {
			return ErrSaleMeatWeight
		}
		if sale.Birds < 1 {
			return ErrSaleMeatBirds
		}
		sale.Quantity = sale.Birds
		sale.Amount = sale.WeightKg * sale.UnitPrice
	} else {
		if sale.Quantity < 1 {
			return ErrSaleQuantity
		}
		sale.WeightKg, sale.Birds = 0, 0
		sale.Amount = float64(sale.Quantity) * sale.UnitPrice
	}
	sale.Category = product.Label
	if strings.TrimSpace(sale.Product) == "" {
		sale.Product = product.Label
	}
	return nil
}

// takeStock checks there are enough eggs, and records the birds leaving their batch, for a saved sale
func takeStock(tx *gorm.DB, sale *models.Sale) error {
	if eggs := sale.EggsMoved(); eggs > 0 {
		stock, err := EggStockByFlock(tx, sale.FarmID, sale.ID)
		if err != nil {
			return err
		}
		for _, f := range stock {
			if f.FlockID == *sale.FlockID && f.OnHand < eggs {
				return &NotEnoughEggsError{Flock: f.FlockName, OnHand: f.OnHand}
			}
		}
	}
	if birds := sale.BirdsMoved(); birds > 0 {
		m := models.BirdMovement{FarmID: sale.FarmID, FlockID: *sale.FlockID, Type: models.MoveSold,
			Delta: -birds, Date: sale.Date, SaleID: &sale.ID, CreatedByUserID: sale.CreatedByUserID,
			Notes: fmt.Sprintf("Sale %s", sale.RefNo)}
		return ApplyMovement(tx, &m)
	}
	return nil
}

// returnStock removes a sale's bird movements (eggs need nothing: their stock is computed from sales)
func returnStock(tx *gorm.DB, saleID uint) ([]uint, error) {
	var moves []models.BirdMovement
	if err := tx.Where("sale_id = ?", saleID).Find(&moves).Error; err != nil {
		return nil, err
	}
	var flocks []uint
	for _, m := range moves {
		if err := tx.Delete(&models.BirdMovement{}, m.ID).Error; err != nil {
			return nil, err
		}
		if _, err := RecomputeFlockTotals(tx, m.FlockID); err != nil {
			return nil, err
		}
		flocks = append(flocks, m.FlockID)
	}
	return flocks, nil
}

func checkFlock(tx *gorm.DB, sale *models.Sale) error {
	if sale.FlockID == nil {
		return nil
	}
	var n int64
	tx.Model(&models.Flock{}).Where("id = ? AND farm_id = ?", *sale.FlockID, sale.FarmID).Count(&n)
	if n == 0 {
		return ErrSaleFlockMissing
	}
	return nil
}

// AddSale records a sale and takes its birds or eggs out of stock, in one transaction
func (s *SalesService) AddSale(sale *models.Sale) error {
	if err := prepare(sale); err != nil {
		return err
	}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := checkFlock(tx, sale); err != nil {
			return err
		}
		if err := tx.Omit("Flock").Create(sale).Error; err != nil {
			return err
		}
		return takeStock(tx, sale)
	})
	if err != nil {
		return err
	}
	if sale.FlockID != nil && sale.BirdsMoved() > 0 {
		BroadcastFlocks(s.DB, sale.FarmID, *sale.FlockID)
	}
	broadcast.SendSaleUpdate(sale.FarmID, "sale_added", *sale)
	broadcast.SendFarmNotification(sale.FarmID, "New Sale Added",
		fmt.Sprintf("%s: KES %.0f", sale.Product, sale.Amount), "/sales")
	return nil
}

// UpdateSale saves an edited sale: its old stock movement is put back and the new one taken
func (s *SalesService) UpdateSale(sale *models.Sale) error {
	if err := prepare(sale); err != nil {
		return err
	}
	var affected []uint
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := checkFlock(tx, sale); err != nil {
			return err
		}
		returned, err := returnStock(tx, sale.ID)
		if err != nil {
			return err
		}
		affected = returned
		if err := tx.Omit("Flock").Save(sale).Error; err != nil {
			return err
		}
		return takeStock(tx, sale)
	})
	if err != nil {
		return err
	}
	if sale.FlockID != nil && sale.BirdsMoved() > 0 {
		affected = append(affected, *sale.FlockID)
	}
	BroadcastFlocks(s.DB, sale.FarmID, affected...)
	broadcast.SendSaleUpdate(sale.FarmID, "sale_updated", *sale)
	return nil
}

// DeleteSale removes a sale and puts its birds back in their batch
func (s *SalesService) DeleteSale(saleID uint, farmID uint) error {
	var affected []uint
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ? AND farm_id = ?", saleID, farmID).Delete(&models.Sale{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrSaleNotFound
		}
		var err error
		affected, err = returnStock(tx, saleID)
		return err
	})
	if err != nil {
		return err
	}
	BroadcastFlocks(s.DB, farmID, affected...)
	broadcast.SendSaleUpdate(farmID, "sale_deleted", saleID)
	return nil
}
