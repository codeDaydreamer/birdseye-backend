package services

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
	"crypto/rand"
	"encoding/base64"

	"github.com/golang-jwt/jwt/v4"
	"birdseye-backend/pkg/services/email"
	"strings"

	"github.com/go-sql-driver/mysql" 
	
)
// Google OAuth configuration
var googleOAuthConfig *oauth2.Config

var ErrTokenExpired = errors.New("token has expired")


func InitGoogleOAuth() {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	

	if clientID == "" || clientSecret == "" || redirectURL == "" {
		log.Fatal("Missing required Google OAuth environment variables!")
	}

	googleOAuthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}
// GetGoogleOAuthConfig returns the Google OAuth configuration
func GetGoogleOAuthConfig() *oauth2.Config {
	if googleOAuthConfig == nil {
		log.Fatal("Google OAuth is not initialized! Call InitGoogleOAuth() first.")
	}
	return googleOAuthConfig
}

// GoogleAuthURL generates the Google OAuth login URL
func GoogleAuthURL() string {
	return googleOAuthConfig.AuthCodeURL("random-state-string", oauth2.AccessTypeOffline)
}


// GoogleAuthCallback handles Google's OAuth callback and logs in/registers the user
func GoogleAuthCallback(code string) (string, *models.User, error) {
	ctx := context.Background()

	// Debugging: Ensure the OAuth config is set
	if googleOAuthConfig == nil {
		log.Println("Google OAuth Config is nil!")
		return "", nil, errors.New("google oauth configuration is not initialized")
	}

	

	// Exchange the authorization code for an access token
	token, err := googleOAuthConfig.Exchange(ctx, code)
	if err != nil {
		log.Println("Token exchange failed:", err)
		return "", nil, fmt.Errorf("failed to exchange token: %w", err)
	}

	fmt.Println("Token successfully exchanged:", token)

	// Fetch user info
	userInfo, err := fetchGoogleUserInfo(token.AccessToken)
	if err != nil {
		log.Println("Fetching user info failed:", err)
		return "", nil, err
	}

	// Check if the user exists in the database
	var user models.User
	if err := db.DB.Where("email = ?", userInfo.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// New user, register them
			user = models.User{
				Username:       userInfo.Name,
				Email:          userInfo.Email,
				ProfilePicture: userInfo.Picture,
				Password:       "", // No password needed for Google sign-in
			}

			if err := db.DB.Create(&user).Error; err != nil {
				log.Println("User registration failed:", err)
				return "", nil, fmt.Errorf("failed to create user: %w", err)
			}
		} else {
			log.Println("Database error:", err)
			return "", nil, fmt.Errorf("database error: %w", err)
		}
		// Send welcome email asynchronously
	go func() {
		if err := email.SendWelcomeEmail(user.Email, user.Username); err != nil {
			log.Printf("failed to send welcome email to %s: %v", user.Email, err)
		}
	}()
	}

	// Generate JWT token
	tokenString, err := generateJWT(user)
	if err != nil {
		log.Println("JWT token generation failed:", err)
		return "", nil, fmt.Errorf("error generating token: %w", err)
	}

	fmt.Println("Successfully generated JWT token:", tokenString)
	return tokenString, &user, nil
}


// fetchGoogleUserInfo retrieves user information from Google
func fetchGoogleUserInfo(accessToken string) (*GoogleUser, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var userInfo GoogleUser
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return &userInfo, nil
}

// GoogleUser represents the structure of Google's user info response
type GoogleUser struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}
func generateOTP(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes)[:length], nil
}
func GenerateAndSendOTP(user *models.User) error {
	otp, err := generateOTP(6)
	if err != nil {
		return fmt.Errorf("failed to generate OTP: %w", err)
	}

	hashedOTP, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash OTP: %w", err)
	}

	expiry := time.Now().Add(15 * time.Minute)

	user.OTPHashed = string(hashedOTP)
	user.OTPExpiresAt = &expiry 

	if err := db.DB.Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user with OTP: %w", err)
	}

	go func() {
		if err := email.SendOTPEmail(user.Email, user.Username, otp); err != nil {
			log.Printf("failed to send OTP email to %s: %v", user.Email, err)
		}
	}()

	return nil
}





func RegisterUser(user *models.User) (*models.User, error) {
	if err := user.HashPassword(); err != nil {
		return nil, fmt.Errorf("error hashing password: %w", err)
	}

	if result := db.DB.Create(user); result.Error != nil {
		err := result.Error

		// Check if error is a MySQL duplicate entry error
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			// Inspect error message for which unique key failed
			errMsg := mysqlErr.Message
			if strings.Contains(errMsg, "uni_users_username") {
				return nil, errors.New("username is already taken")
			}
			if strings.Contains(errMsg, "uni_users_email") {
				return nil, errors.New("email is already registered")
			}
			return nil, errors.New("duplicate entry error")
		}

		return nil, fmt.Errorf("error inserting user into database: %w", err)
	}

	if err := GenerateAndSendOTP(user); err != nil {
		return nil, err
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
	now := time.Now()
user.LastLogin = &now

if err := db.DB.Save(&user).Error; err != nil {
    log.Printf("Warning: failed to update last login time for user %d: %v", user.ID, err)
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

func GetUserByID(userID uint) (*models.User, error) {
	var user models.User

	// Preload Subscription and BillingInfo associations
	if err := db.DB.
		Preload("Subscription").
		Preload("BillingInfo").
		First(&user, userID).Error; err != nil {
		
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user with ID %d not found", userID)
		}
		return nil, fmt.Errorf("error retrieving user: %w", err)
	}

	// Set computed trial fields for JSON response (assuming you added these fields in struct)
	user.TrialEndsAt = user.ComputeTrialEndsAt()
	user.IsTrialActive = user.ComputeIsTrialActive()


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
