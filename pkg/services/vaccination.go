package services

import (
	"birdseye-backend/pkg/broadcast"
	"birdseye-backend/pkg/models"
	"errors"
	"time"
	"gorm.io/gorm"
	"fmt"
	"log"


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

// AddVaccination adds a new vaccination
func (s *VaccinationService) AddVaccination(vaccination *models.Vaccination) error {
	// Ensure the date is correctly formatted
	vaccination.Date = vaccination.Date.Local()

	if err := s.DB.Create(vaccination).Error; err != nil {
		return err
	}

	broadcast.SendFlockUpdate(vaccination.FlockID, "vaccination_added", *vaccination)
	return nil
}

// UpdateVaccination updates a vaccination record
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
		"mode_of_administration": true,
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

// DeleteVaccination deletes a vaccination record
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
// SendVaccinationReminder sends a reminder to the user for upcoming vaccinations
func (s *VaccinationService) SendVaccinationReminder(vaccination *models.Vaccination, userID uint) error {
	// Calculate the reminder date (e.g., 3 days before the vaccination date)
	reminderDate := vaccination.Date.AddDate(0, 0, -3) // Set to 3 days before the vaccination date

	// Check if the current date is the reminder date
	if time.Now().After(reminderDate) && time.Now().Before(vaccination.Date) {
		// Log the vaccination reminder
		log.Println("Sending vaccination reminder...")

		// Fetch the flock name using the FlockID
		var flock models.Flock
		if err := s.DB.First(&flock, vaccination.FlockID).Error; err != nil {
			log.Printf("Error retrieving flock name for FlockID %d: %v", vaccination.FlockID, err)
			return err
		}

		// Send WebSocket update with vaccination name and flock name
		log.Println("📡 Sending WebSocket update for vaccination reminder...")
		reminderData := map[string]interface{}{
			"vaccination_name": vaccination.VaccineName, // Include the vaccination name
			"vaccination_date": vaccination.Date.Format("2006-01-02"), // Include the vaccination date
			"flock_name":       flock.Name, // Include the flock name
		}
		broadcast.SendVaccinationUpdate(userID, "vaccination_reminder", reminderData)
		log.Println("✅ WebSocket update sent.")

		// Send push notification with vaccination name and flock name
		title := fmt.Sprintf("Reminder: %s vaccination for flock %s", vaccination.VaccineName, flock.Name)
		message := fmt.Sprintf("Don't forget! The %s vaccination for flock %s is approaching on %s.", vaccination.VaccineName, flock.Name, vaccination.Date.Format("2006-01-02"))
		url := "/vaccination/" + fmt.Sprintf("%d", vaccination.ID)

		log.Printf("🔔 Sending push notification: Title='%s', Message='%s', URL='%s'\n", title, message, url)
		broadcast.SendNotification(userID, title, message, url)
		log.Println("✅ Push notification sent.")
	}

	return nil
}
