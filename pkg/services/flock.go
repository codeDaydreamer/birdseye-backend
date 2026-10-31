package services

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"birdseye-backend/pkg/broadcast"
	"birdseye-backend/pkg/models"

	"gorm.io/gorm"
)

// ExpenseBirdsBought is the expense category a bought batch's purchase is filed under
const ExpenseBirdsBought = "Chicks / birds bought"

var (
	ErrFlockName           = errors.New("give the batch a name")
	ErrFlockBreed          = errors.New("choose a breed")
	ErrProductionType      = errors.New("type must be layers, broilers, kienyeji, breeders or chicks")
	ErrFlockSource         = errors.New("source must be bought or hatched")
	ErrFlockBirds          = errors.New("a new batch needs at least 1 bird")
	ErrFlockAge            = errors.New("age at placement can't be negative")
	ErrFlockCost           = errors.New("cost per bird can't be negative")
	ErrFlockPlacedInFuture = errors.New("the placement date can't be in the future")
	ErrFlockStatus         = errors.New("status must be active or closed")
	ErrBadDate             = errors.New("dates must look like 2026-10-31")
)

type FlockService struct {
	DB             *gorm.DB
	SalesService   *SalesService
	ExpenseService *ExpenseService
}

func NewFlockService(db *gorm.DB, salesService *SalesService, expenseService *ExpenseService) *FlockService {
	return &FlockService{DB: db, SalesService: salesService, ExpenseService: expenseService}
}

// FlockInput is what the farmer fills in for a batch. Birds is only used when creating it: after that the
// count changes through movements.
type FlockInput struct {
	Name               *string  `json:"name"`
	Breed              *string  `json:"breed"`
	ProductionType     *string  `json:"production_type"`
	Source             *string  `json:"source"`
	Supplier           *string  `json:"supplier"`
	CostPerBird        *float64 `json:"cost_per_bird"`
	PlacedAt           *string  `json:"placed_at"` // YYYY-MM-DD or RFC 3339
	AgeAtPlacementDays *int     `json:"age_at_placement_days"`
	Health             *float64 `json:"health"`
	Status             *string  `json:"status"`
	Birds              int      `json:"birds"`
}

// ParseDate reads a date sent as YYYY-MM-DD or an RFC 3339 timestamp
func ParseDate(s string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Time{}, ErrBadDate
}

// apply copies the given fields onto the batch, checking each one
func (in FlockInput) apply(f *models.Flock) error {
	if in.Name != nil {
		f.Name = strings.TrimSpace(*in.Name)
	}
	if in.Breed != nil {
		f.Breed = strings.TrimSpace(*in.Breed)
	}
	if in.ProductionType != nil {
		f.ProductionType = *in.ProductionType
	}
	if in.Source != nil {
		f.Source = *in.Source
	}
	if in.Supplier != nil {
		f.Supplier = strings.TrimSpace(*in.Supplier)
	}
	if in.CostPerBird != nil {
		f.CostPerBird = *in.CostPerBird
	}
	if in.PlacedAt != nil {
		t, err := ParseDate(*in.PlacedAt)
		if err != nil {
			return err
		}
		f.PlacedAt = t
	}
	if in.AgeAtPlacementDays != nil {
		f.AgeAtPlacementDays = *in.AgeAtPlacementDays
	}
	if in.Health != nil {
		f.Health = *in.Health
	}
	if in.Status != nil && *in.Status != f.Status {
		switch *in.Status {
		case models.FlockClosed:
			now := time.Now()
			f.ClosedAt = &now
		case models.FlockActive:
			f.ClosedAt = nil
		default:
			return ErrFlockStatus
		}
		f.Status = *in.Status
	}

	switch {
	case f.Name == "":
		return ErrFlockName
	case f.Breed == "":
		return ErrFlockBreed
	case !models.ProductionTypes[f.ProductionType]:
		return ErrProductionType
	case f.Source != models.SourceBought && f.Source != models.SourceHatched:
		return ErrFlockSource
	case f.AgeAtPlacementDays < 0:
		return ErrFlockAge
	case f.CostPerBird < 0:
		return ErrFlockCost
	case f.PlacedAt.After(time.Now()):
		return ErrFlockPlacedInFuture
	}
	return nil
}

// GetFlocksByFarm lists the farm's batches
func (s *FlockService) GetFlocksByFarm(farmID uint) ([]models.Flock, error) {
	var flocks []models.Flock
	err := s.DB.Where("farm_id = ?", farmID).Order("status, placed_at DESC").Find(&flocks).Error
	return flocks, err
}

