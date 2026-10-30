package api

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services"
)

func handleAdminCreateUser(c *gin.Context) {
	var newUser models.User
	if err := c.ShouldBindJSON(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	createdUser, err := services.AdminCreateUser(&newUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully", "user": createdUser})
}
func handleAdminResetUserPassword(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		NewPassword string `json:"new_password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.NewPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New password is required"})
		return
	}

	err := services.AdminResetUserPassword(id, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User password reset successfully"})
}
