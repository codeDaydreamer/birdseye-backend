package middlewares

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// ParseUserToken validates a user JWT and returns its user ID. Only user tokens ("id" claim)
// are accepted, never admin tokens ("admin_id"), since user IDs and admin IDs overlap.
func ParseUserToken(tokenString string) (uint, error) {
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	if tokenString == "" {
		return 0, errors.New("empty token")
	}
	if JWTSecret == "" {
		return 0, errors.New("server misconfiguration")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(JWTSecret), nil
	})
	if err != nil {
		var vErr *jwt.ValidationError
		if errors.As(err, &vErr) && vErr.Errors&jwt.ValidationErrorExpired != 0 {
			return 0, ErrTokenExpired
		}
		return 0, errors.New("invalid token")
	}
	if !token.Valid {
		return 0, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("invalid token claims")
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return 0, errors.New("missing exp claim in token")
	}
	if time.Now().After(time.Unix(int64(exp), 0)) {
		return 0, ErrTokenExpired
	}

	id, ok := claims["id"].(float64)
	if !ok || id <= 0 {
		return 0, errors.New("invalid user ID in token")
	}
	return uint(id), nil
}

// UserAuth authenticates a regular user (never an admin token) for routes that aren't farm-scoped,
// such as the user's own profile. Sets user_id on the context.
func UserAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
			c.Abort()
			return
		}

		userID, err := ParseUserToken(token)
		if err != nil {
			if errors.Is(err, ErrTokenExpired) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expired, please log in again"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
			}
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

// ErrFarmRequired means the user belongs to several farms and didn't say which one
var ErrFarmRequired = errors.New("farm_id is required for users with more than one farm")

// ErrNotFarmMember means the user has no active membership in the requested farm
var ErrNotFarmMember = errors.New("not a member of this farm")

// ResolveFarmMember finds the user's active membership in the requested farm, or in their only
// farm when none is requested. It reads the DB every time so staff removals apply immediately.
func ResolveFarmMember(userID uint, requestedFarmID string) (*models.FarmMember, error) {
	var members []models.FarmMember
	query := db.DB.Where("user_id = ? AND status = ?", userID, "active")

	if requestedFarmID != "" {
		farmID, err := strconv.ParseUint(requestedFarmID, 10, 64)
		if err != nil {
			return nil, ErrNotFarmMember
		}
		query = query.Where("farm_id = ?", farmID)
	}

	if err := query.Limit(2).Find(&members).Error; err != nil {
		return nil, err
	}

	switch len(members) {
	case 0:
		return nil, ErrNotFarmMember
	case 1:
		return &members[0], nil
	default:
		return nil, ErrFarmRequired
	}
}

// FarmAuth authenticates the user and resolves which farm the request acts on, from the
// X-Farm-ID header or ?farm_id= query param, else the user's single farm.
// Sets user_id, farm_id and farm_role on the context.
func FarmAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
			c.Abort()
			return
		}

		userID, err := ParseUserToken(token)
		if err != nil {
			if errors.Is(err, ErrTokenExpired) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expired, please log in again"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
			}
			c.Abort()
			return
		}

		requested := c.GetHeader("X-Farm-ID")
		if requested == "" {
			requested = c.Query("farm_id")
		}

		member, err := ResolveFarmMember(userID, requested)
		if err != nil {
			switch {
			case errors.Is(err, ErrFarmRequired):
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			case errors.Is(err, ErrNotFarmMember):
				c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this farm"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve farm"})
			}
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("farm_id", member.FarmID)
		c.Set("farm_role", member.Role)
		c.Next()
	}
}

// RequireFarmRole allows the request only if FarmAuth resolved one of the given roles.
// Must be chained after FarmAuth.
func RequireFarmRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("farm_role")
		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient farm role"})
		c.Abort()
	}
}
