package api

import (
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/broadcast"
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	
	"net/http"

	"github.com/gin-gonic/gin"
)

// NotificationHandler handles notification-related requests
type NotificationHandler struct{}

// SetupNotificationRoutes sets up the notification API routes with authentication middleware
func SetupNotificationRoutes(r *gin.Engine) {
	handler := &NotificationHandler{}
	notificationRoutes := r.Group("/notifications").Use(middlewares.AuthMiddleware())
	{
		notificationRoutes.GET("/", handler.GetNotifications)
		notificationRoutes.POST("/", handler.CreateNotification)
		notificationRoutes.PUT("/:id/read", handler.MarkNotificationAsRead)
	}
}

// GetNotifications retrieves notifications for the authenticated user
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var notifications []models.Notification
	if err := db.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve notifications"})
		return
	}

	c.JSON(http.StatusOK, notifications)
}

// CreateNotification creates a new notification and broadcasts it via WebSocket
func (h *NotificationHandler) CreateNotification(c *gin.Context) {
	var notification models.Notification
	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := db.DB.Create(&notification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification"})
		return
	}

	// Broadcast notification via WebSocket using the required parameters
	broadcast.SendNotification(notification.UserID, notification.Title, notification.Body, notification.URL)

	c.JSON(http.StatusCreated, notification)
}


// MarkNotificationAsRead marks a notification as read
func (h *NotificationHandler) MarkNotificationAsRead(c *gin.Context) {
	id := c.Param("id")
	var notification models.Notification

	if err := db.DB.First(&notification, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	notification.Read = true
	if err := db.DB.Save(&notification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notification"})
		return
	}

	c.JSON(http.StatusOK, notification)
}
