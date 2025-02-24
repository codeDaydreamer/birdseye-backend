package models

import (
	"time"
)

// Notification struct for storing user notifications
type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"` // Associate notification with a specific user
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Type      string    `json:"type"`     // e.g., "info", "success", "error"
	URL       string    `json:"url,omitempty"` // Optional URL for redirection
	Read      bool      `json:"read" gorm:"default:false"` // Track if the notification is read
	CreatedAt time.Time `json:"created_at"`
}

// TableName overrides the default table name
func (Notification) TableName() string {
	return "notifications"
}
