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

// ExpenseHatchingEggs is the expense category bought hatching eggs are filed under
const ExpenseHatchingEggs = "Hatching eggs bought"

var (
	ErrIncubatorNotFound = errors.New("incubator not found")
	ErrIncubatorName     = errors.New("give the incubator a name")
	ErrIncubatorFull     = errors.New("that incubator doesn't have room for these eggs")
	ErrSettingNotFound   = errors.New("egg setting not found")
	ErrSettingName       = errors.New("give the setting a name")
	ErrSettingEggs       = errors.New("set at least 1 egg")
	ErrSettingSource     = errors.New("source must be own or bought")
	ErrSettingFlock      = errors.New("choose the batch the eggs came from")
	ErrSettingDone       = errors.New("this setting has already hatched or been closed")
	ErrSettingKept       = errors.New("a hatched or closed setting can't be deleted: its eggs were used")
	ErrCandleCounts      = errors.New("clear and rotten eggs can't add up to more than the eggs set")
	ErrHatchCounts       = errors.New("more chicks hatched than fertile eggs left")
	ErrCullCounts        = errors.New("can't cull more chicks than hatched")
	ErrHatchTarget       = errors.New("choose where the chicks go: a new batch or an existing one")
	ErrBookingNotFound   = errors.New("booking not found")
	ErrBookingCustomer   = errors.New("enter the customer's name")
	ErrBookingQuantity   = errors.New("book at least 1 chick")
	ErrBookingDone       = errors.New("this booking has already been fulfilled or cancelled")
)

// HatcheryService runs incubators, egg settings and chick bookings
type HatcheryService struct {
	DB     *gorm.DB
	Flocks *FlockService
	Sales  *SalesService
}

// --- Incubators ---

func (s *HatcheryService) ListIncubators(farmID uint) ([]models.Incubator, error) {
	list := []models.Incubator{}
	err := s.DB.Where("farm_id = ?", farmID).Order("name").Find(&list).Error
	return list, err
}

func (s *HatcheryService) SaveIncubator(farmID uint, inc *models.Incubator) error {
	inc.Name = strings.TrimSpace(inc.Name)
	if inc.Name == "" {
		return ErrIncubatorName
	}
	if inc.Capacity < 0 {
		inc.Capacity = 0
	}
	inc.FarmID = farmID
	if inc.ID == 0 {
		return s.DB.Create(inc).Error
	}
	res := s.DB.Model(&models.Incubator{}).Where("id = ? AND farm_id = ?", inc.ID, farmID).
		Updates(map[string]interface{}{"name": inc.Name, "capacity": inc.Capacity, "active": inc.Active})
	if res.Error == nil && res.RowsAffected == 0 {
		var n int64
		s.DB.Model(&models.Incubator{}).Where("id = ? AND farm_id = ?", inc.ID, farmID).Count(&n)
		if n == 0 {
			return ErrIncubatorNotFound
		}
	}
	return res.Error
}

func (s *HatcheryService) DeleteIncubator(farmID, id uint) error {
	res := s.DB.Where("id = ? AND farm_id = ?", id, farmID).Delete(&models.Incubator{})
	if res.Error == nil && res.RowsAffected == 0 {
		return ErrIncubatorNotFound
	}
	// Settings keep their history; they just lose the incubator link
	s.DB.Model(&models.EggSetting{}).Where("incubator_id = ?", id).Update("incubator_id", nil)
	return res.Error
}

// --- Egg settings ---

func (s *HatcheryService) ListSettings(farmID uint) ([]models.EggSetting, error) {
	list := []models.EggSetting{}
	err := s.DB.Where("farm_id = ?", farmID).Order("set_date DESC, id DESC").Find(&list).Error
	return list, err
}

func (s *HatcheryService) getSetting(tx *gorm.DB, farmID, id uint) (*models.EggSetting, error) {
	var setting models.EggSetting
	if err := tx.Where("id = ? AND farm_id = ?", id, farmID).First(&setting).Error; err != nil {
		return nil, ErrSettingNotFound
	}
	return &setting, nil
}