// GetFlockByID loads one of the farm's batches
func (s *FlockService) GetFlockByID(flockID, farmID uint) (*models.Flock, error) {
	var flock models.Flock
	if err := s.DB.Where("id = ? AND farm_id = ?", flockID, farmID).First(&flock).Error; err != nil {
		return nil, ErrFlockNotFound
	}
	return &flock, nil
}

// CreateFlock starts a batch: the batch, its "placed" movement and, for bought birds with a price, the
// purchase expense, all in one transaction
func (s *FlockService) CreateFlock(farmID, userID uint, in FlockInput) (*models.Flock, error) {
	flock, err := PrepareFlock(farmID, in)
	if err != nil {
		return nil, err
	}
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		return CreateFlockTx(tx, &flock, in.Birds, userID, "")
	})
	if err != nil {
		return nil, err
	}
	created, err := s.GetFlockByID(flock.ID, farmID)
	if err != nil {
		return nil, err
	}
	broadcast.SendFlockUpdate(farmID, "flock_added", *created)
	return created, nil
}

// PrepareFlock checks a new batch's details and the plan's limits, returning the batch to create
func PrepareFlock(farmID uint, in FlockInput) (models.Flock, error) {
	flock := models.Flock{
		FarmID: farmID, ProductionType: models.ProductionLayers, Source: models.SourceBought,
		Status: models.FlockActive, PlacedAt: time.Now(), Health: 100,
	}
	if err := in.apply(&flock); err != nil {
		return flock, err
	}
	flock.Status, flock.ClosedAt = models.FlockActive, nil
	if in.Birds < 1 {
		return flock, ErrFlockBirds
	}
	if err := CheckFlockLimit(farmID); err != nil {
		return flock, err
	}
	return flock, CheckBirdLimit(farmID, in.Birds, 0)
}

// CreateFlockTx saves a prepared batch with its "placed" movement and, for bought birds with a price, the
// purchase expense, inside tx
func CreateFlockTx(tx *gorm.DB, flock *models.Flock, birds int, userID uint, note string) error {
	if err := tx.Create(flock).Error; err != nil {
		return err
	}
	placed := models.BirdMovement{FarmID: flock.FarmID, FlockID: flock.ID, Type: models.MovePlaced,
		Delta: birds, Date: flock.PlacedAt, Notes: note, CreatedByUserID: userID}
	if err := ApplyMovement(tx, &placed); err != nil {
		return err
	}
	if flock.Source == models.SourceBought && flock.CostPerBird > 0 {
		desc := fmt.Sprintf("%d birds for %s", birds, flock.Name)
		if flock.Supplier != "" {
			desc += " from " + flock.Supplier
		}
		expense := models.Expense{FarmID: flock.FarmID, FlockID: &flock.ID, CreatedByUserID: userID,
			Date: flock.PlacedAt, Description: desc, Amount: flock.CostPerBird * float64(birds),
			Category: ExpenseBirdsBought}
		if err := tx.Create(&expense).Error; err != nil {
			return err
		}
		broadcast.SendExpenseUpdate(flock.FarmID, "expense_added", expense)
	}
	return nil
}

// UpdateFlock changes a batch's details. Bird counts aren't editable: they change through movements.
func (s *FlockService) UpdateFlock(farmID, flockID uint, in FlockInput) (*models.Flock, error) {
	flock, err := s.GetFlockByID(flockID, farmID)
	if err != nil {
		return nil, err
	}
	if err := in.apply(flock); err != nil {
		return nil, err
	}
	err = s.DB.Model(flock).Select("name", "breed", "production_type", "source", "supplier", "cost_per_bird",
		"placed_at", "age_at_placement_days", "health", "status", "closed_at").Updates(flock).Error
	if err != nil {
		return nil, err
	}
	updated, err := s.GetFlockByID(flockID, farmID)
	if err != nil {
		return nil, err
	}
	broadcast.SendFlockUpdate(farmID, "flock_updated", *updated)
	return updated, nil
}

// DeleteFlock removes a batch and its movement log
func (s *FlockService) DeleteFlock(flockID, farmID uint) error {
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ? AND farm_id = ?", flockID, farmID).Delete(&models.Flock{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrFlockNotFound
		}
		return tx.Where("flock_id = ?", flockID).Delete(&models.BirdMovement{}).Error
	})
	if err != nil {
		return err
	}
	broadcast.SendFlockUpdate(farmID, "flock_deleted", flockID)
	return nil
}

