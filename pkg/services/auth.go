package services

import (
	"fmt"
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"github.com/dgrijalva/jwt-go"
	"time"
	"strings"
	"errors"
	"gorm.io/gorm"
)

var jwtSecret = []byte("your_secret_key_here") // Secret key for signing JWT

// RegisterUser registers a new user in the MySQL database and returns the user details
func RegisterUser(user *models.User) (*models.User, error) {
	// Hash the user's password
	err := user.HashPassword()
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %v", err)
	}

	// Insert user into the database using GORM
	result := db.DB.Create(user)
	if result.Error != nil {
		return nil, fmt.Errorf("error inserting user into database: %v", result.Error)
	}

	// Return the user object after registration
	return user, nil
}

// LoginUser authenticates a user and returns a JWT and user details
func LoginUser(email, password string) (string, *models.User, error) {
	var user models.User

	// Query user by email using GORM
	result := db.DB.Where("email = ?", email).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return "", nil, fmt.Errorf("user not found")
		}
		return "", nil, fmt.Errorf("error querying user: %v", result.Error)
	}

	// Check if the password is correct
	if !user.CheckPassword(password) {
		return "", nil, fmt.Errorf("incorrect password")
	}

	// Generate JWT token
	token, err := generateJWT(user)
	if err != nil {
		return "", nil, fmt.Errorf("error generating token: %v", err)
	}

	return token, &user, nil
}

// generateJWT generates a JWT token for the user
func generateJWT(user models.User) (string, error) {
	// Create a new JWT token
	token := jwt.New(jwt.SigningMethodHS256)

	// Create claims (payload)
	claims := token.Claims.(jwt.MapClaims)
	claims["id"] = user.ID
	claims["username"] = user.Username
	claims["email"] = user.Email
	claims["exp"] = time.Now().Add(time.Hour * 168).Unix() // Expiration time (7 days)

	// Sign the token with the secret key
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("error signing the token: %v", err)
	}

	return tokenString, nil
}

// GetUserFromToken decodes the JWT token and retrieves the user
func GetUserFromToken(tokenString string) (*models.User, error) {
	// Remove "Bearer " prefix if it exists
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	// Parse and validate the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure the token is signed with the correct algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	// Extract the user from the token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("could not parse token claims")
	}

	// Create a user model from the claims
	user := &models.User{
		ID:       int(claims["id"].(float64)), // Type assertion to get the user ID
		Username: claims["username"].(string),
		Email:    claims["email"].(string),
	}

	return user, nil
}

// ChangePassword allows the user to change their password after verifying the current password
func ChangePassword(userID int, currentPassword, newPassword string) error {
	var user models.User
	result := db.DB.First(&user, userID)
	if result.Error != nil {
		return fmt.Errorf("user not found: %v", result.Error)
	}

	// Check if the current password matches the stored password
	if !user.CheckPassword(currentPassword) {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash the new password
	err := user.HashPassword()
	if err != nil {
		return fmt.Errorf("error hashing new password: %v", err)
	}

	// Update the user's password in the database using GORM
	result = db.DB.Save(&user)
	if result.Error != nil {
		return fmt.Errorf("error updating password in database: %v", result.Error)
	}

	return nil
}

// UpdateUserProfile updates the user's profile information
func UpdateUserProfile(userID int, username, email, contact string) (*models.User, error) {
	var user models.User
	result := db.DB.First(&user, userID)
	if result.Error != nil {
		return nil, fmt.Errorf("user not found: %v", result.Error)
	}

	// Update the user's profile
	user.Username = username
	user.Email = email
	user.Contact = contact

	// Save the updated user in the database
	result = db.DB.Save(&user)
	if result.Error != nil {
		return nil, fmt.Errorf("error updating user profile: %v", result.Error)
	}

	return &user, nil
}

// UpdateUserProfilePicture updates the user's profile picture path
func UpdateUserProfilePicture(userID int, profilePicturePath string) (*models.User, error) {
	var user models.User
	result := db.DB.First(&user, userID)
	if result.Error != nil {
		return nil, fmt.Errorf("user not found: %v", result.Error)
	}

	user.ProfilePicture = profilePicturePath
	result = db.DB.Save(&user)
	if result.Error != nil {
		return nil, fmt.Errorf("error updating profile picture: %v", result.Error)
	}

	return &user, nil
}
