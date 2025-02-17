package services

import (
	"birdseye-backend/pkg/models"
	"fmt"
	"time"
	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"strings"
	"gorm.io/gorm"
)

// ReportsService handles report generation
type ReportsService struct {
	DB *gorm.DB
}

// NewReportsService initializes a new service instance
func NewReportsService(db *gorm.DB) *ReportsService {
	return &ReportsService{DB: db}
}
// GenerateReport generates a report of any type based on a query and format function
func (r *ReportsService) GenerateReport(reportType string, query interface{}, formatHTMLFunc func(items interface{}) string, userID uint) (models.Report, error) {
	var items []interface{} // Create a slice to store query results

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

	// Save the PDF to a file
	filePath := fmt.Sprintf("reports/%s_report_%d_%d.pdf", reportType, userID, time.Now().Unix())
	err = pdfGen.WriteFile(filePath)
	if err != nil {
		return models.Report{}, fmt.Errorf("failed to save PDF: %w", err)
	}

	// Save the report information in the database
	report := &models.Report{
		ReportType:  reportType,
		Content:     filePath,
		UserID:      userID,
		GeneratedAt: time.Now(),
	}

	err = r.DB.Create(&report).Error
	if err != nil {
		return models.Report{}, err
	}

	return *report, nil
}


// GenerateSalesReport generates a sales report for a given date range and user
func (r *ReportsService) GenerateSalesReport(startDate, endDate time.Time, userID uint) (models.Report, error) {
	// Call GenerateReport with a specific query and format function for sales
	query := "date BETWEEN ? AND ? AND user_id = ?"
	formatHTMLFunc := func(items interface{}) string {
		var sales []models.Sale = items.([]models.Sale)
		reportContent := "<html><body><h1>Sales Report</h1><table><tr><th>ID</th><th>Product</th><th>Amount</th></tr>"
		for _, sale := range sales {
			reportContent += fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%f</td></tr>", sale.ID, sale.Product, sale.Amount)
		}
		reportContent += "</table></body></html>"
		return reportContent
	}

	return r.GenerateReport("Sales", query, formatHTMLFunc, userID)
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
