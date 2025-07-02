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

var JWTSecret string
var ErrTokenExpired = errors.New("token has expired")

func InitAuthMiddleware() {
	JWTSecret = os.Getenv("JWT_SECRET")
	if JWTSecret == "" {
		panic("Error: JWT_SECRET is not set in environment variables.")
	}
}

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
			if errors.Is(err, ErrTokenExpired) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expired, please log in again"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
			}
			c.Abort()
			return
		}

		c.Set("user_id", user.ID)
		c.Set("role", user.Role)
		c.Next()
	}
}

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

	// ✅ Accept both "id" and "admin_id"
	var idRaw interface{}
	if idVal, ok := claims["id"]; ok {
		idRaw = idVal
	} else if adminID, ok := claims["admin_id"]; ok {
		idRaw = adminID
	} else {
		return nil, errors.New("invalid user ID in token")
	}

	switch v := idRaw.(type) {
	case float64:
		user.ID = uint(v)
	case int:
		user.ID = uint(v)
	default:
		return nil, errors.New("invalid user ID format")
	}

	// ✅ Check expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().After(time.Unix(int64(exp), 0)) {
			return nil, ErrTokenExpired
		}
	} else {
		return nil, errors.New("missing exp claim in token")
	}

	// Optional: Extract email
	if email, ok := claims["email"].(string); ok {
		user.Email = email
	}

	// ✅ Extract role
	if role, ok := claims["role"].(string); ok {
		user.Role = role
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
		c.Set("role", user.Role)
		c.Next()
	}
}
