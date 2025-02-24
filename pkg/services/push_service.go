package services

import (
    "birdseye-backend/pkg/models"
    "gorm.io/gorm"
)

type PushService struct {
    db *gorm.DB
}

func NewPushService(db *gorm.DB) *PushService {
    return &PushService{db: db}
}

// Save or update a push subscription
func (s *PushService) SaveSubscription(sub *models.PushSubscription) error {
    var existing models.PushSubscription
    if err := s.db.Where("user_id = ?", sub.UserID).First(&existing).Error; err == nil {
        // Update existing subscription
        existing.Endpoint = sub.Endpoint
        existing.P256dh = sub.P256dh
        existing.Auth = sub.Auth
        return s.db.Save(&existing).Error
    }

    // Save new subscription
    return s.db.Create(sub).Error
}
