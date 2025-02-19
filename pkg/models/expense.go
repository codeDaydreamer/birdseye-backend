package models

import (
	"time"
)

// Expense represents an expense entry in the database
type Expense struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID      uint      `json:"user_id" gorm:"index;not null"`
	FlockID     uint      `json:"flock_id" gorm:"not null;index"` // Foreign key reference to Flock
	Date        time.Time `json:"date" gorm:"not null;type:date"`
	Description string    `json:"description" gorm:"type:varchar(255);not null"`
	Amount      float64   `json:"amount" gorm:"not null"`
	Category    string    `json:"category" gorm:"type:varchar(50);not null"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Relationships
	Flock Flock `json:"flock" gorm:"foreignKey:FlockID"`
}


