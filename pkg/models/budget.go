package models

import (
	"time"
)

// Budget represents a budget entry in the database with monthly support
type Budget struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    int      `json:"user_id" gorm:"index;not null"` // Foreign key reference to User
	FlockID   uint      `json:"flock_id" gorm:"index;not null"` // Foreign key reference to Flock
	Amount    float64   `json:"amount" gorm:"not null"` // The budget amount
	Month     int       `json:"month" gorm:"not null"`   // The month for the budget (1=January, 12=December)
	Year      int       `json:"year" gorm:"not null"`    // The year for the budget
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	
	Flock Flock `json:"flock" gorm:"foreignKey:FlockID"`
}
