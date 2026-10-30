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
	Status        string    `gorm:"default:active" json:"status"`           // active | suspended
	PaymentStatus string    `gorm:"default:'unpaid'" json:"payment_status"` // moved from User
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

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

// CreateFarmForUser creates a new Farm owned by the user, plus their owner FarmMember row
func CreateFarmForUser(tx *gorm.DB, user *User) (*Farm, error) {
	farm := Farm{
		Name:          fmt.Sprintf("%s's Farm", user.Username),
		OwnerUserID:   user.ID,
		PaymentStatus: user.PaymentStatus,
	}
	if err := tx.Omit("Subscription", "BillingInfo").Create(&farm).Error; err != nil {
		return nil, err
	}
	if err := tx.Create(&FarmMember{FarmID: farm.ID, UserID: user.ID, Role: FarmRoleOwner}).Error; err != nil {
		return nil, err
	}
	return &farm, nil
}
