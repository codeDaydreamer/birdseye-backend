package models

import (
	"time"
)


// EggProduction represents the egg production data in the database
type EggProduction struct {
	ID            uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID        uint      `json:"user_id" gorm:"index;not null"`
	FlockID       uint      `json:"flock_id" gorm:"index;not null"`
	FlockName     string    `json:"flock_name" gorm:"column:flock_name"`  // Not stored in DB, fetched via join
	EggsCollected int       `json:"eggs_collected" gorm:"not null"`
	PricePerUnit  float64   `json:"price_per_unit" gorm:"not null;default:0"` // Price per egg
	TotalRevenue  float64   `json:"total_revenue" gorm:"-"`                    // Computed in service
	DateProduced  time.Time `json:"date_produced" gorm:"not null;type:date"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// EggAdjustment represents non-sale changes in egg count like giveaways or breakages
type EggAdjustment struct {
    ID              uint       `json:"id" gorm:"primaryKey;autoIncrement"`
    UserID          uint       `json:"user_id" gorm:"index;not null"`        // keep for auth association
    FlockID         uint       `json:"flock_id" gorm:"index;not null"`       // maps from frontend flock_id
    EggProductionID *uint      `json:"egg_production_id" gorm:"index"`       // optional
    Reason          string     `json:"adjustment_type" gorm:"type:varchar(100);not null"` // maps from frontend adjustment_type
    Quantity        int        `json:"eggs_adjusted" gorm:"not null"`        // maps from frontend eggs_adjusted
    Notes           string     `json:"notes,omitempty" gorm:"type:text"`
    DateAdjusted    time.Time  `json:"date_adjusted" gorm:"not null;type:date"`
    CreatedAt       time.Time  `json:"created_at" gorm:"autoCreateTime"`
    UpdatedAt       time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}
