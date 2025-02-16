package models

import (
	"golang.org/x/crypto/bcrypt"
	
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
	ID           int    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int    `gorm:"uniqueIndex" json:"user_id"`
	Plan         string `json:"plan"`
	Status       string `json:"status"`
	ExpiryDate   string `json:"expiry_date"`
	PaymentMethod string `json:"payment_method"`
}

// User represents a system user
type User struct {
	ID             uint          `gorm:"primaryKey;autoIncrement" json:"id"`
	Username       string       `gorm:"unique;not null" json:"username"`
	Email          string       `gorm:"unique;not null" json:"email"`
	Password       string       `json:"password"`
	ProfilePicture string       `json:"profile_picture"`
	Contact        string       `json:"contact"`
	Subscription   Subscription `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"subscription"`
	BillingInfo    BillingInfo  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"billing_info"`
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


