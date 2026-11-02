package models

import "time"

// AdminAuditEvent records a change an admin made in the console (every non-GET /admin request).
// Append-only: nothing in the app updates or deletes these.
type AdminAuditEvent struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AdminID    uint      `gorm:"index" json:"admin_id"`
	AdminEmail string    `gorm:"type:varchar(191)" json:"admin_email"`
	Method     string    `gorm:"type:varchar(8)" json:"method"`
	Path       string    `gorm:"type:varchar(255);index" json:"path"`
	Action     string    `gorm:"type:varchar(120);index" json:"action"` // e.g. "PUT /admin/user/:id"
	TargetID   string    `gorm:"type:varchar(64)" json:"target_id"`
	Status     int       `json:"status"`
	IP         string    `gorm:"type:varchar(64)" json:"ip"`
	UserAgent  string    `gorm:"type:varchar(300)" json:"user_agent"`
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
}
