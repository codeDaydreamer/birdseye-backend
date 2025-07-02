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



// AdminUpdateUserByID updates a user identified by string ID with new values
func AdminUpdateUserByID(idStr string, updates map[string]interface{}) (*models.User, error) {
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

	// Remove fields that should never be updated here
	delete(updates, "created_at")
	delete(updates, "updated_at")
	delete(updates, "password") // password update handled separately

	// Optionally remove ID if sent
	delete(updates, "id")

	if err := db.DB.Model(user).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Reload user after update
	if err := db.DB.First(user, uint(id)).Error; err != nil {
		return nil, err
	}

	// Hide password before returning
	user.Password = ""

	return user, nil
}


func GetAdminByID(id uint) (*models.Admin, error) {
	var admin models.Admin
	if err := db.DB.First(&admin, id).Error; err != nil {
		return nil, err
	}
	return &admin, nil
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

// RegisterAdmin creates a new admin account with hashed password
func RegisterAdmin(admin *models.Admin) (*models.Admin, error) {
	// Check for existing admin
	var existingAdmin models.Admin
	if err := db.DB.Where("email = ? OR username = ?", admin.Email, admin.Username).First(&existingAdmin).Error; err == nil {
		return nil, errors.New("admin with this email or username already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(admin.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	admin.Password = string(hashedPassword)

	admin.Role = "admin"

	// Save to DB
	if err := db.DB.Create(admin).Error; err != nil {
		return nil, err
	}

	admin.Password = "" // remove password before return
	return admin, nil
}

// LoginAdmin authenticates admin and returns JWT token
func LoginAdmin(identifier, password string) (string, *models.Admin, error) {
	var admin models.Admin

	// Look up by email or username
	if err := db.DB.Where("email = ? OR username = ?", identifier, identifier).First(&admin).Error; err != nil {
		return "", nil, errors.New("invalid email/username or password")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err != nil {
		return "", nil, errors.New("invalid email/username or password")
	}

	// Generate JWT
	tokenString, err := generateAdminJWT(admin.ID, admin.Username)
	if err != nil {
		return "", nil, err
	}

	admin.Password = "" // remove password before returning
	return tokenString, &admin, nil
}

// generateAdminJWT generates a JWT token with admin_id and role
func generateAdminJWT(adminID uint, username string) (string, error) {
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		return "", errors.New("JWT_SECRET environment variable not set")
	}

	claims := jwt.MapClaims{
		"admin_id": adminID,
		"username": username,
		"role":     "admin",                               
		"exp":      time.Now().Add(72 * time.Hour).Unix(), // 3 days
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}
// AdminCreateUser allows an admin to create a new user with optional default role
func AdminCreateUser(user *models.User) (*models.User, error) {
	// Check if user with email or username already exists
	var existing models.User
	if err := db.DB.Where("email = ? OR username = ?", user.Email, user.Username).First(&existing).Error; err == nil {
		return nil, errors.New("user with this email or username already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user.Password = string(hashedPassword)

	// Set default role if missing
	if user.Role == "" {
		user.Role = "user"
	}

	if err := db.DB.Create(user).Error; err != nil {
		return nil, err
	}

	// Don't return password
	user.Password = ""
	return user, nil
}
func AdminResetUserPassword(userIDStr, newPassword string) error {
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return errors.New("invalid user ID")
	}

	var user models.User
	if err := db.DB.First(&user, uint(userID)).Error; err != nil {
		return errors.New("user not found")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("failed to hash password")
	}

	user.Password = string(hashedPassword)

	// Only update the Password field explicitly, no other fields get changed
	if err := db.DB.Model(&user).Update("password", user.Password).Error; err != nil {
		return errors.New("failed to update password")
	}

	return nil
}
