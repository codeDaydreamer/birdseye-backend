package services

import (
	"birdseye-backend/pkg/broadcast"
	"birdseye-backend/pkg/models"
	"errors"
	"time"
	"gorm.io/gorm"
)

type VaccinationService struct {
	DB *gorm.DB
}

func NewVaccinationService(db *gorm.DB) *VaccinationService {
	return &VaccinationService{DB: db}
}

// GetVaccinationsByFlock retrieves all vaccination records for a flock
func (s *VaccinationService) GetVaccinationsByFlock(flockID uint) ([]models.Vaccination, error) {
	var vaccinations []models.Vaccination
	err := s.DB.Where("flock_id = ?", flockID).Find(&vaccinations).Error
	if err != nil {
		return nil, err
	}
	return vaccinations, nil
}

// GetVaccinationByID retrieves a single vaccination record by ID
func (s *VaccinationService) GetVaccinationByID(vaccinationID uint) (*models.Vaccination, error) {
	var vaccination models.Vaccination
	err := s.DB.First(&vaccination, vaccinationID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("vaccination record not found")
		}
		return nil, err
	}
	return &vaccination, nil
}

func (s *VaccinationService) AddVaccination(vaccination *models.Vaccination) error {
	// Ensure the date is correctly formatted
	vaccination.Date = vaccination.Date.Local()

	if err := s.DB.Create(vaccination).Error; err != nil {
		return err
	}

	broadcast.SendFlockUpdate(vaccination.FlockID, "vaccination_added", *vaccination)
	return nil
}

func (s *VaccinationService) UpdateVaccination(flockID, vaccinationID uint, updatedData map[string]interface{}) error {
	var vaccination models.Vaccination
	if err := s.DB.Where("id = ? AND flock_id = ?", vaccinationID, flockID).First(&vaccination).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("vaccination record not found")
		}
		return err
	}

	allowedFields := map[string]bool{
		"vaccine_name": true,
		"status":       true,
		"date":         true,
	}

	validUpdates := make(map[string]interface{})
	for key, value := range updatedData {
		if allowedFields[key] {
			if key == "date" {
				if strDate, ok := value.(string); ok {
					parsedTime, err := time.Parse(time.RFC3339, strDate)
					if err != nil {
						return errors.New("invalid date format, expected RFC3339")
					}
					validUpdates["date"] = parsedTime.Format("2006-01-02 15:04:05")
				}
			} else {
				validUpdates[key] = value
			}
		}
	}

	query := s.DB.Model(&vaccination)
	if _, exists := updatedData["date"]; !exists {
		query = query.Omit("date")
	}

	if err := query.Updates(validUpdates).Error; err != nil {
		return err
	}

	broadcast.SendFlockUpdate(vaccination.FlockID, "vaccination_updated", vaccination)
	return nil
}

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

	broadcast.SendFlockUpdate(vaccination.FlockID, "vaccination_deleted", vaccinationID)
	return nil
}