// GetMovements lists a batch's movement log, newest first
func (s *FlockService) GetMovements(flockID, farmID uint) ([]models.BirdMovement, error) {
	if _, err := s.GetFlockByID(flockID, farmID); err != nil {
		return nil, err
	}
	moves := []models.BirdMovement{}
	err := s.DB.Where("flock_id = ? AND farm_id = ?", flockID, farmID).
		Order("date DESC, id DESC").Find(&moves).Error
	return moves, err
}

// MovementInput is a death, cull, recount or transfer the farmer records
type MovementInput struct {
	Type      string `json:"type"` // died | culled | adjust | transfer
	Quantity  int    `json:"quantity"`
	Date      string `json:"date"`
	Cause     string `json:"cause"`
	Notes     string `json:"notes"`
	ToFlockID uint   `json:"to_flock_id"`
}

// RecordMovement applies a farmer's movement to a batch. For adjust, quantity is the change (+ or -).
func (s *FlockService) RecordMovement(farmID, userID, flockID uint, in MovementInput) error {
	date := time.Now()
	if in.Date != "" {
		d, err := ParseDate(in.Date)
		if err != nil {
			return err
		}
		date = d
	}
	if in.Quantity == 0 || (in.Type != models.MoveAdjust && in.Quantity < 0) {
		return ErrInvalidQuantity
	}
	notes := strings.TrimSpace(in.Notes)
	if len(notes) > 255 {
		notes = notes[:255]
	}

	affected := []uint{flockID}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		m := models.BirdMovement{FarmID: farmID, FlockID: flockID, Date: date, Notes: notes,
			CreatedByUserID: userID}
		switch in.Type {
		case models.MoveDied:
			cause := in.Cause
			if cause == "" {
				cause = "unknown"
			}
			if !models.DeathCauses[cause] {
				return ErrInvalidCause
			}
			m.Type, m.Delta, m.Cause = models.MoveDied, -in.Quantity, cause
		case models.MoveCulled:
			m.Type, m.Delta = models.MoveCulled, -in.Quantity
		case models.MoveAdjust:
			m.Type, m.Delta = models.MoveAdjust, in.Quantity
		case "transfer":
			affected = append(affected, in.ToFlockID)
			return TransferBirds(tx, farmID, userID, flockID, in.ToFlockID, in.Quantity, date, notes)
		default:
			return ErrInvalidMovement
		}
		return ApplyMovement(tx, &m)
	})
	if err != nil {
		return err
	}
	BroadcastFlocks(s.DB, farmID, affected...)
	return nil
}

// UndoMovement deletes a death, cull, recount or transfer
func (s *FlockService) UndoMovement(farmID, movementID uint) error {
	var affected []uint
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		affected, err = DeleteMovement(tx, farmID, movementID)
		return err
	})
	if err != nil {
		return err
	}
	BroadcastFlocks(s.DB, farmID, affected...)
	return nil
}

// RefreshFinancials rewrites the batch's row in flocks_financial_data for the current month (read by
// /flock-financial)
func (s *FlockService) RefreshFinancials(flock *models.Flock) {
	start := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, -1)
	var revenue, eggSales, expenses float64

	sales, err := s.SalesService.GetSalesByFlockAndPeriod(flock.ID, flock.FarmID, start, end)
	if err != nil {
		log.Printf("Financials: loading sales of flock %d: %v", flock.ID, err)
		return
	}
	for _, sale := range sales {
		if sale.Category == "Egg Sales" {
			eggSales += sale.Amount
		}
		revenue += sale.Amount
	}
	rows, err := s.ExpenseService.GetExpensesByFlockAndPeriod(flock.ID, flock.FarmID, start, end)
	if err != nil {
		log.Printf("Financials: loading expenses of flock %d: %v", flock.ID, err)
		return
	}
	for _, e := range rows {
		expenses += e.Amount
	}

	data := models.FlocksFinancialData{FlockID: flock.ID, FarmID: flock.FarmID, Month: int(start.Month()),
		Year: start.Year(), Revenue: revenue, EggSales: eggSales, Expenses: expenses, NetRevenue: revenue - expenses}
	if err := s.DB.Where("flock_id = ? AND farm_id = ? AND month = ? AND year = ?", flock.ID, flock.FarmID,
		start.Month(), start.Year()).Assign(data).FirstOrCreate(&data).Error; err != nil {
		log.Printf("Financials: saving flock %d: %v", flock.ID, err)
	}
}