// SettingInput is a new batch of eggs set to hatch
type SettingInput struct {
	Name          string  `json:"name"`
	IncubatorID   *uint   `json:"incubator_id"`
	SetDate       string  `json:"set_date"`
	Source        string  `json:"source"` // own | bought
	SourceFlockID *uint   `json:"source_flock_id"`
	Supplier      string  `json:"supplier"`
	CostPerEgg    float64 `json:"cost_per_egg"`
	EggsSet       int     `json:"eggs_set"`
	Notes         string  `json:"notes"`
}

// CreateSetting sets eggs. Own eggs come out of their batch's eggs on hand (refused beyond it); bought eggs
// with a price file their purchase as an expense.
func (s *HatcheryService) CreateSetting(farmID, userID uint, in SettingInput) (*models.EggSetting, error) {
	setting := models.EggSetting{FarmID: farmID, Name: strings.TrimSpace(in.Name), Source: in.Source,
		Supplier: strings.TrimSpace(in.Supplier), CostPerEgg: in.CostPerEgg, EggsSet: in.EggsSet,
		Notes: strings.TrimSpace(in.Notes), Status: models.SettingIncubating, SetDate: time.Now()}
	if in.SetDate != "" {
		d, err := ParseDate(in.SetDate)
		if err != nil {
			return nil, err
		}
		setting.SetDate = d
	}
	if in.IncubatorID != nil && *in.IncubatorID > 0 {
		setting.IncubatorID = in.IncubatorID
	}
	switch {
	case setting.Name == "":
		return nil, ErrSettingName
	case setting.EggsSet < 1:
		return nil, ErrSettingEggs
	case setting.Source != "own" && setting.Source != "bought":
		return nil, ErrSettingSource
	case setting.Source == "own" && (in.SourceFlockID == nil || *in.SourceFlockID == 0):
		return nil, ErrSettingFlock
	case setting.CostPerEgg < 0:
		return nil, ErrSalePrice
	}
	if setting.Source == "own" {
		setting.SourceFlockID = in.SourceFlockID
		setting.Supplier, setting.CostPerEgg = "", 0
	}

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if setting.IncubatorID != nil {
			var inc models.Incubator
			if err := tx.Where("id = ? AND farm_id = ?", *setting.IncubatorID, farmID).First(&inc).Error; err != nil {
				return ErrIncubatorNotFound
			}
			if inc.Capacity > 0 {
				var inside int64
				tx.Model(&models.EggSetting{}).Where("incubator_id = ? AND status = ?", inc.ID, models.SettingIncubating).
					Select("COALESCE(SUM(eggs_set - clear - rotten), 0)").Scan(&inside)
				if int(inside)+setting.EggsSet > inc.Capacity {
					return fmt.Errorf("%w (%d of %d places used)", ErrIncubatorFull, inside, inc.Capacity)
				}
			}
		}
		if setting.Source == "own" {
			stock, err := EggStockByFlock(tx, farmID, 0)
			if err != nil {
				return err
			}
			found := false
			for _, f := range stock {
				if f.FlockID == *setting.SourceFlockID {
					found = true
					if f.OnHand < setting.EggsSet {
						return &NotEnoughEggsError{Flock: f.FlockName, OnHand: f.OnHand}
					}
				}
			}
			if !found {
				return ErrSettingFlock
			}
		}
		if setting.Source == "bought" && setting.CostPerEgg > 0 {
			desc := fmt.Sprintf("%d hatching eggs for %s", setting.EggsSet, setting.Name)
			if setting.Supplier != "" {
				desc += " from " + setting.Supplier
			}
			expense := models.Expense{FarmID: farmID, CreatedByUserID: userID, Date: setting.SetDate,
				Description: desc, Amount: setting.CostPerEgg * float64(setting.EggsSet), Category: ExpenseHatchingEggs}
			if err := tx.Create(&expense).Error; err != nil {
				return err
			}
			setting.ExpenseID = &expense.ID
			broadcast.SendExpenseUpdate(farmID, "expense_added", expense)
		}
		return tx.Create(&setting).Error
	})
	if err != nil {
		return nil, err
	}
	setting.FillComputed(time.Now())
	return &setting, nil
}

