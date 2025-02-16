package models

import (
	"birdseye-backend/pkg/broadcast"
	"encoding/json"
	"gorm.io/gorm"
)

// Flock represents a flock of birds in the farm
type Flock struct {
	ID                  uint            `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID              uint            `json:"user_id" gorm:"index;not null"`
	Name                string          `json:"name" gorm:"not null"`
	Status              string          `json:"status" gorm:"not null"`
	InitialBirdCount    int             `json:"initial_bird_count" gorm:"not null"`
	BirdCount           int             `json:"bird_count" gorm:"not null"`
	Health              float64         `json:"health" gorm:"not null"`
	MortalityRate       float64         `json:"mortality_rate" gorm:"not null"`
	Breed               string          `json:"breed" gorm:"not null"`
	Age                 uint            `json:"age" gorm:"not null"`
	FeedIntake          float64         `json:"feed_intake" gorm:"not null"`
	Revenue             float64         `json:"revenue" gorm:"not null"`
	Expenses            float64         `json:"expenses" gorm:"foreignKey:FlockID"`

	// JSON fields (Remove DEFAULT values)
	MortalityRateData   json.RawMessage `json:"mortality_rate_data" gorm:"type:json"`
	FeedConsumption     json.RawMessage `json:"feed_consumption" gorm:"type:json"`
	SalesData           json.RawMessage `json:"sales_data" gorm:"type:json"`
	EggProduction7Days  json.RawMessage `json:"egg_production_7_days" gorm:"type:json"`
	EggProduction4Weeks json.RawMessage `json:"egg_production_4_weeks" gorm:"type:json"`

	// Relationships
	VaccinationSchedule []Vaccination   `json:"vaccination_schedule" gorm:"foreignKey:FlockID"`
	EggProductions []EggProduction `json:"egg_productions" gorm:"foreignKey:FlockID"`
	Sales               []Sale          `json:"sales" gorm:"foreignKey:FlockID"`   



}

// BeforeSave: Ensure JSON fields are initialized
func (f *Flock) BeforeSave(tx *gorm.DB) error {
	var err error

	// Ensure JSON fields are set to "[]" if nil
	if f.MortalityRateData == nil {
		f.MortalityRateData = []byte("[]")
	}
	if f.FeedConsumption == nil {
		f.FeedConsumption = []byte("[]")
	}
	if f.SalesData == nil {
		f.SalesData = []byte("[]")
	}
	if f.EggProduction7Days == nil {
		f.EggProduction7Days = []byte("[]")
	}
	if f.EggProduction4Weeks == nil {
		f.EggProduction4Weeks = []byte("[]")
	}

	return err
}

// AfterFind: Ensure JSON fields are not nil after retrieval
func (f *Flock) AfterFind(tx *gorm.DB) error {
	if f.MortalityRateData == nil {
		f.MortalityRateData = []byte("[]")
	}
	if f.FeedConsumption == nil {
		f.FeedConsumption = []byte("[]")
	}
	if f.SalesData == nil {
		f.SalesData = []byte("[]")
	}
	if f.EggProduction7Days == nil {
		f.EggProduction7Days = []byte("[]")
	}
	if f.EggProduction4Weeks == nil {
		f.EggProduction4Weeks = []byte("[]")
	}
	return nil
}


// WebSocket event hooks
func (f *Flock) AfterCreate(tx *gorm.DB) error {
	broadcast.SendFlockUpdate("flock_added", *f)
	return nil
}

func (f *Flock) AfterUpdate(tx *gorm.DB) error {
	broadcast.SendFlockUpdate("flock_updated", *f)
	return nil
}

func (f *Flock) AfterDelete(tx *gorm.DB) error {
	broadcast.SendFlockUpdate("flock_deleted", f.ID)
	return nil
}
