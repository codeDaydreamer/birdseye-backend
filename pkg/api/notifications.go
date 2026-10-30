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
	notificationRoutes := r.Group("/notifications").Use(middlewares.FarmAuth())
	{
		notificationRoutes.GET("/", handler.GetNotifications)
		notificationRoutes.POST("/", handler.CreateNotification)
		notificationRoutes.PUT("/:id/read", handler.MarkNotificationAsRead)
	}
}

// GetNotifications retrieves the user's notifications for the current farm
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	farmID, userID := farmContext(c)

	var notifications []models.Notification
	if err := db.DB.Where("user_id = ? AND farm_id = ?", userID, farmID).Order("created_at DESC").Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve notifications"})
		return
	}

	c.JSON(http.StatusOK, notifications)
}

// CreateNotification creates a notification for the user and broadcasts it via WebSocket
func (h *NotificationHandler) CreateNotification(c *gin.Context) {
	farmID, userID := farmContext(c)

	var notification models.Notification
	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notification.ID = 0
	notification.UserID = userID
	notification.FarmID = farmID

	if err := db.DB.Create(&notification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification"})
		return
	}

	// Broadcast notification via WebSocket using the required parameters
	broadcast.SendNotification(notification.UserID, notification.Title, notification.Body, notification.URL)

	c.JSON(http.StatusCreated, notification)
}


// MarkNotificationAsRead marks one of the user's notifications as read
func (h *NotificationHandler) MarkNotificationAsRead(c *gin.Context) {
	_, userID := farmContext(c)
	id := c.Param("id")
	var notification models.Notification

	if err := db.DB.First(&notification, "id = ? AND user_id = ?", id, userID).Error; err != nil {
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
