package models

import (
	"encoding/json"
	"fmt"
	"time"

	"birdseye-backend/pkg/db" // Ensure to import the db package

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// BillingInfo holds a farm's billing details (one per farm)
type BillingInfo struct {
	ID     int  `gorm:"primaryKey;autoIncrement" json:"id"`
	FarmID uint `gorm:"uniqueIndex;not null" json:"farm_id"`
	// Opaque provider data only -- never store PAN/CVV
	CardLast4     string `gorm:"type:varchar(4)" json:"card_last4"`
	CardBrand     string `gorm:"type:varchar(30)" json:"card_brand"`
	ProviderToken string `json:"-"`
	Address       string `json:"address"`
	City          string `json:"city"`
	Country       string `json:"country"`
	ZipCode       string `json:"zip_code"`
}

// Subscription holds a farm's subscription details (one per farm)
type Subscription struct {
	ID            int        `gorm:"primaryKey;autoIncrement" json:"id"`
	FarmID        uint       `gorm:"uniqueIndex;not null" json:"farm_id"`
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


	OTP            string       `gorm:"-" json:"-"`
	OTPHashed      string       `json:"-"`
	OTPExpiresAt   *time.Time   `json:"-"`
	EmailVerified  bool         `gorm:"default:false" json:"email_verified"`
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


// MarshalJSON never exposes the password hash in API responses (Password still binds from JSON input)
func (u User) MarshalJSON() ([]byte, error) {
	type alias User
	return json.Marshal(&struct {
		*alias
		Password string `json:"password,omitempty"`
	}{alias: (*alias)(&u)})
}

// MarshalJSON never exposes the password hash in API responses (Password still binds from JSON input)
func (a Admin) MarshalJSON() ([]byte, error) {
	type alias Admin
	return json.Marshal(&struct {
		*alias
		Password string `json:"password,omitempty"`
	}{alias: (*alias)(&a)})
}
