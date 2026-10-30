package services

import (
	"time"

	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
)

// FarmBillingStatus is a farm's subscription state, derived from its trial and what's been paid
type FarmBillingStatus struct {
	PaymentStatus            string     `json:"payment_status"` // trial | paid | unpaid
	IsTrialActive            bool       `json:"is_trial_active"`
	TrialEndsAt              time.Time  `json:"trial_ends_at"`
	PaidUntil                *time.Time `json:"paid_until"`
	BillingPeriod            string     `json:"billing_period"`
	NeedsSubscriptionRenewal bool       `json:"needs_subscription_renewal"`
}

// GetFarmBillingStatus works out whether the farm is on trial, paid up (including the admin's grace
// period), or needs to pay, and keeps Farm.PaymentStatus in sync so admin listings can filter on it.
func GetFarmBillingStatus(farmID uint) (*FarmBillingStatus, error) {
	var farm models.Farm
	if err := db.DB.Omit("Subscription", "BillingInfo").First(&farm, farmID).Error; err != nil {
		return nil, err
	}
	settings, err := models.GetPlatformSettings()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	status := &FarmBillingStatus{
		TrialEndsAt:   farm.TrialEndsAt,
		IsTrialActive: now.Before(farm.TrialEndsAt),
		PaidUntil:     farm.PaidUntil,
		BillingPeriod: farm.BillingPeriod,
	}

	grace := time.Duration(settings.GracePeriodDays) * 24 * time.Hour
	switch {
	case farm.PaidUntil != nil && now.Before(farm.PaidUntil.Add(grace)):
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
