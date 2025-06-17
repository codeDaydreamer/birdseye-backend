package middlewares

import (
	"errors"
	"time"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"birdseye-backend/pkg/models"
)

// Global variable for JWT secret
var JWTSecret string

// ErrTokenExpired is the error returned when the token has expired
var ErrTokenExpired = errors.New("token has expired")

// InitAuthMiddleware initializes the secret key globally
func InitAuthMiddleware() {
	JWTSecret = os.Getenv("JWT_SECRET")
	if JWTSecret == "" {
		panic("Error: JWT_SECRET is not set in environment variables.")
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

        // Get the user ID from the token
        user, err := GetUserFromToken(token)
        if err != nil {
            if errors.Is(err, ErrTokenExpired) {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expired, please log in again"})
            } else {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
            }
            c.Abort()
            return
        }

        // Attach user ID to context for later use
        c.Set("user_id", user.ID)
        c.Next()
    }
}


// GetUserFromToken extracts user data from JWT token and checks for expiration
func GetUserFromToken(tokenString string) (*models.User, error) {
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	if tokenString == "" {
		return nil, errors.New("empty token")
	}

	if JWTSecret == "" {
		return nil, errors.New("server misconfiguration")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	user := &models.User{}

	// Extract ID safely and convert it to uint if it's int
	if id, ok := claims["id"].(float64); ok {
		user.ID = uint(id) // Convert from float64 to uint
	} else if id, ok := claims["id"].(int); ok {
		user.ID = uint(id) // If it's an int, convert it to uint
	} else {
		return nil, errors.New("invalid user ID in token")
	}

// Check if token is expired
if exp, ok := claims["exp"].(float64); ok {
    if time.Now().After(time.Unix(int64(exp), 0)) {
        return nil, ErrTokenExpired // Token has expired
    }
} else {
    return nil, errors.New("missing exp claim in token")
}


	// Extract email
	if email, ok := claims["email"].(string); ok {
		user.Email = email
	}

	return user, nil
}

func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
			c.Abort()
			return
		}

		user, err := GetUserFromToken(token)
		if err != nil {
			if errors.Is(err, ErrTokenExpired) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expired, please log in again"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
			}
			c.Abort()
			return
		}

		if user.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		c.Set("user_id", user.ID)
		c.Next()
	}
}
