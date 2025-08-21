package models


import (
	"birdseye-backend/pkg/db"
	"gorm.io/gorm"
	"log"
	
)
// AdminConfig stores toggleable settings
type AdminConfig struct {
	ID                uint `gorm:"primaryKey;autoIncrement"`
	BetaPricingActive bool `gorm:"default:true"` // toggle beta pricing
	BetaMaxUsers      int  `gorm:"default:50"`   // max beta users
}

func GetAdminConfig() (*AdminConfig, error) {
	var cfg AdminConfig
	if err := db.DB.First(&cfg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// create default if missing
			cfg = AdminConfig{BetaPricingActive: true, BetaMaxUsers: 50}
			if err := db.DB.Create(&cfg).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &cfg, nil
}
// MigrateAndSeedBetaFields auto-updates existing users with new fields
func MigrateAndSeedBetaFields() error {
	var users []User

	// Fetch all users
	if err := db.DB.Find(&users).Error; err != nil {
		return err
	}

	for _, user := range users {
		updates := map[string]interface{}{}

		// --- Trial computation ---
		if user.TrialEndsAt.IsZero() {
			trialEnds := user.ComputeTrialEndsAt()
			isActive := user.ComputeIsTrialActive()
			updates["trial_ends_at"] = trialEnds
			updates["is_trial_active"] = isActive
		}

		// --- Beta tester & discounted price ---
		// Default to beta tester if not explicitly set
		if !user.IsBetaTester {
			user.IsBetaTester = true
		}
		user.SetBetaPricing()
		updates["discounted_price"] = user.DiscountedPrice
		updates["is_beta_tester"] = user.IsBetaTester

		// Update user in DB
		if len(updates) > 0 {
			if err := db.DB.Model(&User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
				log.Printf("Failed to update user %d: %v", user.ID, err)
			} else {
				log.Printf("User %d updated: %+v", user.ID, updates)
			}
		}
	}

	return nil
}