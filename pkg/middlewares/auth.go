package middlewares

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
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

// AuthMiddleware ensures requests are authenticated using JWT
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
			c.Abort()
			return
		}

		user, err := GetUserFromToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Store the authenticated user ID in the context
		c.Set("user_id", user.ID) // Ensures it's an integer
		c.Set("user", user)        // Stores full user struct if needed
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
		user.ID = int(id)
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
