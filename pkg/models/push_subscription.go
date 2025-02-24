package models

import "gorm.io/gorm"

type PushSubscription struct {
    gorm.Model
    UserID uint   `json:"user_id"` // Link to the user
    Endpoint string `json:"endpoint"`
    P256dh   string `json:"p256dh"`   // Public key for encryption
    Auth     string `json:"auth"`     // Authentication secret
}
