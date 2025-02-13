package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Sale struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID      uint      `json:"user_id" gorm:"index;not null"`
	FlockID     uint      `json:"flock_id" gorm:"index;not null"` // Associate sales with a flock
	RefNo       string    `json:"ref_no" gorm:"type:varchar(50);unique;not null"`
	Product     string    `json:"product" gorm:"type:varchar(100);not null"`
	Category    string    `json:"category" gorm:"type:varchar(50);not null"` // e.g., "Eggs", "Feed", "Equipment"
	Description string    `json:"description" gorm:"type:varchar(255);not null"`
	Date        time.Time `json:"date" gorm:"not null"`
	Amount      float64   `json:"amount" gorm:"not null"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// GenerateRefNo generates a unique reference number for sales
func GenerateRefNo(userID uint) string {
	timestamp := time.Now().Format("20060102150405") // YYYYMMDDHHMMSS
	randomNum := fmt.Sprintf("%04d", time.Now().Nanosecond()%10000) // 4-digit random number
	return fmt.Sprintf("REF-%d-%s-%s", userID, timestamp, randomNum)
}

// Hook to generate RefNo before creating a sale
func (s *Sale) BeforeCreate(tx *gorm.DB) (err error) {
	s.RefNo = GenerateRefNo(s.UserID)
	return
}
