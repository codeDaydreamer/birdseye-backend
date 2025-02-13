package models

import (
	"time"
)

// EggProduction represents the egg production data in the database
type EggProduction struct {
	ID            uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID        uint      `json:"user_id" gorm:"index;not null"` 
	FlockID       uint      `json:"flock_id" gorm:"index;not null"`
	EggsCollected int       `json:"eggs_collected" gorm:"not null"`
	PricePerUnit  float64   `json:"price_per_unit" gorm:"not null;default:0"` // Price per egg
	TotalRevenue  float64   `json:"total_revenue" gorm:"-"`                    // Computed in service
	DateProduced  time.Time `json:"date_produced" gorm:"not null;type:date"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}
