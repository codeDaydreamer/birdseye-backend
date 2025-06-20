package api

import (
	
	"log"
	"net/http"
	"os"
	"fmt"
	"time"
	"github.com/gin-gonic/gin"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services"
	"golang.org/x/oauth2"
	"birdseye-backend/pkg/db"
)

// SetupRoutes sets up all the API routes
// SetupRoutes sets up all the API routes
func SetupRoutes(router *gin.Engine) {
	auth := router.Group("/auth")
	{
		// User Public routes
		auth.POST("/register", handleUserRegistration)
		auth.POST("/login", handleUserLogin)
		auth.GET("/google/callback", handleGoogleCallback)
		auth.GET("/google/login", handleGoogleLogin)

		// Protected user routes
		protected := auth.Group("/")
		protected.Use(middlewares.AuthMiddleware())
		protected.GET("/me", handleGetUserProfile)
		protected.PUT("/update-profile", handleUpdateProfile)
		protected.POST("/update-profile-picture", handleUpdateProfilePicture)
		protected.PUT("/change-password", handleChangePassword)

		// Admin Public routes (register & login)
		auth.POST("/admin/register", handleAdminRegistration)
		auth.POST("/admin/login", handleAdminLogin)
	}

	// Admin protected routes
	admin := router.Group("/admin")
	admin.Use(middlewares.AuthMiddleware(), middlewares.AdminAuthMiddleware())

	admin.GET("/users", handleAdminGetAllUsers)
	admin.GET("/user/:id", handleAdminGetUserByID)
	admin.PUT("/user/:id", handleAdminUpdateUser)
	admin.DELETE("/user/:id", handleAdminDeleteUser)
}


// ---------- HANDLER FUNCTIONS ----------
// Handle admin registration
func handleAdminRegistration(c *gin.Context) {
	var admin models.Admin  // Assuming you have a separate Admin model

	if err := c.ShouldBindJSON(&admin); err != nil {
		log.Printf("Error binding request body: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	registeredAdmin, err := services.RegisterAdmin(&admin)
	if err != nil {
		log.Printf("Error registering admin: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Registered Admin: %+v", registeredAdmin)
	c.JSON(http.StatusOK, gin.H{
		"message": "Admin registration successful.",
		"admin":   registeredAdmin,
	})
}

// Handle admin login
func handleAdminLogin(c *gin.Context) {
	var loginDetails struct {
		Identifier string `json:"identifier"` // email or username
		Password   string `json:"password"`
	}

	if err := c.ShouldBindJSON(&loginDetails); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	token, admin, err := services.LoginAdmin(loginDetails.Identifier, loginDetails.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Generated Admin Token: %s", token)
	log.Printf("Logged-in Admin: %+v", admin)

	c.JSON(http.StatusOK, gin.H{"token": token, "admin": admin})
}

// Handle user registration
func handleUserRegistration(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		log.Printf("Error binding request body: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	registeredUser, err := services.RegisterUser(&user)
	if err != nil {
		log.Printf("Error registering user: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Registered User: %+v", registeredUser)
	c.JSON(http.StatusOK, gin.H{
		"message": "Registration successful. Redirecting to homepage.",
		"user":    registeredUser,
	})
}

// Handle user login
func handleUserLogin(c *gin.Context) {
	var loginDetails struct {
		Identifier string `json:"identifier"` // Can be email or username
		Password   string `json:"password"`
	}

	if err := c.ShouldBindJSON(&loginDetails); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	token, user, err := services.LoginUser(loginDetails.Identifier, loginDetails.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

now := time.Now()

if err := db.DB.Model(&user).Update("last_login", now).Error; err != nil {
    log.Printf("Failed to update last login time: %v", err)
}



	log.Printf("Generated Token: %s", token)
	log.Printf("Logged-in User: %+v", user)

	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

func handleGoogleLogin(c *gin.Context) {
	config := services.GetGoogleOAuthConfig() // Get the OAuth config

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

func handleGoogleCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		log.Println("Authorization code missing")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization code missing"})
		return
	}

	log.Println("Received authorization code:", code)

	token, user, err := services.GoogleAuthCallback(code)
	if err != nil {
		log.Println("Error in GoogleAuthCallback:", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Log the token here to verify if it's being sent correctly
	log.Printf("Generated Google Auth Token: %s", token)

	log.Printf("Authenticated User: %+v\n", user) // Log user details

	// Get frontend URL from environment variable or use default
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173" // Change this if needed
	}

	// Redirect user to frontend with token
	redirectURL := fmt.Sprintf("http://localhost:5173/?token=%s",  token)
	log.Println("Redirecting to frontend with token:", redirectURL)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}



// Handle fetching the authenticated user’s profile
func handleGetUserProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, err := services.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching user from database"})
		return
	}

	log.Printf("User data fetched from database: %+v", user)

	sanitizedUser := gin.H{
		"id":             user.ID,
		"username":       user.Username,
		"email":          user.Email,
		"profile_picture": user.ProfilePicture,
		"contact":        user.Contact,
	}

	c.JSON(http.StatusOK, gin.H{"user": sanitizedUser})
}

// Handle updating user profile
func handleUpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, err := services.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	var updateRequest struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Contact  string `json:"contact"`
	}

	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	log.Printf("Updating user profile with: %+v", updateRequest)

	updatedUser, err := services.UpdateUserProfile(user.ID, updateRequest.Username, updateRequest.Email, updateRequest.Contact)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Updated User Profile: %+v", updatedUser)
	c.JSON(http.StatusOK, gin.H{"user": updatedUser})
}

// Handle updating profile picture
func handleUpdateProfilePicture(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, err := services.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	file, err := c.FormFile("profilePicture")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Profile picture file required"})
		return
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080/birdseye_backend"
	}

	uploadDir := "uploads/"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.Mkdir(uploadDir, os.ModePerm)
	}

	filePath := uploadDir + file.Filename
	profilePicturePath := baseURL + "/" + filePath

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload profile picture"})
		return
	}

	updatedUser, err := services.UpdateUserProfilePicture(user.ID, profilePicturePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": updatedUser})
}

// Handle changing user password
func handleChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, err := services.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	var changePasswordRequest struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := c.ShouldBindJSON(&changePasswordRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err = services.ChangePassword(user.ID, changePasswordRequest.CurrentPassword, changePasswordRequest.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}



// Admin handler examples

func handleAdminGetAllUsers(c *gin.Context) {
	users, err := services.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func handleAdminGetUserByID(c *gin.Context) {
	id := c.Param("id")
	user, err := services.GetUserByIDString(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func handleAdminUpdateUser(c *gin.Context) {
	id := c.Param("id")
	var updateData struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Contact  string `json:"contact"`
		// Add other fields as needed
	}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid update data"})
		return
	}
	updatedUser, err := services.AdminUpdateUserByID(id, updateData.Username, updateData.Email, updateData.Contact)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": updatedUser})
}

func handleAdminDeleteUser(c *gin.Context) {
	id := c.Param("id")
	err := services.AdminDeleteUserByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}