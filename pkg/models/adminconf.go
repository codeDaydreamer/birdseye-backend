package models


import (
	"birdseye-backend/pkg/db"
	"gorm.io/gorm"
	
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
