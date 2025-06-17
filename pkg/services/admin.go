package services

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// GetAllUsers fetches all users from the database
func GetAllUsers() ([]models.User, error) {
	var users []models.User
	if err := db.DB.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// GetUserByIDString converts string ID to uint and fetches user by ID
func GetUserByIDString(idStr string) (*models.User, error) {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	return GetUserByID(uint(id))
}

// AdminUpdateUserByID updates the user identified by string ID with provided data
func AdminUpdateUserByID(idStr, username, email, contact string) (*models.User, error) {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	user := &models.User{}
	if err := db.DB.First(user, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	// Update fields - add more fields here if needed
	user.Username = username
	user.Email = email
	user.Contact = contact

	if err := db.DB.Save(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

// AdminDeleteUserByID deletes a user by string ID
func AdminDeleteUserByID(idStr string) error {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return errors.New("invalid user ID")
	}

	if err := db.DB.Delete(&models.User{}, uint(id)).Error; err != nil {
		return err
	}

	return nil
}

// RegisterAdmin registers a new admin user with hashed password
func RegisterAdmin(admin *models.Admin) (*models.Admin, error) {
	// Check if admin with same email or username exists
	var existingAdmin models.Admin
	if err := db.DB.Where("email = ? OR username = ?", admin.Email, admin.Username).First(&existingAdmin).Error; err == nil {
		return nil, errors.New("admin with this email or username already exists")
	}

	// Hash the password before saving
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(admin.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	admin.Password = string(hashedPassword)

	// Save admin to database
	if err := db.DB.Create(admin).Error; err != nil {
		return nil, err
	}

	// Remove password before returning
	admin.Password = ""
	return admin, nil
}

// LoginAdmin authenticates admin and returns a JWT token if successful
func LoginAdmin(identifier, password string) (string, *models.Admin, error) {
	var admin models.Admin

	// Find admin by email or username
	if err := db.DB.Where("email = ? OR username = ?", identifier, identifier).First(&admin).Error; err != nil {
		return "", nil, errors.New("invalid email/username or password")
	}

	// Compare hashed password with provided password
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err != nil {
		return "", nil, errors.New("invalid email/username or password")
	}

	// Generate JWT token
	tokenString, err := generateAdminJWT(admin.ID, admin.Username)
	if err != nil {
		return "", nil, err
	}

	// Remove password before returning
	admin.Password = ""

	return tokenString, &admin, nil
}

// generateAdminJWT creates a signed JWT token for the admin
func generateAdminJWT(adminID uint, username string) (string, error) {
	// Fetch secret key from environment variable
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		return "", errors.New("JWT_SECRET environment variable not set")
	}

	claims := jwt.MapClaims{
		"admin_id": adminID,
		"username": username,
		"exp":      time.Now().Add(72 * time.Hour).Unix(), // token expires in 72 hours
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}
