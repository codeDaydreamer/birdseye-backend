package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/middlewares"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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

// AddVaccination adds a vaccination record for a flock (using 'id' as parameter)
func (h *VaccinationHandler) AddVaccination(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid flock ID"})
		return
	}

	var vaccination models.Vaccination
	if err := c.ShouldBindJSON(&vaccination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vaccination.FlockID = uint(id)

	if err := db.DB.Create(&vaccination).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add vaccination"})
		return
	}

	c.JSON(http.StatusCreated, vaccination)
}

// UpdateVaccination updates a vaccination record
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

	var updatedData models.Vaccination
	if err := c.ShouldBindJSON(&updatedData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply updates
	vaccination.VaccineName = updatedData.VaccineName
	vaccination.Date = updatedData.Date
	vaccination.Status = updatedData.Status

	if err := db.DB.Save(&vaccination).Error; err != nil {
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
