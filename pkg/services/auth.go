package services

import (
	"strconv"
	"net/url"
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"

	"birdseye-backend/pkg/services/email"

	"github.com/golang-jwt/jwt/v4"

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


func GoogleAuthCallback(code string) (string, *models.User, error) {
	ctx := context.Background()

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


	// Fetch user info
	userInfo, err := fetchGoogleUserInfo(token.AccessToken)
	if err != nil {
		log.Println("Fetching user info failed:", err)
		return "", nil, err
	}

	user, err := signInWithGoogleProfile(userInfo.Email, userInfo.Name, userInfo.Picture)
	if err != nil {
		return "", nil, err
	}

	// Generate JWT token
	tokenString, err := generateJWT(*user)
	if err != nil {
		log.Println("JWT token generation failed:", err)
		return "", nil, fmt.Errorf("error generating token: %w", err)
	}

	return tokenString, user, nil
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
	if length <= 0 {
		return "", errors.New("OTP length must be greater than zero")
	}

	const digits = "0123456789"
	bytes := make([]byte, length)

	for i := range bytes {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random digit: %w", err)
		}
		bytes[i] = digits[num.Int64()]
	}

	return string(bytes), nil
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
	// Hash the user's password
	if err := user.HashPassword(); err != nil {
		return nil, fmt.Errorf("error hashing password: %w", err)
	}

	// Insert user into database, as the owner of their own new farm
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		_, err := models.CreateFarmForUser(tx, user)
		return err
	}); err != nil {

		// Check if error is a MySQL duplicate entry error
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
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

	// Send OTP asynchronously
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

func GetUserByID(userID uint) (*models.User, error) {
	var user models.User

	// Fetch user by ID
	if err := db.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user with ID %d not found", userID)
		}
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
func UpdateUserProfile(userID uint, username, email, phonenumber string) (*models.User, error) { // userID is now uint
	var user models.User

	if err := db.DB.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Update user information
	user.Username = username
	user.Email = email
	user.PhoneNumber = phonenumber

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

// RequestPasswordReset generates and sends an OTP to the user's email for password reset
func RequestPasswordReset(emailAddr string) error {
	var user models.User
	if err := db.DB.Where("email = ?", emailAddr).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("no account found with this email")
		}
		return fmt.Errorf("database error: %w", err)
	}

	// Generate OTP
	otp, err := generateOTP(6) // Example: "492815"
	if err != nil {
		return fmt.Errorf("failed to generate OTP: %w", err)
	}

	// Hash OTP for storage
	hashedOTP, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash OTP: %w", err)
	}

	// Save OTP & expiry in DB
	expiry := time.Now().Add(15 * time.Minute)
	user.OTPHashed = string(hashedOTP)
	user.OTPExpiresAt = &expiry
	if err := db.DB.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to save OTP: %w", err)
	}

	// Send the OTP email (custom HTML email template)
	if err := email.SendPasswordResetOTP(user.Email, user.Username, otp); err != nil {
		return fmt.Errorf("failed to send password reset OTP email: %w", err)
	}

	return nil
}
// ResetPasswordWithOTP resets a user's password after verifying the provided OTP
func ResetPasswordWithOTP(emailAddr, otp, newPassword string) error {
	var user models.User
	if err := db.DB.Where("email = ?", emailAddr).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("no account found with this email")
		}
		return fmt.Errorf("database error: %w", err)
	}

	// Check if OTP is expired
	if user.OTPExpiresAt == nil || time.Now().After(*user.OTPExpiresAt) {
		return errors.New("OTP has expired")
	}

	// Verify OTP
	if err := bcrypt.CompareHashAndPassword([]byte(user.OTPHashed), []byte(otp)); err != nil {
		return errors.New("invalid OTP")
	}

	// Update password
	user.Password = newPassword
	if err := user.HashPassword(); err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Clear OTP fields after successful reset
	user.OTPHashed = ""
	user.OTPExpiresAt = nil

	if err := db.DB.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}


