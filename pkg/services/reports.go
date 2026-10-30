package services

import (
	"birdseye-backend/pkg/models"
	"gorm.io/gorm"
	"errors"
)

// ReportsService handles report generation
type ReportsService struct {
	DB *gorm.DB
}

// NewReportsService initializes a new service instance
func NewReportsService(db *gorm.DB) *ReportsService {
	return &ReportsService{DB: db}
}

// GetUserReports fetches all reports for the specified user
func (s *ReportsService) GetUserReports(userID uint) ([]models.Report, error) {
	var reports []models.Report
	err := s.DB.Where("user_id = ?", userID).Find(&reports).Error
	if err != nil {
		return nil, err
	}
	return reports, nil
}


func (s *ReportsService) DeleteReport(userID uint, reportID string) error {
    // Find and delete the report for the given user
    result := s.DB.Where("id = ? AND user_id = ?", reportID, userID).Delete(&models.Report{})
    if result.Error != nil {
        return result.Error
    }

    if result.RowsAffected == 0 {
        return errors.New("report not found or unauthorized")
    }

    return nil
}
