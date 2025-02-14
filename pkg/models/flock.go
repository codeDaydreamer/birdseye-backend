package models

import (
	"time"
	"gorm.io/gorm"
)

// Flock represents a flock of birds in the farm
type Flock struct {
	ID                   uint            `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID               uint            `json:"user_id" gorm:"index;not null"`
	Name                 string          `json:"name" gorm:"not null"`
	Status               string          `json:"status" gorm:"not null"`
	InitialBirdCount     int             `json:"initial_bird_count" gorm:"not null"`
	BirdCount            int             `json:"bird_count" gorm:"not null"`
	Health               float64         `json:"health" gorm:"not null"`
	MortalityRate        float64         `json:"mortality_rate" gorm:"not null"`
	Breed                string          `json:"breed" gorm:"not null"`
	Age                  uint            `json:"age" gorm:"not null"`
	EggProduction        []int           `json:"egg_production" gorm:"type:json"`
	MortalityRateData    []int           `json:"mortality_rate_data" gorm:"type:json"`
	FeedConsumption      []int           `json:"feed_consumption" gorm:"type:json"`
	SalesData            []int           `json:"sales_data" gorm:"type:json"`
	FeedIntake           float64         `json:"feed_intake" gorm:"not null"`
	FeedQualityList      []FeedQuality   `json:"feed_quality_list" gorm:"type:json"`
	EggProduction7Days   []int           `json:"egg_production_7_days" gorm:"type:json"`
	EggProduction4Weeks  []int           `json:"egg_production_4_weeks" gorm:"type:json"`
	Revenue              float64         `json:"revenue" gorm:"not null"`
	Expenses             float64         `json:"expenses" gorm:"not null"`
	VaccinationSchedule  []Vaccination   `json:"vaccination_schedule" gorm:"foreignKey:FlockID"`
	EggProductions       []EggProduction `json:"egg_productions" gorm:"foreignKey:FlockID"`
	CreatedAt            time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt            time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
}

// FeedQuality represents attributes of feed quality
type FeedQuality struct {
	Text string `json:"text"`
	Icon string `json:"icon"`
}

// BeforeCreate GORM hook for JSON serialization
func (f *Flock) BeforeCreate(tx *gorm.DB) error {
	return nil
}

// BeforeUpdate GORM hook for JSON serialization
func (f *Flock) BeforeUpdate(tx *gorm.DB) error {
	return nil
}