// Candle records the clear (infertile) and rotten eggs taken out at candling
func (s *HatcheryService) Candle(farmID, id uint, clear, rotten int, date string) (*models.EggSetting, error) {
	setting, err := s.getSetting(s.DB, farmID, id)
	if err != nil {
		return nil, err
	}
	if setting.Status != models.SettingIncubating {
		return nil, ErrSettingDone
	}
	if clear < 0 || rotten < 0 || clear+rotten > setting.EggsSet {
		return nil, ErrCandleCounts
	}
	when := time.Now()
	if date != "" {
		if when, err = ParseDate(date); err != nil {
			return nil, err
		}
	}
	setting.Clear, setting.Rotten, setting.CandledAt = clear, rotten, &when
	if err := s.DB.Model(setting).Select("clear", "rotten", "candled_at").Updates(setting).Error; err != nil {
		return nil, err
	}
	setting.FillComputed(time.Now())
	return setting, nil
}

// HatchInput is the hatch count and where the good chicks go
type HatchInput struct {
	Hatched int    `json:"hatched"`
	Cull    int    `json:"cull"`
	Date    string `json:"date"`
	// Put the chicks in a new batch...
	NewBatch *struct {
		Name           string `json:"name"`
		ProductionType string `json:"production_type"`
		Breed          string `json:"breed"`
	} `json:"new_batch"`
	// ...or an existing one
	FlockID uint `json:"flock_id"`
}

// Hatch records the hatch and places the good chicks (hatched - cull) in a new or existing batch
func (s *HatcheryService) Hatch(farmID, userID, id uint, in HatchInput) (*models.EggSetting, error) {
	setting, err := s.getSetting(s.DB, farmID, id)
	if err != nil {
		return nil, err
	}
	if setting.Status != models.SettingIncubating {
		return nil, ErrSettingDone
	}
	switch {
	case in.Hatched < 0 || in.Hatched > setting.EggsSet-setting.Clear-setting.Rotten:
		return nil, ErrHatchCounts
	case in.Cull < 0 || in.Cull > in.Hatched:
		return nil, ErrCullCounts
	}
	when := time.Now()
	if in.Date != "" {
		if when, err = ParseDate(in.Date); err != nil {
			return nil, err
		}
	}
	good := in.Hatched - in.Cull
	if good > 0 && in.NewBatch == nil && in.FlockID == 0 {
		return nil, ErrHatchTarget
	}
	note := "Hatched from " + setting.Name

	var flockID uint
	var newFlock models.Flock
	if good > 0 && in.NewBatch != nil {
		day := when.Format("2006-01-02")
		zero := 0
		hatched := models.SourceHatched
		if newFlock, err = PrepareFlock(farmID, FlockInput{Name: &in.NewBatch.Name, Breed: &in.NewBatch.Breed,
			ProductionType: &in.NewBatch.ProductionType, Source: &hatched, PlacedAt: &day,
			AgeAtPlacementDays: &zero, Birds: good}); err != nil {
			return nil, err
		}
	}

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if good > 0 {
			if in.NewBatch != nil {
				if err := CreateFlockTx(tx, &newFlock, good, userID, note); err != nil {
					return err
				}
				flockID = newFlock.ID
			} else {
				placed := models.BirdMovement{FarmID: farmID, FlockID: in.FlockID, Type: models.MovePlaced,
					Delta: good, Date: when, Notes: note, CreatedByUserID: userID}
				if err := CheckBirdLimit(farmID, good, 0); err != nil {
					return err
				}
				if err := ApplyMovement(tx, &placed); err != nil {
					return err
				}
				flockID = in.FlockID
			}
		}
		setting.Hatched, setting.Cull, setting.HatchedAt = in.Hatched, in.Cull, &when
		setting.Status = models.SettingHatched
		if flockID != 0 {
			setting.ChickFlockID = &flockID
		}
		return tx.Model(setting).Select("hatched", "cull", "hatched_at", "status", "chick_flock_id").
			Updates(setting).Error
	})
	if err != nil {
		return nil, err
	}
	if flockID != 0 {
		BroadcastFlocks(s.DB, farmID, flockID)
	}
	setting.FillComputed(time.Now())
	return setting, nil
}

