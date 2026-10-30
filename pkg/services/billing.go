package services

import (
	"errors"
	"time"

	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"

	"gorm.io/gorm"
)

const (
	// A successful payment keeps the farm paid for one month, plus a short grace period
	subscriptionPeriod = 30 * 24 * time.Hour
	paymentGracePeriod = 2 * 24 * time.Hour

	// StandardPrice is the monthly price in KES; beta testers pay their DiscountedPrice instead
	StandardPrice = 2500
)

// FarmBillingStatus is a farm's subscription state, derived from its trial and last payment
type FarmBillingStatus struct {
	PaymentStatus            string     `json:"payment_status"` // trial | paid | unpaid
	IsTrialActive            bool       `json:"is_trial_active"`
	TrialEndsAt              time.Time  `json:"trial_ends_at"`
	PaidUntil                *time.Time `json:"paid_until"`
	NeedsSubscriptionRenewal bool       `json:"needs_subscription_renewal"`
}

// GetFarmBillingStatus works out whether the farm is on trial, paid up, or needs to pay, and keeps
// Farm.PaymentStatus in sync so admin listings can filter on it.
func GetFarmBillingStatus(farmID uint) (*FarmBillingStatus, error) {
	var farm models.Farm
	if err := db.DB.Omit("Subscription", "BillingInfo").First(&farm, farmID).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	status := &FarmBillingStatus{
		TrialEndsAt:   farm.TrialEndsAt,
		IsTrialActive: now.Before(farm.TrialEndsAt),
	}

	var lastPayment models.Payment
	err := db.DB.Where("farm_id = ? AND status = ?", farmID, "success").
		Order("paid_at DESC").First(&lastPayment).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil {
		paidUntil := lastPayment.PaidAt.Time.Add(subscriptionPeriod)
		status.PaidUntil = &paidUntil
	}

	switch {
	case status.PaidUntil != nil && now.Before(status.PaidUntil.Add(paymentGracePeriod)):
		status.PaymentStatus = "paid"
	case status.IsTrialActive:
		status.PaymentStatus = "trial"
	default:
		status.PaymentStatus = "unpaid"
		status.NeedsSubscriptionRenewal = true
	}

	if farm.PaymentStatus != status.PaymentStatus {
		db.DB.Model(&models.Farm{}).Where("id = ?", farmID).Update("payment_status", status.PaymentStatus)
	}

	return status, nil
}

// SubscriptionPrice is what the farm pays per month: the owner's beta price while beta pricing is
// on, otherwise the standard price. Always set server-side, never taken from the client.
func SubscriptionPrice(farmID uint) (float64, error) {
	var farm models.Farm
	if err := db.DB.Omit("Subscription", "BillingInfo").First(&farm, farmID).Error; err != nil {
		return 0, err
	}

	var owner models.User
	if err := db.DB.First(&owner, farm.OwnerUserID).Error; err != nil {
		return 0, err
	}

	cfg, err := models.GetAdminConfig()
	if err != nil {
		return 0, err
	}

	if cfg.BetaPricingActive && owner.IsBetaTester && owner.DiscountedPrice > 0 {
		return float64(owner.DiscountedPrice), nil
	}
	return StandardPrice, nil
}

// GetFarmManagers returns the active owners and managers of a farm, who receive farm-wide emails
func GetFarmManagers(farmID uint) ([]models.User, error) {
	var users []models.User
	err := db.DB.
		Joins("JOIN farm_members ON farm_members.user_id = users.id").
		Where("farm_members.farm_id = ? AND farm_members.status = ? AND farm_members.role IN ?",
			farmID, "active", []string{models.FarmRoleOwner, models.FarmRoleManager}).
		Find(&users).Error
	return users, err
}

// GetOwnedFarm returns the farm the user owns (users start out owning exactly one)
func GetOwnedFarm(userID uint) (*models.Farm, error) {
	var farm models.Farm
	if err := db.DB.Omit("Subscription", "BillingInfo").
		Where("owner_user_id = ?", userID).Order("id").First(&farm).Error; err != nil {
		return nil, err
	}
	return &farm, nil
}
