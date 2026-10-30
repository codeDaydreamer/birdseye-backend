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

	// Routes for vaccinations tied to one of the farm's flocks
	vaccinationRoutes := r.Group("/flocks/:id/vaccinations").Use(middlewares.FarmAuth(), requireFlockInFarm())
	{
		vaccinationRoutes.GET("/", handler.GetVaccinations)
		vaccinationRoutes.POST("/", handler.AddVaccination)
		vaccinationRoutes.PUT("/:vaccination_id", handler.UpdateVaccination)
		vaccinationRoutes.DELETE("/:vaccination_id", handler.DeleteVaccination)
	}

	// All vaccinations across the farm's flocks
	farmVaccinationRoutes := r.Group("/vaccinations").Use(middlewares.FarmAuth())
	{
		farmVaccinationRoutes.GET("/", handler.GetFarmVaccinations)
	}
}

// requireFlockInFarm rejects requests whose :id flock doesn't belong to the caller's farm
func requireFlockInFarm() gin.HandlerFunc {
	return func(c *gin.Context) {
		farmID, _ := farmContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid flock ID"})
			c.Abort()
			return
		}
		if !flockInFarm(uint(id), farmID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Flock not found"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// GetFarmVaccinations retrieves vaccinations for all of the farm's flocks
func (h *VaccinationHandler) GetFarmVaccinations(c *gin.Context) {
	farmID, _ := farmContext(c)

	var vaccinations []models.Vaccination
	if err := db.DB.Where("farm_id = ?", farmID).Find(&vaccinations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve vaccinations"})
		return
	}

	c.JSON(http.StatusOK, vaccinations)
}

// GetVaccinations retrieves vaccinations for one flock
func (h *VaccinationHandler) GetVaccinations(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var vaccinations []models.Vaccination
	if err := db.DB.Where("flock_id = ?", id).Find(&vaccinations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve vaccinations"})
		return
	}

	c.JSON(http.StatusOK, vaccinations)
}

func (h *VaccinationHandler) AddVaccination(c *gin.Context) {
	farmID, userID := farmContext(c)
	id, _ := strconv.Atoi(c.Param("id"))

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

	// New: handle mode_of_administration (optional)
	if mode, ok := rawData["mode_of_administration"].(string); ok {
		vaccination.ModeOfAdministration = mode
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
	vaccination.FarmID = farmID
	vaccination.CreatedByUserID = userID

	if err := db.DB.Create(&vaccination).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add vaccination", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, vaccination)
}

// vaccinationEditableFields are the only columns a client may change on update
var vaccinationEditableFields = map[string]bool{
	"vaccine_name":           true,
	"status":                 true,
	"date":                   true,
	"mode_of_administration": true,
	"period":                 true,
}

func (h *VaccinationHandler) UpdateVaccination(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	vaccinationID, err := strconv.Atoi(c.Param("vaccination_id"))
	if err != nil {
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

	validUpdates := make(map[string]interface{})
	for key, value := range updatedData {
		if vaccinationEditableFields[key] {
			validUpdates[key] = value
		}
	}

	query := db.DB.Model(&vaccination)
	if _, exists := validUpdates["date"]; !exists {
		query = query.Omit("date")
	}

	if err := query.Updates(validUpdates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update vaccination"})
		return
	}

	c.JSON(http.StatusOK, vaccination)
}

func (h *VaccinationHandler) DeleteVaccination(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	vaccinationID, err := strconv.Atoi(c.Param("vaccination_id"))
	if err != nil {
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
