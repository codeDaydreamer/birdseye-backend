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


func generateReportHeader(title, subtitle string) string {
    return fmt.Sprintf(`
    <html>
    <head>
        <style>
            @page {
                size: A4;
                margin: 20mm;
            }
            body {
                font-family: "Times New Roman", serif;
                margin: 0;
                padding: 0;
                font-size: 19px;
                position: relative;
                min-height: 100vh;
                padding-top: 300px; /* Adjust this value to match the header height */
                padding-bottom: 100px; /* Adjust this value to match the footer height */
            }

            header {
                text-align: center;
                position: fixed;
                top: 0;
                left: 0;
                right: 0;
                padding: 15px 0;
                background-color:rgb(255, 240, 202);
                border-bottom: 2px solid black;
            }
            .logo {
                width: 100px;
                height: auto;
                display: block;
                margin: 0 auto;
            }
            .report-title {
                font-size: 26px;
                font-weight: bold;
                margin-top: 5px;
            }
            .report-subtitle {
                font-size: 20px;
                margin-top: 5px;
            }
            hr {
                border: none;
                height: 2px;
                background-color: black;
                margin: 10px 0;
            }
            footer {
                text-align: center;
                position: fixed;
                bottom: 0;
                left: 0;
                right: 0;
                padding: 10px 0;
                background-color:rgba(255, 240, 202, 0.86);
                border-top: 2px solid black;
                font-size: 20px;
            }
            table.report-table {
                width:950px;
                border-collapse: collapse;
                margin-top: 10px;
                font-size: 19px;
                margin-left:20px;
            }    
            .grand-total-row td{
                font-weight:bold;
                background-colour:rgba(255, 240, 202, 0.67);
                border-top: 3px solid black;
                font-size: 20px;
            } 
            th{
                text-align: left;
                padding: 10px;
                border: 2px solid black;
                font-weight: bold; 
                background-colour: rgba(255, 240, 202, 0.86);           
            }    
            thead tr th{
                text-align: left;
                padding: 10px;
                border: 1px solid black;
                font-weight: bold;
                background-colour: rgba(255, 240, 202, 0.86); 
            }  
            tbody tr.data-rows:nth-child(even){
                background-color:rgba(255, 240, 202, 0.86);
            }
            tbody td {
                padding: 10px;
                border:  1px solid black;
            }  
            td.amount {
                text-align: right;
                font-weight: bold;
            }  
            .subtotal-row td {
                font-weight: bold;
                background-color:rgba(255, 240, 202, 0.86)
                border-top: 2px solid black;
            }   
            .grand-total-row td {
                font-weight: bold;
                background-color:rgb(252, 230, 173);
                border-top: 3px solid black;
                font-size: 21px;


            }   
            .report-table thead {
                 display: table-header-group;
            }  
            .report-table tbody tr {
                page-break-inside: avoid;
            }  
            .report-table tbody {
                display: table-row-group;
            }  
            .page-break {
                page-break-before: always;
            }                      
            .page-number:after {
                content: counter(page);
            }

        </style>
    </head>
    <body>
        <header>
            <img src="http://localhost:8080/birdseye_backend/uploads/icon-512x512.svg" alt="Birdseye Logo" class="logo">
            <div class="report-title">Birdseye Poultry Management System</div>
            <div class="report-subtitle">%s</div>
            <div class="report-subtitle">%s</div>
            <hr>
        </header>

        <main>
    `, title, subtitle)
}

func generateReportFooter() string {
    return `
        </main>
        <footer>
            <hr>
            <div>Generated by Birdseye Poultry Management System</div>
            <div>&copy; 2025 BirdsEye. All rights reserved.</div>
        </footer>
    </body>
    </html>`
}


//generate the sales report
func (r *ReportsService) GenerateSalesReport(startDate, endDate time.Time, userID uint) (models.Report, error) {
	var sales []models.Sale

	// Fetch sales records
	err := r.DB.Where("date BETWEEN ? AND ? AND user_id = ?", startDate, endDate, userID).Find(&sales).Error
	if err != nil {
		return models.Report{}, err
	}

	// Calculate subtotals by category
	categorySubtotals := make(map[string]float64)
	var grandTotal float64

	for _, sale := range sales {
		categorySubtotals[sale.Category] += sale.Amount
		grandTotal += sale.Amount
	}

	// Helper function to format amounts to KES
	formatKES := func(amount float64) string {
		return fmt.Sprintf("KES %.2f", amount)
	}

	// Generate the report header and footer
	reportHeader := generateReportHeader("Sales Report", fmt.Sprintf("From: %s To: %s", startDate.Format("Jan 2, 2006"), endDate.Format("Jan 2, 2006")))
	reportFooter := generateReportFooter()

	// Build the report content with table rows for sales data
	reportContent := reportHeader + `
    <table class="report-table">
        <thead>
            <tr>
                <th>#</th>
                <th>Ref No</th>
                <th>Product Category</th>
                <th>Amount (KES)</th>
            </tr>
        </thead>
        <tbody>`

// Add sales rows
itemNumber := 1
for _, sale := range sales {
    reportContent += fmt.Sprintf(`
        <tr class="data-rows">
            <td>%d</td>
            <td>%s</td>
            <td>%s</td>
            <td class="amount">%s</td>
        </tr>`, itemNumber, sale.RefNo, sale.Category, formatKES(sale.Amount))
    itemNumber++
}

// Add subtotals by category
reportContent += `
        <tr class="subtotal-row">
            <td colspan="3"><strong>Category Subtotals</strong></td>
            <td class="amount"><strong>` + formatKES(grandTotal) + `</strong></td>
        </tr>`

// Add subtotal rows for each category
for category, subtotal := range categorySubtotals {
    reportContent += fmt.Sprintf(`
        <tr class="subtotal-row">
            <td colspan="3">%s Subtotal</td>
            <td class="amount">%s</td>
        </tr>`, category, formatKES(subtotal))
}

// Add grand total
reportContent += `
        <tr class="grand-total-row">
            <td colspan="3"><strong>Grand Total</strong></td>
            <td class="amount"><strong>` + formatKES(grandTotal) + `</strong></td>
        </tr>`

reportContent += `</tbody></table>` + reportFooter

	// Generate PDF
	pdfGen, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return models.Report{}, fmt.Errorf("failed to create PDF generator: %w", err)
	}
	pdfGen.AddPage(wkhtmltopdf.NewPageReader(strings.NewReader(reportContent)))

	if err = pdfGen.Create(); err != nil {
		return models.Report{}, fmt.Errorf("failed to create PDF: %w", err)
	}

	// Ensure reports directory exists
	reportsDir := "reports"
	if err := os.MkdirAll(reportsDir, os.ModePerm); err != nil {
		return models.Report{}, fmt.Errorf("failed to create reports directory: %w", err)
	}

	// Define the file path using RefNo and the date range
	fileName := fmt.Sprintf("sales_report_%s_%s.pdf", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	filePath := filepath.Join(reportsDir, fileName)

	// Save the PDF
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

	// Save report metadata in DB with report name based on file path
	report := models.Report{
		ReportType:  "Sales",
		Content:     absoluteFileURL,
		UserID:      userID,
		GeneratedAt: time.Now(),
		Name:        fmt.Sprintf("Sales Report from %s to %s", startDate.Format("Jan 2, 2006"), endDate.Format("Jan 2, 2006")),
		StartDate:   startDate, // Save start date
		EndDate:     endDate,   // Save end date
	}

	if err = r.DB.Create(&report).Error; err != nil {
		return models.Report{}, err
	}

	return report, nil
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
