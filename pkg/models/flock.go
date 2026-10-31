package models

import (
	"time"

	"gorm.io/gorm"
)

// Production types: what a batch of birds is kept for
const (
	ProductionLayers   = "layers"
	ProductionBroilers = "broilers"
	ProductionKienyeji = "kienyeji"
	ProductionBreeders = "breeders"
	ProductionChicks   = "chicks" // reared to sell as chicks, growers or point-of-lay
)

// ProductionTypes lists the valid production types
var ProductionTypes = map[string]bool{
	ProductionLayers: true, ProductionBroilers: true, ProductionKienyeji: true,
	ProductionBreeders: true, ProductionChicks: true,
}

// Where a batch's birds came from
const (
	SourceBought  = "bought"
	SourceHatched = "hatched"
)

// Batch statuses
const (
	FlockActive = "active"
	FlockClosed = "closed"
)

// Flock is a batch of birds kept together, from placement (bought or hatched here) until it's closed.
// Its counts are running totals of its BirdMovements, kept on the row so lists and plan limits stay cheap;
// RecomputeFlockTotals is the only thing that writes them.
type Flock struct {
	ID             uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	FarmID         uint   `json:"farm_id" gorm:"index;not null"` // tenant
	Name           string `json:"name" gorm:"not null"`
	Breed          string `json:"breed" gorm:"not null"`
	ProductionType string `json:"production_type" gorm:"type:varchar(16);not null;default:layers"`
	Status         string `json:"status" gorm:"type:varchar(16);not null;default:active"`

	Source             string     `json:"source" gorm:"type:varchar(16);not null;default:bought"`
	Supplier           string     `json:"supplier" gorm:"type:varchar(120)"`
	CostPerBird        float64    `json:"cost_per_bird"`
	PlacedAt           time.Time  `json:"placed_at" gorm:"type:date;not null"`
	AgeAtPlacementDays int        `json:"age_at_placement_days" gorm:"not null;default:0"`
	ClosedAt           *time.Time `json:"closed_at" gorm:"type:date"`

	// Running totals from the movement log
	InitialBirdCount int     `json:"initial_bird_count" gorm:"not null;default:0"` // placed + moved in
	BirdCount        int     `json:"bird_count" gorm:"not null;default:0"`
	Deaths           int     `json:"deaths" gorm:"not null;default:0"`
	Culled           int     `json:"culled" gorm:"not null;default:0"`
	Sold             int     `json:"sold" gorm:"not null;default:0"`
	MortalityRate    float64 `json:"mortality_rate" gorm:"not null;default:0"` // deaths / initial, %

	Health float64 `json:"health" gorm:"not null;default:100"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Worked out on load, not stored
	HatchDate time.Time `json:"hatch_date" gorm:"-"`
	AgeDays   int       `json:"age_days" gorm:"-"`
	AgeWeeks  int       `json:"age_weeks" gorm:"-"`
	Stage     string    `json:"stage" gorm:"-"`
}

// AfterFind fills in the batch's age and stage
func (f *Flock) AfterFind(tx *gorm.DB) error {
	f.FillComputed(time.Now())
	return nil
}

// FillComputed works out hatch date, age and stage as of now (or as of closing, for a closed batch)
func (f *Flock) FillComputed(now time.Time) {
	f.HatchDate = f.PlacedAt.AddDate(0, 0, -f.AgeAtPlacementDays)
	asOf := now
	if f.ClosedAt != nil {
		asOf = *f.ClosedAt
	}
	f.AgeDays = DaysBetween(f.HatchDate, asOf)
	if f.AgeDays < 0 {
		f.AgeDays = 0
	}
	f.AgeWeeks = f.AgeDays / 7
	f.Stage = FlockStage(f.ProductionType, f.AgeDays, f.Status == FlockClosed)
}

// DaysBetween counts whole calendar days from a to b
func DaysBetween(a, b time.Time) int {
	da := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
	db := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, time.UTC)
	return int(db.Sub(da).Hours() / 24)
}

// FlockStage names where a batch is in its life, from its type and age in days
func FlockStage(productionType string, ageDays int, closed bool) string {
	if closed {
		return "closed"
	}
	switch productionType {
	case ProductionBroilers:
		if ageDays < 21 {
			return "chick"
		}
		return "finisher"
	case ProductionChicks:
		if ageDays < 42 {
			return "chick"
		}
		return "grower"
	default: // layers, breeders, kienyeji
		switch {
		case ageDays < 42:
			return "chick"
		case ageDays < 18*7:
			return "grower"
		case ageDays < 72*7:
			return "laying"
		default:
			return "spent"
		}
	}
}