// CloseSetting ends a setting that won't hatch (power cut, all clear...)
func (s *HatcheryService) CloseSetting(farmID, id uint) error {
	setting, err := s.getSetting(s.DB, farmID, id)
	if err != nil {
		return err
	}
	if setting.Status != models.SettingIncubating {
		return ErrSettingDone
	}
	return s.DB.Model(setting).Update("status", models.SettingClosed).Error
}

// DeleteSetting removes a setting entered by mistake (and the purchase expense of bought eggs). Only while
// it's incubating: own eggs are subtracted from eggs on hand by their setting, so deleting a hatched or closed
// one would bring eggs back that became chicks or were thrown away.
func (s *HatcheryService) DeleteSetting(farmID, id uint) error {
	setting, err := s.getSetting(s.DB, farmID, id)
	if err != nil {
		return err
	}
	if setting.Status != models.SettingIncubating {
		return ErrSettingKept
	}
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if setting.ExpenseID != nil {
			if err := tx.Where("id = ? AND farm_id = ?", *setting.ExpenseID, farmID).Delete(&models.Expense{}).Error; err != nil {
				return err
			}
		}
		return tx.Delete(setting).Error
	})
}

// --- Chick bookings ---

func (s *HatcheryService) ListBookings(farmID uint) ([]models.ChickBooking, error) {
	list := []models.ChickBooking{}
	err := s.DB.Where("farm_id = ?", farmID).Order("status = 'booked' DESC, wanted_date, id").Find(&list).Error
	return list, err
}

func (s *HatcheryService) SaveBooking(farmID uint, b *models.ChickBooking) error {
	b.Customer = strings.TrimSpace(b.Customer)
	switch {
	case b.Customer == "":
		return ErrBookingCustomer
	case b.Quantity < 1:
		return ErrBookingQuantity
	case b.PricePerChick < 0 || b.Deposit < 0:
		return ErrSalePrice
	}
	b.FarmID = farmID
	if b.ID == 0 {
		b.Status, b.SaleID = models.BookingBooked, nil
		return s.DB.Create(b).Error
	}
	var old models.ChickBooking
	if err := s.DB.Where("id = ? AND farm_id = ?", b.ID, farmID).First(&old).Error; err != nil {
		return ErrBookingNotFound
	}
	if b.Status != models.BookingCancelled && b.Status != models.BookingBooked {
		b.Status = old.Status
	}
	if old.Status == models.BookingFulfilled {
		b.Status = old.Status
	}
	b.SaleID, b.CreatedAt = old.SaleID, old.CreatedAt
	return s.DB.Save(b).Error
}

func (s *HatcheryService) DeleteBooking(farmID, id uint) error {
	res := s.DB.Where("id = ? AND farm_id = ?", id, farmID).Delete(&models.ChickBooking{})
	if res.Error == nil && res.RowsAffected == 0 {
		return ErrBookingNotFound
	}
	return res.Error
}

// FulfilBooking hands over the chicks: a chicks sale from the batch (which takes the birds out of it), linked
// to the booking. Paid if the deposit covers it, otherwise pending.
func (s *HatcheryService) FulfilBooking(farmID, userID, id, flockID uint) (*models.ChickBooking, error) {
	var b models.ChickBooking
	if err := s.DB.Where("id = ? AND farm_id = ?", id, farmID).First(&b).Error; err != nil {
		return nil, ErrBookingNotFound
	}
	if b.Status != models.BookingBooked {
		return nil, ErrBookingDone
	}
	amount := float64(b.Quantity) * b.PricePerChick
	status := "pending..."
	if b.Deposit >= amount {
		status = "paid"
	}
	desc := "Booking: " + b.Customer
	if b.Deposit > 0 {
		desc += fmt.Sprintf(" (deposit KES %.0f)", b.Deposit)
	}
	sale := models.Sale{FarmID: farmID, CreatedByUserID: userID, FlockID: &flockID, ProductType: "chicks",
		Unit: "bird", Quantity: b.Quantity, UnitPrice: b.PricePerChick, Description: desc, Date: time.Now(),
		SaleType: "Cash", Status: status}
	if err := s.Sales.AddSale(&sale); err != nil {
		return nil, err
	}
	b.Status, b.SaleID = models.BookingFulfilled, &sale.ID
	if err := s.DB.Model(&b).Select("status", "sale_id").Updates(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}
