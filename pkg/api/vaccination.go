package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/middlewares"
	"net/http"
	"strconv"
	"time"
	"github.com/gin-gonic/gin"
	"fmt"
)

// VaccinationHandler handles vaccination-related requests
type VaccinationHandler struct{}

// SetupVaccinationRoutes sets up the vaccination API routes
func SetupVaccinationRoutes(r *gin.Engine) {
	handler := &VaccinationHandler{}

	// Use :id instead of :flock_id
	vaccinationRoutes := r.Group("/flocks/:id/vaccinations").Use(middlewares.AuthMiddleware())
	{
		vaccinationRoutes.GET("/", handler.GetVaccinations)
		vaccinationRoutes.POST("/", handler.AddVaccination)
		vaccinationRoutes.PUT("/:vaccination_id", handler.UpdateVaccination)
		vaccinationRoutes.DELETE("/:vaccination_id", handler.DeleteVaccination)
	}
}

// GetVaccinations retrieves all vaccinations for a flock (using 'id' as parameter)
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

	// Log the raw JSON request body for debugging
	var rawData map[string]interface{}
	if err := c.ShouldBindJSON(&rawData); err != nil {
		fmt.Println("JSON Bind Error:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format", "details": err.Error()})
		return
	}

	// Print received keys
	fmt.Println("Received Data Keys:")
	for key := range rawData {
		fmt.Println("-", key)
	}

	// Extracting necessary fields manually to avoid conflicts
	var vaccination models.Vaccination

	// Required Fields
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

	// User ID (Optional, but must be uint)
	if userID, ok := rawData["user_id"].(float64); ok { // JSON numbers are float64 by default
		vaccination.UserID = uint(userID)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or invalid 'user_id'"})
		return
	}

	// Date Parsing
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

	// Assign Flock ID from URL, not JSON request
	vaccination.FlockID = uint(id)

	// Log final struct before saving
	fmt.Printf("Final Vaccination Object: %+v\n", vaccination)

	// Insert into DB with debugging
	result := db.DB.Debug().Create(&vaccination)
	fmt.Println("Rows Affected:", result.RowsAffected)

	if result.Error != nil {
		fmt.Println("DB Error:", result.Error)
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

    // Parse JSON into a map to check for missing fields
    var updatedData map[string]interface{}
    if err := c.ShouldBindJSON(&updatedData); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Ensure "date" is updated only if explicitly provided
    query := db.DB.Model(&vaccination)
    if _, exists := updatedData["date"]; !exists {
        query = query.Omit("date")
    }

    // Perform update
    if err := query.Updates(updatedData).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update vaccination"})
        return
    }

    c.JSON(http.StatusOK, vaccination)
}

// DeleteVaccination deletes a vaccination record
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
