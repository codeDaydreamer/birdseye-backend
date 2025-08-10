package models

import (
	"birdseye-backend/pkg/broadcast"
	"encoding/json"
	"gorm.io/gorm"
	"time"
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
	
	SalesData           json.RawMessage `json:"sales_data" gorm:"type:json"`
	
	// Relationships
	VaccinationSchedule []Vaccination   `json:"vaccination_schedule" gorm:"foreignKey:FlockID"`
	EggProductions []EggProduction `json:"egg_productions" gorm:"foreignKey:FlockID"`
	Sales               []Sale          `json:"sales" gorm:"foreignKey:FlockID"`  
	
	CreatedAt           time.Time       `json:"created_at"`  // Add this
    UpdatedAt           time.Time       `json:"updated_at"`



}

// BeforeSave: Ensure JSON fields are initialized
func (f *Flock) BeforeSave(tx *gorm.DB) error {
	var err error

	// Ensure JSON fields are set to "[]" if nil
	if f.MortalityRateData == nil {
		f.MortalityRateData = []byte("[]")
	}


	return err
}

// AfterFind: Ensure JSON fields are not nil after retrieval
func (f *Flock) AfterFind(tx *gorm.DB) error {
	if f.MortalityRateData == nil {
		f.MortalityRateData = []byte("[]")
	}

	return nil
}


func (f *Flock) AfterCreate(tx *gorm.DB) error {
	broadcast.SendFlockUpdate(f.UserID, "flock_added", *f)
	return nil
}

func (f *Flock) AfterUpdate(tx *gorm.DB) error {
	broadcast.SendFlockUpdate(f.UserID, "flock_updated", *f)
	return nil
}

func (f *Flock) AfterDelete(tx *gorm.DB) error {
	broadcast.SendFlockUpdate(f.UserID, "flock_deleted", f.ID)
	return nil
}


