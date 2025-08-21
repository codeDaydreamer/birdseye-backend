package models

import (
	"fmt"
	"log"
	"time"

	"birdseye-backend/pkg/db" // Ensure to import the db package

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// BillingInfo represents user billing details
type BillingInfo struct {
	ID         int    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int    `gorm:"uniqueIndex" json:"user_id"`
	CardNumber string `json:"card_number"`
	ExpiryDate string `json:"expiry_date"`
	CVV        string `json:"cvv"`
	Address    string `json:"address"`
	City       string `json:"city"`
	Country    string `json:"country"`
	ZipCode    string `json:"zip_code"`
}

// Subscription represents user subscription details
type Subscription struct {
	ID            int        `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        int        `gorm:"uniqueIndex" json:"user_id"`
	Plan          string     `json:"plan"`
	Status        string     `json:"status"`
	ExpiryDate    *time.Time `json:"expiry_date"`
	PaymentMethod string     `json:"payment_method"`
}

// User represents a system user
type User struct {
	ID             uint         `gorm:"primaryKey;autoIncrement" json:"id"`
	Username       string       `gorm:"unique;not null" json:"username"`
	Email          string       `gorm:"unique;not null" json:"email"`
	Password       string       `json:"password"`
	ProfilePicture string       `json:"profile_picture"`
	PhoneNumber    string       `json:"phone_number"`
	Role           string       `gorm:"default:user" json:"role"`
	LastLogin      *time.Time   `json:"last_login"`
	Status         string       `gorm:"default:active" json:"status"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`

	PaymentStatus  string       `gorm:"default:'unpaid'" json:"payment_status"`

	TrialEndsAt    time.Time    `json:"trial_ends_at"` // persistent
	IsTrialActive  bool         `json:"is_trial_active"`

	// New field for beta testers discount
	IsBetaTester   bool         `gorm:"default:true" json:"is_beta_tester"`
	DiscountedPrice int         `gorm:"default:1500" json:"discounted_price"` // 2500 by default, 1500 for beta testers

	OTP            string       `gorm:"-" json:"-"`
	OTPHashed      string       `json:"-"`
	OTPExpiresAt   *time.Time   `json:"-"`
	NeedsSubscriptionRenewal bool `gorm:"-" json:"needs_subscription_renewal"`
	EmailVerified  bool         `gorm:"default:false" json:"email_verified"`

	Subscription Subscription `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"subscription"`
	BillingInfo  BillingInfo  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"billing_info"`
}


// SetBetaPricing sets the discounted price for beta testers
func (u *User) SetBetaPricing() {
	if u.IsBetaTester {
		u.DiscountedPrice = 1500
	} else {
		u.DiscountedPrice = 2500
	}
}


// Admin represents an administrator user
type Admin struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"unique;not null" json:"username"`
	Email     string    `gorm:"unique;not null" json:"email"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Role      string    `gorm:"default:admin" json:"role"` // Default role is admin
}

// HashPassword hashes the password using bcrypt
func (u *User) HashPassword() error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword compares a hashed password with a plain password
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// GetUserByID retrieves a user by their ID from the database
func GetUserByID(userID uint) (*User, error) {
	var user User

	// Fetch the user from the database using the provided userID
	if err := db.DB.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user with ID %d not found", userID)
		}
		return nil, fmt.Errorf("error retrieving user: %w", err)
	}

	return &user, nil
}

// ComputeTrialEndsAt returns the trial end date (14 days for beta launch)
func (u *User) ComputeTrialEndsAt() time.Time {
	return u.CreatedAt.AddDate(0, 0, 14) // 14 days trial
}

// ComputeIsTrialActive checks if trial is still active
func (u *User) ComputeIsTrialActive() bool {
	return time.Now().Before(u.ComputeTrialEndsAt())
}
// UpdateTrialPeriods sets TrialEndsAt and IsTrialActive for users who don't have it set
func UpdateTrialPeriods() error {
	var users []User

	if err := db.DB.Find(&users).Error; err != nil {
		return err
	}

	for _, user := range users {
		if user.TrialEndsAt.IsZero() {
			trialEnds := user.ComputeTrialEndsAt()
			isActive := user.ComputeIsTrialActive()

			if err := db.DB.Model(&User{}).Where("id = ?", user.ID).
				Updates(map[string]interface{}{
					"trial_ends_at":   trialEnds,
					"is_trial_active": isActive,
				}).Error; err != nil {
				log.Printf("Failed to update trial for user %d: %v", user.ID, err)
			} else {
				log.Printf("Trial updated for user %d: ends at %v", user.ID, trialEnds)
			}
		}

		// Ensure beta pricing is applied
		user.SetBetaPricing()
		if err := db.DB.Model(&User{}).Where("id = ?", user.ID).
			Update("discounted_price", user.DiscountedPrice).Error; err != nil {
			log.Printf("Failed to update pricing for user %d: %v", user.ID, err)
		}
	}

	return nil
}
