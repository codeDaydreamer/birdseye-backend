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

// AddVaccination adds a new vaccination record and broadcasts an update
func (s *VaccinationService) AddVaccination(vaccination *models.Vaccination) error {
	if err := s.DB.Create(vaccination).Error; err != nil {
		return err
	}
	broadcast.SendFlockUpdate("vaccination_added", *vaccination)
	return nil
}

// UpdateVaccination updates an existing vaccination record independently of the date
func (s *VaccinationService) UpdateVaccination(flockID, vaccinationID uint, updatedData map[string]interface{}) error {
    var vaccination models.Vaccination
    if err := s.DB.Where("id = ? AND flock_id = ?", vaccinationID, flockID).First(&vaccination).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return errors.New("vaccination record not found")
        }
        return err
    }

    // Only allow specific fields to be updated
    allowedFields := map[string]bool{
        "vaccine_name": true,
        "status":       true,
    }

    validUpdates := make(map[string]interface{})
    for key, value := range updatedData {
        if allowedFields[key] {
            validUpdates[key] = value
        }
    }

    // If no valid updates are provided, return an error
    if len(validUpdates) == 0 {
        return errors.New("no valid fields provided for update")
    }

    // Use Select() to explicitly update only the allowed fields
    if err := s.DB.Model(&vaccination).Select("vaccine_name", "status").Updates(validUpdates).Error; err != nil {
        return err
    }

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
