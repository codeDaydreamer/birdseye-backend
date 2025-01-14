package api

import (
    "github.com/gin-gonic/gin"
    "birdseye-backend/pkg/services"
    "birdseye-backend/pkg/models"
    "net/http"
    "log"
    "os"
)

// SetupRoutes sets up all the API routes
func SetupRoutes(router *gin.Engine) {
    auth := router.Group("/auth")
    {
        // Register route
        auth.POST("/register", func(c *gin.Context) {
            var user models.User
            
            // Attempt to bind JSON to user struct
            if err := c.ShouldBindJSON(&user); err != nil {
                // Log the error to diagnose why the binding failed
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
                log.Printf("Error binding request body: %v\n", err) // Log the error for debugging
                return
            }

            // Call the RegisterUser function from services
            registeredUser, err := services.RegisterUser(&user)
            if err != nil {
                // Log the registration error
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                log.Printf("Error registering user: %v\n", err) // Log error for registration
                return
            }

            // Call LoginUser to get the token
            token, _, err := services.LoginUser(registeredUser.Email, user.Password)
            if err != nil {
                // Log the login error
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                log.Printf("Error logging in user: %v\n", err) // Log error for login
                return
            }

            // Return the generated token and user details
            c.JSON(http.StatusOK, gin.H{
                "token": token,
                "user":  registeredUser, // Send user details along with token
                "message": "Registration successful. You can now log in.",
            })
        })

        // Login route
        auth.POST("/login", func(c *gin.Context) {
            var loginDetails struct {
                Email    string `json:"email"`
                Password string `json:"password"`
            }

            if err := c.ShouldBindJSON(&loginDetails); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
                return
            }

            // Call the LoginUser function from services
            token, user, err := services.LoginUser(loginDetails.Email, loginDetails.Password)
            if err != nil {
                c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
                return
            }

            // Return the generated token and user details
            c.JSON(http.StatusOK, gin.H{
                "token": token,
                "user":  user, // Send user details along with token
            })
        })

        // /me route to get authenticated user info
        auth.GET("/me", func(c *gin.Context) {
            // Retrieve the JWT token from the Authorization header
            token := c.GetHeader("Authorization")
            if token == "" {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
                return
            }

            // Call the service to decode the token and get user data
            user, err := services.GetUserFromToken(token)
            if err != nil {
                c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
                return
            }

            // Return the user data
            c.JSON(http.StatusOK, gin.H{
                "user": user,
            })
        })

        // Update Profile route
        auth.PUT("/updateProfile", func(c *gin.Context) {
            // Retrieve the JWT token from Authorization header
            token := c.GetHeader("Authorization")
            if token == "" {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
                return
            }

            // Get the authenticated user
            user, err := services.GetUserFromToken(token)
            if err != nil {
                c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
                return
            }

            // Parse the profile update request body
            var updateRequest struct {
                Username string `json:"username"`
                Email    string `json:"email"`
                Contact  string `json:"contact"`
            }

            if err := c.ShouldBindJSON(&updateRequest); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
                return
            }

            // Call the service to update the user's profile
            updatedUser, err := services.UpdateUserProfile(user.ID, updateRequest.Username, updateRequest.Email, updateRequest.Contact)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }

            // Return the updated user data
            c.JSON(http.StatusOK, gin.H{
                "user": updatedUser,
            })
        })

        // Update Profile Picture route
        auth.POST("/updateProfilePicture", func(c *gin.Context) {
            // Retrieve the JWT token from Authorization header
            token := c.GetHeader("Authorization")
            if token == "" {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
                return
            }

            // Get the authenticated user
            user, err := services.GetUserFromToken(token)
            if err != nil {
                c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
                return
            }

            // Parse the uploaded file (profile picture)
            file, _ := c.FormFile("profilePicture")
            if file == nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Profile picture file required"})
                return
            }

            // Get the base URL from environment variable
            baseURL := os.Getenv("BASE_URL")
            if baseURL == "" {
                baseURL = "http://localhost:8080/birdseye_backend" // Default if not set in .env
            }

            // Generate file path with base URL
            profilePicturePath := baseURL + "/uploads/" + file.Filename
            if err := c.SaveUploadedFile(file, "uploads/"+file.Filename); err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload profile picture"})
                return
            }

            // Call the service to update the user's profile picture
            updatedUser, err := services.UpdateUserProfilePicture(user.ID, profilePicturePath)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }

            // Return the updated user data with the new profile picture path
            c.JSON(http.StatusOK, gin.H{
                "user": updatedUser,
            })
        })

        // Change password route
        auth.PUT("/change-password", func(c *gin.Context) {
            // Get user ID from the JWT token
            token := c.GetHeader("Authorization")
            if token == "" {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
                return
            }

            // Get the authenticated user
            user, err := services.GetUserFromToken(token)
            if err != nil {
                c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
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

            // Call the ChangePassword function from services
            err = services.ChangePassword(user.ID, changePasswordRequest.CurrentPassword, changePasswordRequest.NewPassword)
            if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
            }

            // Return success message
            c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
        })

        // OTP verification route (optional, if implemented in services)
        auth.POST("/verify-otp", func(c *gin.Context) {
            // Implement OTP verification here, if applicable
            c.JSON(http.StatusOK, gin.H{"message": "OTP verified"})
        })
    }
}
