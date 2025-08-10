package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/middlewares"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type VaccinationHandler struct{}

func SetupVaccinationRoutes(r *gin.Engine) {
	handler := &VaccinationHandler{}

	// Routes for vaccinations tied to a flock
	vaccinationRoutes := r.Group("/flocks/:id/vaccinations").Use(middlewares.AuthMiddleware())
	{
		vaccinationRoutes.GET("/", handler.GetVaccinations)
		vaccinationRoutes.POST("/", handler.AddVaccination)
		vaccinationRoutes.PUT("/:vaccination_id", handler.UpdateVaccination)
		vaccinationRoutes.DELETE("/:vaccination_id", handler.DeleteVaccination)
	}

	// ✅ New route for fetching vaccinations by logged-in user
	userVaccinationRoutes := r.Group("/vaccinations").Use(middlewares.AuthMiddleware())
	{
		userVaccinationRoutes.GET("/", handler.GetVaccinationsByUserID)
	}
}

// ✅ NEW: Get vaccinations by user ID
func (h *VaccinationHandler) GetVaccinationsByUserID(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Fetch user data to ensure the user exists
	user, err := models.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user data"})
		return
	}

	// Join vaccinations with flocks to ensure only this user's records are fetched
	var vaccinations []models.Vaccination
	if err := db.DB.
		Joins("JOIN flocks ON flocks.id = vaccinations.flock_id").
		Where("flocks.user_id = ?", user.ID).
		Find(&vaccinations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve vaccinations"})
		return
	}

	c.JSON(http.StatusOK, vaccinations)
}

// Existing method: GetVaccinations by flock ID
func (h *VaccinationHandler) GetVaccinations(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid flock ID"})
		return
	}

	var vaccinations []models.Vaccination
	if err := db.DB.Where("flock_id = ?", id).Find(&vaccinations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve vaccinations"})
		return
	}

	c.JSON(http.StatusOK, vaccinations)
}

func (h *VaccinationHandler) AddVaccination(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid flock ID"})
		return
	}

	var rawData map[string]interface{}
	if err := c.ShouldBindJSON(&rawData); err != nil {
		fmt.Println("JSON Bind Error:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format", "details": err.Error()})
		return
	}

	var vaccination models.Vaccination

	if vaccineName, ok := rawData["vaccine_name"].(string); ok {
		vaccination.VaccineName = vaccineName
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or invalid 'vaccine_name'"})
		return
	}

	if status, ok := rawData["status"].(string); ok {
		vaccination.Status = status
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or invalid 'status'"})
		return
	}

	if userID, ok := rawData["user_id"].(float64); ok {
		vaccination.UserID = uint(userID)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or invalid 'user_id'"})
		return
	}

	dateStr, ok := rawData["date"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or invalid 'date'"})
		return
	}
	parsedDate, err := time.Parse("2006-01-02 15:04:05", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Expected 'YYYY-MM-DD HH:MM:SS'", "details": err.Error()})
		return
	}
	vaccination.Date = parsedDate
	vaccination.FlockID = uint(id)

	result := db.DB.Debug().Create(&vaccination)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add vaccination", "details": result.Error.Error()})
		return
	}

	c.JSON(http.StatusCreated, vaccination)
}

func (h *VaccinationHandler) UpdateVaccination(c *gin.Context) {
	id, err1 := strconv.Atoi(c.Param("id"))
	vaccinationID, err2 := strconv.Atoi(c.Param("vaccination_id"))

	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var vaccination models.Vaccination
	if err := db.DB.Where("id = ? AND flock_id = ?", vaccinationID, id).First(&vaccination).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vaccination not found"})
		return
	}

	var updatedData map[string]interface{}
	if err := c.ShouldBindJSON(&updatedData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := db.DB.Model(&vaccination)
	if _, exists := updatedData["date"]; !exists {
		query = query.Omit("date")
	}

	if err := query.Updates(updatedData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update vaccination"})
		return
	}

	c.JSON(http.StatusOK, vaccination)
}

func (h *VaccinationHandler) DeleteVaccination(c *gin.Context) {
	id, err1 := strconv.Atoi(c.Param("id"))
	vaccinationID, err2 := strconv.Atoi(c.Param("vaccination_id"))

	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var vaccination models.Vaccination
	if err := db.DB.Where("id = ? AND flock_id = ?", vaccinationID, id).First(&vaccination).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vaccination not found"})
		return
	}

	if err := db.DB.Delete(&vaccination).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete vaccination"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vaccination deleted successfully"})
}
