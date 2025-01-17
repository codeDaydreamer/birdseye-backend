package models

import (
	"golang.org/x/crypto/bcrypt"
	
)

type User struct {
	ID             int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Username       string `gorm:"unique;not null" json:"username"`
	Email          string `gorm:"unique;not null" json:"email"`
	Password       string `json:"password"`
	ProfilePicture string `json:"profile_picture"`
	Contact        string `json:"contact"`
	// Add other fields as necessary
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
