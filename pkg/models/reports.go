package models

import (
	"time"
)

// Report represents a generated report
type Report struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ReportType  string    `json:"report_type" gorm:"type:varchar(100);not null"` // e.g., Sales, Inventory
	GeneratedAt time.Time `json:"generated_at" gorm:"type:datetime(3)"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	Content     string    `json:"content"`
	UserID      uint      `json:"user_id" gorm:"not null"` // Foreign key to associate with a specific user
	Name        string    `json:"name" gorm:"type:varchar(255);not null"` // Report filename (for user-friendly display)
	StartDate   time.Time `json:"start_date" gorm:"type:datetime(3)"` // Start date of the report range
	EndDate     time.Time `json:"end_date" gorm:"type:datetime(3)"`   // End date of the report range
}

// TableName overrides the default table name
func (Report) TableName() string {
	return "reports"
}
