package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"birdseye-backend/pkg/db"

	"gorm.io/gorm"
)

// Billing periods a plan can be bought for
const (
	BillingMonthly = "monthly"
	BillingYearly  = "yearly"
)

// FeatureSet maps feature keys (see services.PlanFeatureCatalog) to whether a plan includes them
type FeatureSet map[string]bool

// Value stores the set as JSON
func (f FeatureSet) Value() (driver.Value, error) {
	if f == nil {
		return "{}", nil
	}
	b, err := json.Marshal(f)
	return string(b), err
}

// Scan reads the set from JSON
func (f *FeatureSet) Scan(value interface{}) error {
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	case nil:
		*f = FeatureSet{}
		return nil
	default:
		return errors.New("unsupported FeatureSet value")
	}
	out := FeatureSet{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			return err
		}
	}
	*f = out
	return nil
}

// Plan is a subscription package the admin defines: prices, limits and included features.
// A price of 0 means that billing period isn't offered; a limit of 0 means unlimited.
type Plan struct {
	ID           uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string     `gorm:"type:varchar(100);not null" json:"name"`
	Description  string     `gorm:"type:text" json:"description"`
	MonthlyPrice float64    `json:"monthly_price"`
	YearlyPrice  float64    `json:"yearly_price"`
	MaxStaff     int        `json:"max_staff"`  // active members incl. the owner
	MaxFlocks    int        `json:"max_flocks"` // flocks on the farm
	MaxBirds     int        `json:"max_birds"`  // birds across all flocks
	Features     FeatureSet `gorm:"type:json" json:"features"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`  // offered to farms
	IsDefault    bool       `gorm:"default:false" json:"is_default"` // given to new farms for their trial
	SortOrder    int        `json:"sort_order"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// PriceFor returns the plan's price for a billing period (0 if not offered)
func (p *Plan) PriceFor(period string) float64 {
	if period == BillingYearly {
		return p.YearlyPrice
	}
	return p.MonthlyPrice
}

// Promo code discount types
const (
	DiscountPercent = "percent"
	DiscountFixed   = "fixed"
)

// PromoCode is an admin-created discount owners can enter when paying
type PromoCode struct {
	ID              uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Code            string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"` // stored upper-case
	Description     string     `json:"description"`
	DiscountType    string     `gorm:"type:varchar(10);not null" json:"discount_type"` // percent | fixed
	DiscountValue   float64    `json:"discount_value"`                                 // % off, or KES off
	PlanID          *uint      `json:"plan_id"`                                        // only for this plan, if set
	ValidFrom       *time.Time `json:"valid_from"`
	ValidUntil      *time.Time `json:"valid_until"`
	MaxRedemptions  int        `json:"max_redemptions"` // 0 = unlimited
	RedemptionCount int        `json:"redemption_count"`
	IsActive        bool       `gorm:"default:true" json:"is_active"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// PlatformSettings holds the admin-controlled billing settings (a single row)
type PlatformSettings struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"-"`
	TrialEnabled    bool      `gorm:"default:true" json:"trial_enabled"`
	TrialDays       int       `gorm:"default:14" json:"trial_days"`
	GracePeriodDays int       `gorm:"default:2" json:"grace_period_days"` // after paid_until before access stops
	Currency        string    `gorm:"type:varchar(10);default:KES" json:"currency"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// GetPlatformSettings returns the settings row, creating it with defaults on first use
func GetPlatformSettings() (*PlatformSettings, error) {
	var s PlatformSettings
	err := db.DB.First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		s = PlatformSettings{TrialEnabled: true, TrialDays: 14, GracePeriodDays: 2, Currency: "KES"}
		err = db.DB.Create(&s).Error
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetDefaultPlan returns the plan new farms start on, or nil if none is marked default
func GetDefaultPlan(tx *gorm.DB) *Plan {
	var plan Plan
	if err := tx.Where("is_default = ? AND is_active = ?", true, true).Order("sort_order, id").First(&plan).Error; err != nil {
		return nil
	}
	return &plan
}
