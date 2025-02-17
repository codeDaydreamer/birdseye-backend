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

			log.Printf("Registered User: %+v", registeredUser)

			c.JSON(http.StatusOK, gin.H{
				"message": "Registration successful. Redirecting to homepage.",
				"user":    registeredUser,
			})
		})

		auth.POST("/login", func(c *gin.Context) {
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

			log.Printf("Generated Token: %s", token)
			log.Printf("Logged-in User: %+v", user)

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
    // Get the user ID from the context
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
        return
    }

    // Fetch the user from the database using the user ID
    user, err := services.GetUserByID(userID.(uint))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching user from database"})
        return
    }

    // Log the full user data
    log.Printf("User data fetched from database: %+v", user)

    // Return the sanitized user data
    sanitizedUser := gin.H{
        "id":             user.ID,
        "username":       user.Username,
        "email":          user.Email,
        "profile_picture": user.ProfilePicture,
        "contact":        user.Contact,
    }

    c.JSON(http.StatusOK, gin.H{"user": sanitizedUser})
})

		// PUT /update-profile: Update the user's profile
		protected.PUT("/update-profile", func(c *gin.Context) {
			// Fetch user ID from the context
			userID, exists := c.Get("user_id")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
				return
			}

			// Fetch user details from the database
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

			// Log the update request data
			log.Printf("Updating user profile with: %+v", updateRequest)

			// Call the service to update user profile
			updatedUser, err := services.UpdateUserProfile(user.ID, updateRequest.Username, updateRequest.Email, updateRequest.Contact)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			log.Printf("Updated User Profile: %+v", updatedUser)

			c.JSON(http.StatusOK, gin.H{"user": updatedUser})
		})

		
		// POST /update-profile-picture: Update the user's profile picture
		protected.POST("/update-profile-picture", func(c *gin.Context) {
			// Fetch user ID from the context
			userID, exists := c.Get("user_id")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
				return
			}

			// Fetch user details from the database
			user, err := services.GetUserByID(userID.(uint))
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
				return
			}

			// Get the file from the request
			file, err := c.FormFile("profilePicture")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Profile picture file required"})
				return
			}

			// Define the upload directory and base URL
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

			// Save the uploaded file
			if err := c.SaveUploadedFile(file, filePath); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload profile picture"})
				return
			}

			// Update user profile with the new profile picture path
			updatedUser, err := services.UpdateUserProfilePicture(user.ID, profilePicturePath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"user": updatedUser})
		})

		// PUT /change-password: Change the user's password
		protected.PUT("/change-password", func(c *gin.Context) {
			// Fetch user ID from the context
			userID, exists := c.Get("user_id")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
				return
			}

			// Fetch user details from the database
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
		})
	}
}