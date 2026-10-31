package models

import (
	"time"

	"gorm.io/gorm"
)

// IncubationDays is how long chicken eggs take to hatch
const IncubationDays = 21

// Incubator is a machine eggs are set in
type Incubator struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	FarmID    uint      `json:"farm_id" gorm:"index;not null"`
	Name      string    `json:"name" gorm:"type:varchar(80);not null"`
	Capacity  int       `json:"capacity" gorm:"not null;default:0"` // eggs it holds
	Active    bool      `json:"active" gorm:"not null;default:true"`
	CreatedAt time.Time `json:"created_at"`
}

// Egg setting statuses
const (
	SettingIncubating = "incubating"
	SettingHatched    = "hatched"
	SettingClosed     = "closed" // failed or abandoned
)

// EggSetting is one batch of eggs set to hatch: from the farm's own birds or bought in, candled to remove
// clear (infertile) and rotten eggs, then hatched into chicks that go into a batch or are sold
type EggSetting struct {
	ID            uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	FarmID        uint       `json:"farm_id" gorm:"index;not null"`
	IncubatorID   *uint      `json:"incubator_id" gorm:"index"`
	Name          string     `json:"name" gorm:"type:varchar(80);not null"`
	SetDate       time.Time  `json:"set_date" gorm:"type:date;not null"`
	Source        string     `json:"source" gorm:"type:varchar(16);not null;default:own"` // own | bought
	SourceFlockID *uint      `json:"source_flock_id" gorm:"index"`                        // own eggs: the batch they came from
	Supplier      string     `json:"supplier" gorm:"type:varchar(120)"`
	CostPerEgg    float64    `json:"cost_per_egg"`
	ExpenseID     *uint      `json:"expense_id"` // bought eggs: their purchase
	EggsSet       int        `json:"eggs_set" gorm:"not null"`
	Clear         int        `json:"clear" gorm:"not null;default:0"`  // infertile, found at candling
	Rotten        int        `json:"rotten" gorm:"not null;default:0"` // died in the shell, found at candling
	CandledAt     *time.Time `json:"candled_at" gorm:"type:date"`
	Hatched       int        `json:"hatched" gorm:"not null;default:0"`
	Cull          int        `json:"cull" gorm:"not null;default:0"` // weak or deformed chicks
	HatchedAt     *time.Time `json:"hatched_at" gorm:"type:date"`
	ChickFlockID  *uint      `json:"chick_flock_id"` // the batch the chicks went into
	Status        string     `json:"status" gorm:"type:varchar(16);not null;default:incubating"`
	Notes         string     `json:"notes" gorm:"type:varchar(255)"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// Worked out on load
	Day            int       `json:"day" gorm:"-"` // day of incubation (1 = set day)
	DueDate        time.Time `json:"due_date" gorm:"-"`
	FertilityRate  float64   `json:"fertility_rate" gorm:"-"`   // (set - clear) / set, %
	HatchOfSet     float64   `json:"hatch_of_set" gorm:"-"`     // hatched / set, %
	HatchOfFertile float64   `json:"hatch_of_fertile" gorm:"-"` // hatched / (set - clear), %
	CostPerChick   float64   `json:"cost_per_chick" gorm:"-"`   // bought eggs only
	GoodChicks     int       `json:"good_chicks" gorm:"-"`      // hatched - cull
}

// FillComputed works out the setting's day, due date and rates as of now
func (s *EggSetting) FillComputed(now time.Time) {
	s.DueDate = s.SetDate.AddDate(0, 0, IncubationDays)
	end := now
	if s.HatchedAt != nil {
		end = *s.HatchedAt
	}
	s.Day = DaysBetween(s.SetDate, end) + 1
	s.FertilityRate, s.HatchOfSet, s.HatchOfFertile, s.CostPerChick = 0, 0, 0, 0
	if s.EggsSet > 0 && s.CandledAt != nil {
		s.FertilityRate = float64(s.EggsSet-s.Clear) / float64(s.EggsSet) * 100
	}
	s.GoodChicks = s.Hatched - s.Cull
	if s.HatchedAt != nil && s.EggsSet > 0 {
		s.HatchOfSet = float64(s.Hatched) / float64(s.EggsSet) * 100
		if fertile := s.EggsSet - s.Clear; fertile > 0 {
			s.HatchOfFertile = float64(s.Hatched) / float64(fertile) * 100
		}
		if s.CostPerEgg > 0 && s.GoodChicks > 0 {
			s.CostPerChick = s.CostPerEgg * float64(s.EggsSet) / float64(s.GoodChicks)
		}
	}
}

// Chick booking statuses
const (
	BookingBooked    = "booked"
	BookingFulfilled = "fulfilled"
	BookingCancelled = "cancelled"
)

// ChickBooking is a customer's order for chicks, often paid partly up front
type ChickBooking struct {
	ID            uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	FarmID        uint      `json:"farm_id" gorm:"index;not null"`
	Customer      string    `json:"customer" gorm:"type:varchar(120);not null"`
	Phone         string    `json:"phone" gorm:"type:varchar(20)"`
	Quantity      int       `json:"quantity" gorm:"not null"`
	WantedDate    time.Time `json:"wanted_date" gorm:"type:date;not null"`
	PricePerChick float64   `json:"price_per_chick"`
	Deposit       float64   `json:"deposit"`
	Status        string    `json:"status" gorm:"type:varchar(16);not null;default:booked"`
	SaleID        *uint     `json:"sale_id"` // the sale that fulfilled it
	Notes         string    `json:"notes" gorm:"type:varchar(255)"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Kinds of feed in the feed log
var FeedTypes = map[string]bool{"starter": true, "grower": true, "finisher": true, "layers_mash": true, "other": true}

// FeedLog is feed given to a batch; a cost files a Feed expense with it
type FeedLog struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	FarmID    uint      `json:"farm_id" gorm:"index;not null"`
	FlockID   uint      `json:"flock_id" gorm:"index;not null"`
	Date      time.Time `json:"date" gorm:"type:date;not null"`
	FeedType  string    `json:"feed_type" gorm:"type:varchar(16);not null"`
	Kg        float64   `json:"kg" gorm:"not null"`
	Cost      float64   `json:"cost"`
	ExpenseID *uint     `json:"expense_id"`
	// Taken from stock: the item and how many of its units (kg / kg_per_unit) were used
	InventoryItemID *uint     `json:"inventory_item_id"`
	UnitsUsed       float64   `json:"units_used"`
	Notes           string    `json:"notes" gorm:"type:varchar(255)"`
	CreatedAt       time.Time `json:"created_at"`
}

// WeighIn is the average weight of a sample of a batch's birds on a day
type WeighIn struct {
	ID         uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	FarmID     uint      `json:"farm_id" gorm:"index;not null"`
	FlockID    uint      `json:"flock_id" gorm:"index;not null"`
	Date       time.Time `json:"date" gorm:"type:date;not null"`
	SampleSize int       `json:"sample_size" gorm:"not null"`
	AvgWeightG float64   `json:"avg_weight_g" gorm:"not null"`
	CreatedAt  time.Time `json:"created_at"`
}

// AfterFind fills in the setting's day and rates
func (s *EggSetting) AfterFind(tx *gorm.DB) error {
	s.FillComputed(time.Now())
	return nil
}
