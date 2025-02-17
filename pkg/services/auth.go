package services

import (
	"errors"
	"fmt"
	"time"
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/middlewares"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
	"log"
)
var ErrTokenExpired = errors.New("token has expired")


// RegisterUser registers a new user in the database
func RegisterUser(user *models.User) (*models.User, error) {
	if err := user.HashPassword(); err != nil {
		return nil, fmt.Errorf("error hashing password: %w", err)
	}

	if result := db.DB.Create(user); result.Error != nil {
		return nil, fmt.Errorf("error inserting user into database: %w", result.Error)
	}

	return user, nil
}
// LoginUser authenticates a user using either email or username and returns a JWT token
func LoginUser(identifier, password string) (string, *models.User, error) {
	var user models.User

	// Try finding user by email or username
	if err := db.DB.Where("email = ? OR username = ?", identifier, identifier).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errors.New("invalid credentials")
		}
		return "", nil, fmt.Errorf("database error: %w", err)
	}

	// Verify password
	if !user.CheckPassword(password) {
		return "", nil, errors.New("invalid credentials")
	}

	// Generate JWT token
	token, err := generateJWT(user)
	if err != nil {
		return "", nil, fmt.Errorf("error generating token: %w", err)
	}

	return token, &user, nil
}

func generateJWT(user models.User) (string, error) {
	log.Printf("generateJWT: Using JWTSecret - %s", middlewares.JWTSecret)

	if middlewares.JWTSecret == "" {
		return "", errors.New("JWT secret is not set")
	}

	claims := jwt.MapClaims{
		"id":  user.ID,
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(), // 7 days expiry
	}
	

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(middlewares.JWTSecret))
}

// GetUserFromToken retrieves user details from the token and fetches fresh data from the database
func GetUserFromToken(tokenString string) (*models.User, error) {
    if middlewares.JWTSecret == "" {
        return nil, errors.New("JWT secret is not set")
    }

    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, errors.New("unexpected signing method")
        }
        return []byte(middlewares.JWTSecret), nil
    })

    if err != nil {
        return nil, fmt.Errorf("could not parse token: %w", err)
    }

    // Check if the token is valid
    if !token.Valid {
        return nil, errors.New("invalid token")
    }

    // Check for token expiration
    if claims, ok := token.Claims.(jwt.MapClaims); ok {
        if exp, ok := claims["exp"].(float64); ok {
            if time.Now().Unix() > int64(exp) {
                return nil, ErrTokenExpired // Token has expired
            }
        }
    }

    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return nil, errors.New("could not parse token claims")
    }

    // Extract the user ID from the claims
    user := &models.User{}
    if id, ok := claims["id"].(float64); ok {
        user.ID = uint(id) // Updated to uint
    } else {
        return nil, errors.New("invalid token claims: missing user ID")
    }

    // Fetch the latest user details from the database based on the user ID
    if err := db.DB.First(user, user.ID).Error; err != nil {
        return nil, fmt.Errorf("user not found in database: %w", err)
    }

    return user, nil
}
// GetUserByID retrieves a user by their ID from the database
func GetUserByID(userID uint) (*models.User, error) {
	var user models.User

	// Fetch the user from the database using the provided userID
	if err := db.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return a more specific error when the user is not found
			return nil, fmt.Errorf("user with ID %d not found", userID)
		}
		// Handle other types of errors (like DB connection issues)
		return nil, fmt.Errorf("error retrieving user: %w", err)
	}

	return &user, nil
}


// ChangePassword updates the user's password after verifying the current password
func ChangePassword(userID uint, currentPassword, newPassword string) error { // userID is now uint
	var user models.User

	if err := db.DB.First(&user, userID).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if !user.CheckPassword(currentPassword) {
		return errors.New("current password is incorrect")
	}

	user.Password = newPassword
	if err := user.HashPassword(); err != nil {
		return fmt.Errorf("error hashing new password: %w", err)
	}

	if err := db.DB.Save(&user).Error; err != nil {
		return fmt.Errorf("error updating password in database: %w", err)
	}

	return nil
}

// UpdateUserProfile updates user information
func UpdateUserProfile(userID uint, username, email, contact string) (*models.User, error) { // userID is now uint
	var user models.User

	if err := db.DB.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Update user information
	user.Username = username
	user.Email = email
	user.Contact = contact

	if err := db.DB.Save(&user).Error; err != nil {
		return nil, fmt.Errorf("error updating user profile: %w", err)
	}

	return &user, nil
}

// UpdateUserProfilePicture updates user's profile picture path
func UpdateUserProfilePicture(userID uint, profilePicturePath string) (*models.User, error) { // userID is now uint
	var user models.User

	if err := db.DB.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	user.ProfilePicture = profilePicturePath

	if err := db.DB.Save(&user).Error; err != nil {
		return nil, fmt.Errorf("error updating profile picture: %w", err)
	}

	return &user, nil
}
