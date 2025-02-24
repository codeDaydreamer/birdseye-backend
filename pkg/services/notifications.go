package services

import (
	"birdseye-backend/pkg/models"
	
	

	"gorm.io/gorm"
)

// NotificationService handles database operations for notifications
type NotificationService struct {
	DB *gorm.DB
}

// NewNotificationService initializes a new NotificationService
func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{DB: db}
}

// CreateNotification stores a new notification in the database and returns it
func (s *NotificationService) CreateNotification(notification *models.Notification) error {
	if err := s.DB.Create(notification).Error; err != nil {
		return err
	}
	return nil
}

// GetNotificationsByUser retrieves all notifications for a specific user
func (s *NotificationService) GetNotificationsByUser(userID uint) ([]models.Notification, error) {
	var notifications []models.Notification
	err := s.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&notifications).Error
	return notifications, err
}

// MarkAsRead updates a notification's read status
func (s *NotificationService) MarkAsRead(notificationID uint, userID uint) error {
	return s.DB.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Update("read", true).Error
}

// MarkAllAsRead marks all notifications for a user as read
func (s *NotificationService) MarkAllAsRead(userID uint) error {
	return s.DB.Model(&models.Notification{}).
		Where("user_id = ?", userID).
		Update("read", true).Error
}

// DeleteNotification removes a notification from the database
func (s *NotificationService) DeleteNotification(notificationID uint, userID uint) error {
	return s.DB.Where("id = ? AND user_id = ?", notificationID, userID).Delete(&models.Notification{}).Error
}
