package models

import (
	"time"
	
)

// Vaccination represents a vaccination record
type Vaccination struct {
	ID         uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	FlockID    uint      `json:"flock_id" gorm:"index;not null"`
	UserID     uint      `json:"user_id" gorm:"index;not null"` // To ensure multitenancy
	VaccineName string    `json:"vaccine_name" gorm:"not null"`
	Date       time.Time `json:"date" gorm:"not null"`
	Status     string    `json:"status" gorm:"not null"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}
