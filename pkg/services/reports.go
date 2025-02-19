package services

import (
	"birdseye-backend/pkg/models"
	"fmt"
	"time"
	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"strings"
	"gorm.io/gorm"
	"path/filepath"
	"os"
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
func (r *ReportsService) GenerateReport(reportType string, query interface{}, formatHTMLFunc func(items interface{}) string, userID uint) (models.Report, error) {
	var items []interface{}

	// Specify the model (e.g., models.Sale) to map the query to
	err := r.DB.Model(&models.Sale{}).Where(query, userID).Find(&items).Error
	if err != nil {
		return models.Report{}, err
	}

	// Generate report content in HTML format using the passed format function
	reportContent := formatHTMLFunc(items)

	// Convert the HTML content to io.Reader
	reportReader := strings.NewReader(reportContent)

	// Convert the HTML content to PDF using wkhtmltopdf
	pdfGen, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return models.Report{}, fmt.Errorf("failed to create PDF generator: %w", err)
	}
	pdfGen.AddPage(wkhtmltopdf.NewPageReader(reportReader))

	// Create the PDF
	err = pdfGen.Create()
	if err != nil {
		return models.Report{}, fmt.Errorf("failed to create PDF: %w", err)
	}

	// Ensure reports directory exists
	reportsDir := "reports"
	if err := os.MkdirAll(reportsDir, os.ModePerm); err != nil {
		return models.Report{}, fmt.Errorf("failed to create reports directory: %w", err)
	}

	// Define the file name
	fileName := fmt.Sprintf("%s_report_%d_%d.pdf", reportType, userID, time.Now().Unix())

	// Define the relative file path
	filePath := filepath.Join(reportsDir, fileName)

	// Save the PDF to file
	if err = pdfGen.WriteFile(filePath); err != nil {
		return models.Report{}, fmt.Errorf("failed to save PDF: %w", err)
	}

	// Get the base URL from environment variable (or default if not set)
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080" // Only the base domain, no path
	}

	// Generate the absolute file URL for frontend download
	absoluteFileURL := fmt.Sprintf("%s/birdseye_backend/reports/%s", baseURL, fileName)

	// Save the report information in the database
	report := &models.Report{
		ReportType:  reportType,
		Content:     absoluteFileURL, // Save the absolute URL to access the file
		UserID:      userID,
		GeneratedAt: time.Now(),
		Name:        fileName, // Store the name of the report (file name)
	}

	err = r.DB.Create(&report).Error
	if err != nil {
		return models.Report{}, err
	}

	return *report, nil
}


// GenerateInventoryReport generates an inventory report for a user
func (r *ReportsService) GenerateInventoryReport(userID uint) (models.Report, error) {
	// Call GenerateReport with a specific query and format function for inventory
	query := "user_id = ?"
	formatHTMLFunc := func(items interface{}) string {
		var inventory []models.InventoryItem = items.([]models.InventoryItem)
		reportContent := "<html><body><h1>Inventory Report</h1><table><tr><th>ID</th><th>Item</th><th>Quantity</th><th>Price</th></tr>"
		for _, item := range inventory {
			reportContent += fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%d</td><td>%f</td></tr>", item.ID, item.ItemName, item.Quantity, item.CostPerUnit)
		}
		reportContent += "</table></body></html>"
		return reportContent
	}

	return r.GenerateReport("Inventory", query, formatHTMLFunc, userID)
}
