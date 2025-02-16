package services

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"errors"
	"log"
	"gorm.io/gorm"
)

// SubscriptionService handles subscription-related operations
type SubscriptionService struct{}
func (s *SubscriptionService) GetSubscription(userID uint) (*models.Subscription, error) {
	log.Println("🔍 Checking DB for user subscription:", userID)

	var subscription models.Subscription
	err := db.DB.Where("user_id = ?", userID).First(&subscription).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		log.Println("ℹ️ No subscription found for user ID:", userID)
		return nil, nil // Return nil to indicate no subscription yet
	} else if err != nil {
		log.Println("⚠️ Database error while retrieving subscription:", err)
		return nil, errors.New("failed to fetch subscription")
	}

	log.Println("✅ Subscription retrieved:", subscription)
	return &subscription, nil
}



// AddSubscription creates a new subscription
func (s *SubscriptionService) AddSubscription(subscription *models.Subscription) error {
	if err := db.DB.Create(subscription).Error; err != nil {
		return errors.New("failed to create subscription")
	}
	return nil
}

// UpdateSubscription updates an existing subscription
func (s *SubscriptionService) UpdateSubscription(subscription *models.Subscription) error {
	if err := db.DB.Save(subscription).Error; err != nil {
		return errors.New("failed to update subscription")
	}
	return nil
}

// DeleteSubscription removes a user's subscription
func (s *SubscriptionService) DeleteSubscription(userID uint) error {
	if err := db.DB.Where("user_id = ?", userID).Delete(&models.Subscription{}).Error; err != nil {
		return errors.New("failed to delete subscription")
	}
	return nil
}

// BillingService handles billing info operations
type BillingService struct{}

func (b *BillingService) GetBillingInfo(userID uint) (*models.BillingInfo, error) {
	log.Println("🔍 Fetching billing info for user ID:", userID)

	var billing models.BillingInfo
	err := db.DB.Where("user_id = ?", userID).First(&billing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		log.Println("ℹ️ No billing info found for user ID:", userID)
		return nil, nil // Return nil instead of an error to indicate absence
	} else if err != nil {
		log.Println("⚠️ Database error while retrieving billing info:", err)
		return nil, errors.New("failed to fetch billing info")
	}

	log.Println("✅ Billing info retrieved:", billing)
	return &billing, nil
}


// AddBillingInfo creates a new billing info record
func (b *BillingService) AddBillingInfo(billing *models.BillingInfo) error {
	if err := db.DB.Create(billing).Error; err != nil {
		return errors.New("failed to create billing info")
	}
	return nil
}

// UpdateBillingInfo updates a user's billing info
func (b *BillingService) UpdateBillingInfo(billing *models.BillingInfo) error {
	if err := db.DB.Save(billing).Error; err != nil {
		return errors.New("failed to update billing info")
	}
	return nil
}

// DeleteBillingInfo removes a user's billing info
func (b *BillingService) DeleteBillingInfo(userID uint) error {
	if err := db.DB.Where("user_id = ?", userID).Delete(&models.BillingInfo{}).Error; err != nil {
		return errors.New("failed to delete billing info")
	}
	return nil
}
