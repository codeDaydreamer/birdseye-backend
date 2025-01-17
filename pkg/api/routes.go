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

    // Bind the incoming JSON body to the user model
    if err := c.ShouldBindJSON(&user); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
        log.Printf("Error binding request body: %v\n", err)
        return
    }

    // Register the user
    registeredUser, err := services.RegisterUser(&user)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        log.Printf("Error registering user: %v\n", err)
        return
    }

    // Send the response with a message and user data
    c.JSON(http.StatusOK, gin.H{
        "message": "Registration successful. Redirecting to homepage.",
        "user":    registeredUser,
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

		// Me route
		auth.GET("/me", func(c *gin.Context) {
			token := c.GetHeader("Authorization")
			if token == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
				return
			}

			user, err := services.GetUserFromToken(token)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"user": user,
			})
		})

		// Update profile route
		auth.PUT("/update-profile", func(c *gin.Context) {
			token := c.GetHeader("Authorization")
			if token == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
				return
			}

			user, err := services.GetUserFromToken(token)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
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

			updatedUser, err := services.UpdateUserProfile(user.ID, updateRequest.Username, updateRequest.Email, updateRequest.Contact)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"user": updatedUser,
			})
		})

		// Update profile picture route
		auth.POST("/update-profile-picture", func(c *gin.Context) {
			token := c.GetHeader("Authorization")
			if token == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
				return
			}

			user, err := services.GetUserFromToken(token)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}

			file, _ := c.FormFile("profilePicture")
			if file == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Profile picture file required"})
				return
			}

			baseURL := os.Getenv("BASE_URL")
			if baseURL == "" {
				baseURL = "http://localhost:8080/birdseye_backend"
			}

			profilePicturePath := baseURL + "/uploads/" + file.Filename
			if err := c.SaveUploadedFile(file, "uploads/"+file.Filename); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload profile picture"})
				return
			}

			updatedUser, err := services.UpdateUserProfilePicture(user.ID, profilePicturePath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"user": updatedUser,
			})
		})

		// Change password route
		auth.PUT("/change-password", func(c *gin.Context) {
			token := c.GetHeader("Authorization")
			if token == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
				return
			}

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

			err = services.ChangePassword(user.ID, changePasswordRequest.CurrentPassword, changePasswordRequest.NewPassword)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
		})
	}
}
