package models

import (
	"golang.org/x/crypto/bcrypt"
)

// User model
type User struct {
	ID              int    `json:"id"`
	Username        string `json:"username"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ProfilePicture  string `json:"profile_picture"`  // Store URL/path for profile picture
	Contact         string `json:"contact"`         // User's contact number
	Subscription    SubscriptionDetails `json:"subscription"`  // Subscription information
	PaymentMethod   string `json:"paymentMethod"`   // Payment method, e.g., "Visa"
	BillingInfo     BillingInfo `json:"billingInfo"` // Billing information including card details
}

// SubscriptionDetails contains subscription-related information
type SubscriptionDetails struct {
	Plan        string `json:"plan"`        // Subscription plan name
	BillingDate string `json:"billingDate"` // Billing date
	NextPayment string `json:"nextPayment"` // Next payment date
}

// BillingInfo contains the user's virtual card information
type BillingInfo struct {
	CardNumber    string `json:"cardNumber"`    // Last 4 digits of the card number
	ExpirationDate string `json:"expirationDate"` // Expiration date of the card
	CVV            string `json:"cvv"`            // CVV (Not returned in full, for security reasons)
	PaymentMethod  string `json:"paymentMethod"`  // Payment method used (e.g., Visa)
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

// HashPasswordWithNewPassword hashes the user's new password
func (u *User) HashPasswordWithNewPassword(newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}
