package models

import "time"

// DeviceToken is a phone's Firebase Cloud Messaging registration token, so the mobile app gets push
// notifications. A token belongs to one user at a time: signing in as someone else on the phone moves it.
type DeviceToken struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Token     string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"-"`
	Platform  string    `gorm:"type:varchar(16);default:android" json:"platform"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