// signInWithGoogleProfile finds the account for a Google-verified email, or creates it (with its own
// farm) on first sign-in. Google has verified the email, so the account counts as verified.
func signInWithGoogleProfile(emailAddr, name, picture string) (*models.User, error) {
	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))
	var user models.User
	err := db.DB.Where("LOWER(email) = ?", emailAddr).First(&user).Error
	if err == nil {
		if !user.EmailVerified {
			db.DB.Model(&user).Update("email_verified", true)
			user.EmailVerified = true
		}
		return &user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Usernames are unique; Google names aren't
	username := strings.TrimSpace(name)
	if username == "" {
		username = strings.Split(emailAddr, "@")[0]
	}
	var taken int64
	db.DB.Model(&models.User{}).Where("username = ?", username).Count(&taken)
	if taken > 0 {
		username = fmt.Sprintf("%s-%04d", username, time.Now().UnixNano()%10000)
	}

	user = models.User{
		Username:       username,
		Email:          emailAddr,
		ProfilePicture: picture,
		Password:       "", // signs in with Google
		EmailVerified:  true,
	}
	// Every new user starts as the owner of their own farm
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		_, err := models.CreateFarmForUser(tx, &user)
		return err
	}); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	go func() {
		if err := email.SendWelcomeEmail(user.Email, user.Username); err != nil {
			log.Printf("failed to send welcome email to %s: %v", user.Email, err)
		}
	}()
	return &user, nil
}

// googleAudiences are the OAuth client IDs whose ID tokens we accept: the web client (which the
// Android app uses as its serverClientId) plus any in GOOGLE_ALLOWED_CLIENT_IDS
func googleAudiences() map[string]bool {
	allowed := map[string]bool{}
	for _, id := range append([]string{os.Getenv("GOOGLE_CLIENT_ID")}, strings.Split(os.Getenv("GOOGLE_ALLOWED_CLIENT_IDS"), ",")...) {
		if id = strings.TrimSpace(id); id != "" {
			allowed[id] = true
		}
	}
	return allowed
}

// googleTokenInfoURL is Google's ID token check (GOOGLE_TOKENINFO_URL overrides it for local testing)
var googleTokenInfoURL = "https://oauth2.googleapis.com/tokeninfo"

// GoogleSignInWithIDToken signs in with an ID token from Google Sign-In on a phone: Google checks
// the token's signature and expiry, and we check it was issued for Birdseye and the email is verified.
func GoogleSignInWithIDToken(idToken string) (string, *models.User, error) {
	if strings.TrimSpace(idToken) == "" {
		return "", nil, errors.New("missing Google ID token")
	}
	checkURL := googleTokenInfoURL
	if override := os.Getenv("GOOGLE_TOKENINFO_URL"); override != "" {
		checkURL = override // local testing only
	}
	resp, err := http.Get(checkURL + "?id_token=" + url.QueryEscape(idToken))
	if err != nil {
		return "", nil, fmt.Errorf("couldn't reach Google: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil, ErrGoogleTokenInvalid
	}

	var info struct {
		Aud           string `json:"aud"`
		Iss           string `json:"iss"`
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Exp           string `json:"exp"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", nil, ErrGoogleTokenInvalid
	}
	exp, _ := strconv.ParseInt(info.Exp, 10, 64)
	switch {
	case !googleAudiences()[info.Aud],
		info.Iss != "accounts.google.com" && info.Iss != "https://accounts.google.com",
		info.EmailVerified != "true" || info.Email == "",
		exp < time.Now().Unix():
		return "", nil, ErrGoogleTokenInvalid
	}

	user, err := signInWithGoogleProfile(info.Email, info.Name, info.Picture)
	if err != nil {
		return "", nil, err
	}
	now := time.Now()
	db.DB.Model(user).Update("last_login", now)

	token, err := generateJWT(*user)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

// ErrGoogleTokenInvalid means the Google ID token was rejected
var ErrGoogleTokenInvalid = errors.New("Google sign-in couldn't be verified, please try again")
