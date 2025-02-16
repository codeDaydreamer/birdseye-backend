package middlewares

import (
	"errors"
	"net/http"
	"os"
	"strings"
    "fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
)

// Global variable for JWT secret
var JWTSecret string

// InitAuthMiddleware initializes the secret key and makes it available globally
func InitAuthMiddleware() {
	JWTSecret = os.Getenv("JWT_SECRET")

	if JWTSecret == "" {
		panic("Error: JWT_SECRET is not set in the environment variables.")
	}
}

// AuthMiddleware is used to authenticate requests using JWT tokens
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
			c.Abort()
			return
		}

		// Extract user from the token using GetUserFromToken
		user, err := GetUserFromToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		// Fetch fresh user data from the database to ensure we have up-to-date information
		freshUser, err := getUserByID(user.ID) // Directly using getUserByID function

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		// Set the fresh user data in the context
		c.Set("user", freshUser)
		c.Next()
	}
}

// GetUserFromToken extracts user data from JWT token
func GetUserFromToken(tokenString string) (*models.User, error) {
	// Remove "Bearer " prefix
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	if tokenString == "" {
		return nil, errors.New("empty token")
	}

	if JWTSecret == "" {
		return nil, errors.New("server misconfiguration")
	}

	// Parse and validate token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure correct signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	user := &models.User{}

	// Extract ID safely as integer
	if id, ok := claims["id"].(float64); ok {
		user.ID = uint(id)

	} else {
		return nil, errors.New("invalid token claims")
	}

	// Extract username
	if username, ok := claims["username"].(string); ok {
		user.Username = username
	}

	// Extract email
	if email, ok := claims["email"].(string); ok {
		user.Email = email
	}

	return user, nil
}

// getUserByID retrieves a user from the database by their ID (avoiding the services package)
func getUserByID(userID uint) (*models.User, error) {
	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user with ID %d not found", userID)
	}
	return &user, nil
}
