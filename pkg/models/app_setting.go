package models

import "time"

// AppSetting is an integration key or setting saved from the admin console (Resend, VAPID, Firebase,
// Google sign-in, Dynapay). Value is AES-GCM encrypted; see pkg/appconfig. A row overrides the
// environment variable with the same name.
type AppSetting struct {
	Key       string    `gorm:"primaryKey;type:varchar(64)" json:"key"`
	Value     string    `gorm:"type:text;not null" json:"-"`
	UpdatedBy uint      `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
}
