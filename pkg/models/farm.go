package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Farm roles
const (
	FarmRoleOwner   = "owner"
	FarmRoleManager = "manager"
	FarmRoleWorker  = "worker"
)

// Farm is the tenant: every domain record belongs to a Farm, and Users access it via FarmMember
type Farm struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string    `gorm:"not null" json:"name"`
	OwnerUserID   uint      `gorm:"index;not null" json:"owner_user_id"`
	Status        string    `gorm:"default:active" json:"status"`          // active | suspended
	PaymentStatus string    `gorm:"default:'trial'" json:"payment_status"` // trial | paid | unpaid (see services.GetFarmBillingStatus)
	TrialEndsAt   time.Time `json:"trial_ends_at"`

	// Filled in by the owner's setup wizard; SetupCompletedAt stays nil until they finish or skip it
	Location         string     `gorm:"type:varchar(120)" json:"location"`
	FarmType         string     `gorm:"type:varchar(20)" json:"farm_type"` // layers | broilers | kienyeji | mixed
	SetupCompletedAt *time.Time `json:"setup_completed_at"`

	// Current plan and what's been paid for; a successful payment moves PaidUntil forward
	PlanID        *uint      `json:"plan_id"`
	BillingPeriod string     `gorm:"type:varchar(10);default:monthly" json:"billing_period"` // monthly | yearly
	PaidUntil     *time.Time `json:"paid_until"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	Subscription Subscription `gorm:"foreignKey:FarmID;constraint:OnDelete:CASCADE" json:"subscription"`
	BillingInfo  BillingInfo  `gorm:"foreignKey:FarmID;constraint:OnDelete:CASCADE" json:"billing_info"`
}

// FarmMember links a User to a Farm with a role (unique per farm/user pair)
type FarmMember struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	FarmID    uint      `gorm:"uniqueIndex:idx_farm_user;not null" json:"farm_id"`
	UserID    uint      `gorm:"uniqueIndex:idx_farm_user;not null" json:"user_id"`
	Role      string    `gorm:"not null;default:worker" json:"role"` // owner | manager | worker
	InvitedBy *uint     `json:"invited_by,omitempty"`
	Status    string    `gorm:"default:active" json:"status"` // active | removed
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FarmInvite is a token-based staff invite, usable even by people without an account yet
type FarmInvite struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	FarmID    uint      `gorm:"index;not null" json:"farm_id"`
	Email     string    `gorm:"type:varchar(255);index;not null" json:"email"`
	Role      string    `gorm:"type:varchar(20);not null" json:"role"`
	Token     string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"`
	InvitedBy uint      `json:"invited_by"`
	Status    string    `gorm:"default:pending" json:"status"` // pending | accepted | revoked | expired
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// FarmTypes are the kinds of poultry operation a farm can describe itself as
var FarmTypes = map[string]bool{"layers": true, "broilers": true, "kienyeji": true, "mixed": true}

// CreateFarmForUser creates a new Farm owned by the user, plus their owner FarmMember row
func CreateFarmForUser(tx *gorm.DB, user *User) (*Farm, error) {
	// New farms start on the default plan, with the trial the admin has configured
	trialEnds := time.Now()
	if settings, err := GetPlatformSettings(); err == nil && settings.TrialEnabled {
		trialEnds = trialEnds.AddDate(0, 0, settings.TrialDays)
	}
	farm := Farm{
		Name:          fmt.Sprintf("%s's Farm", user.Username),
		OwnerUserID:   user.ID,
		PaymentStatus: "trial",
		TrialEndsAt:   trialEnds,
		BillingPeriod: BillingMonthly,
	}
	if plan := GetDefaultPlan(tx); plan != nil {
		farm.PlanID = &plan.ID
	}
	if err := tx.Omit("Subscription", "BillingInfo").Create(&farm).Error; err != nil {
		return nil, err
	}
	if err := tx.Create(&FarmMember{FarmID: farm.ID, UserID: user.ID, Role: FarmRoleOwner}).Error; err != nil {
		return nil, err
	}
	return &farm, nil
}
