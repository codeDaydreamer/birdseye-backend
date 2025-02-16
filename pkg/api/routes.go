package api

import (
	"github.com/gin-gonic/gin"
	"birdseye-backend/pkg/services"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/middlewares"
	"net/http"
	"log"
	"os"
)

// SetupRoutes sets up all the API routes
func SetupRoutes(router *gin.Engine) {
	auth := router.Group("/auth")
	{
		// Public routes (No authentication middleware)
		auth.POST("/register", func(c *gin.Context) {
			var user models.User
			if err := c.ShouldBindJSON(&user); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
				log.Printf("Error binding request body: %v\n", err)
				return
			}

			registeredUser, err := services.RegisterUser(&user)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				log.Printf("Error registering user: %v\n", err)
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "Registration successful. Redirecting to homepage.",
				"user":    registeredUser,
			})
		})

		auth.POST("/login", func(c *gin.Context) {
			var loginDetails struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}

			if err := c.ShouldBindJSON(&loginDetails); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
				return
			}

			token, user, err := services.LoginUser(loginDetails.Email, loginDetails.Password)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"token": token,
				"user":  user,
			})
		})

		// Protected routes (Require authentication)
		protected := auth.Group("/")
		protected.Use(middlewares.AuthMiddleware()) // Apply auth middleware only to these routes

		// GET /me: Return the authenticated user's details
		protected.GET("/me", func(c *gin.Context) {
			user, exists := c.Get("user")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
				return
			}
			
			// Log the user details before sending to frontend
			log.Printf("User data to send to frontend: %+v", user)
			
			// Return only non-sensitive user details
			sanitizedUser := gin.H{
				"id":             user.(*models.User).ID,
				"username":       user.(*models.User).Username,
				"email":          user.(*models.User).Email,
				"profile_picture": user.(*models.User).ProfilePicture,
				"contact":        user.(*models.User).Contact,
			}
			
			// Log the sanitized data being sent to frontend
			log.Printf("Sanitized user data: %+v", sanitizedUser)
			
			c.JSON(http.StatusOK, gin.H{"user": sanitizedUser})
		})

		// PUT /update-profile: Update the user's profile
		protected.PUT("/update-profile", func(c *gin.Context) {
			user, exists := c.Get("user")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
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

			// Fetch the latest user data from the database to ensure fresh data
			updatedUser, err := services.UpdateUserProfile(user.(*models.User).ID, updateRequest.Username, updateRequest.Email, updateRequest.Contact)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"user": updatedUser})
		})

		// POST /update-profile-picture: Update the user's profile picture
		protected.POST("/update-profile-picture", func(c *gin.Context) {
			user, exists := c.Get("user")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
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
				os.Mkdir(uploadDir, os.ModePerm) // Ensure upload directory exists
			}

			filePath := uploadDir + file.Filename
			profilePicturePath := baseURL + "/" + filePath

			if err := c.SaveUploadedFile(file, filePath); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload profile picture"})
				return
			}

			// Fetch the latest user data before updating profile picture
			updatedUser, err := services.UpdateUserProfilePicture(user.(*models.User).ID, profilePicturePath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"user": updatedUser})
		})

		// PUT /change-password: Change the user's password
		protected.PUT("/change-password", func(c *gin.Context) {
			user, exists := c.Get("user")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
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

			err := services.ChangePassword(user.(*models.User).ID, changePasswordRequest.CurrentPassword, changePasswordRequest.NewPassword)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
		})
	}
}
