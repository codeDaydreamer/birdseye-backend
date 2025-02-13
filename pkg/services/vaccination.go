package services

import (
	"birdseye-backend/pkg/broadcast"
	"birdseye-backend/pkg/models"
	"errors"

	"gorm.io/gorm"
)

type VaccinationService struct {
	DB *gorm.DB
}

func NewVaccinationService(db *gorm.DB) *VaccinationService {
	return &VaccinationService{
		DB: db,
	}
}

// GetVaccinationsByFlock retrieves all vaccination records for a flock (using 'id' as the parameter)
func (s *VaccinationService) GetVaccinationsByFlock(id uint) ([]models.Vaccination, error) {
	var vaccinations []models.Vaccination
	err := s.DB.Where("flock_id = ?", id).Find(&vaccinations).Error
	return vaccinations, err
}

// GetVaccinationByID retrieves a single vaccination record by ID
func (s *VaccinationService) GetVaccinationByID(id uint) (*models.Vaccination, error) {
	var vaccination models.Vaccination
	err := s.DB.First(&vaccination, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("vaccination record not found")
		}
		return nil, err
	}
	return &vaccination, nil
}

// AddVaccination adds a new vaccination record and sends a WebSocket update
func (s *VaccinationService) AddVaccination(vaccination *models.Vaccination) error {
	if err := s.DB.Create(vaccination).Error; err != nil {
		return err
	}
	broadcast.SendFlockUpdate("vaccination_added", *vaccination)
	return nil
}
func (s *VaccinationService) UpdateVaccination(vaccinationID, id uint, updatedData map[string]interface{}) error {
	// Fetch the existing record
	var vaccination models.Vaccination
	if err := s.DB.Where("id = ? AND flock_id = ?", vaccinationID, id).First(&vaccination).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("vaccination record not found")
		}
		return err
	}

	// Define allowed fields to update
	allowedFields := map[string]bool{
		"vaccine_name": true,
		"status":       true,
	}

	// Filter only allowed fields
	validUpdates := make(map[string]interface{})
	for key, value := range updatedData {
		if allowedFields[key] {
			validUpdates[key] = value
		}
	}

	// ✅ Explicitly exclude "date" if it's not provided
	if _, dateExists := updatedData["date"]; !dateExists {
		s.DB = s.DB.Omit("date") // Prevent GORM from including `date`
	}

	// If no valid updates are provided, return an error
	if len(validUpdates) == 0 {
		return errors.New("no valid fields provided for update")
	}

	// Perform the update using GORM's Updates() function
	if err := s.DB.Model(&vaccination).Updates(validUpdates).Error; err != nil {
		return err
	}

	// Send WebSocket update
	broadcast.SendFlockUpdate("vaccination_updated", vaccination)
	return nil
}

// DeleteVaccination removes a vaccination record by ID
func (s *VaccinationService) DeleteVaccination(vaccinationID uint) error {
	var vaccination models.Vaccination
	if err := s.DB.First(&vaccination, vaccinationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("vaccination record not found")
		}
		return err
	}

	if err := s.DB.Delete(&vaccination).Error; err != nil {
		return err
	}

	broadcast.SendFlockUpdate("vaccination_deleted", vaccinationID)
	return nil
}
