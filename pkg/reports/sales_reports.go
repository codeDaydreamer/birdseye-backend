package reports

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/wcharczuk/go-chart/v2"
	"gorm.io/gorm"
	"birdseye-backend/pkg/models"

)

type SalesCategorySummary struct {
	Category string
	Total    string
}

type FormattedSale struct {
	RefNo           string
	Product         string
	Category        string
	Description     string
	Quantity        int
	UnitPrice       string
	FormattedAmount string
	FormattedDate   string
}


type SalesReportData struct {
	Title           string
	DateRange       string
	User            string
	Email           string
	Contact         string
	Summary         string
	Sales           []FormattedSale
	CategorySummary []SalesCategorySummary
	TotalAmount     string
	ChartImagePath  string
}

func GenerateSalesReport(db *gorm.DB, userID uint, startDate, endDate time.Time) (string, error) {
	log.Println("Starting sales report generation...")

	var sales []models.Sale
	var categoryTotals = make(map[string]float64)
	var totalAmount float64

	log.Println("Fetching user details...")
	user, err := models.GetUserByID(userID)
	if err != nil {
		log.Println("Error retrieving user details:", err)
		return "", fmt.Errorf("failed to retrieve user details: %w", err)
	}

	log.Println("Fetching sales from the database...")
	if err := db.Where("user_id = ? AND date BETWEEN ? AND ?", userID, startDate, endDate).Find(&sales).Error; err != nil {
		log.Println("Error fetching sales:", err)
		return "", fmt.Errorf("failed to fetch sales: %w", err)
	}

	var formattedSales []FormattedSale
	for _, sale := range sales {
		totalAmount += sale.Amount
		categoryTotals[sale.Category] += sale.Amount
	
		formattedSales = append(formattedSales, FormattedSale{
			RefNo:           sale.RefNo,
			Product:         sale.Product,
			Category:        sale.Category,
			Description:     sale.Description,
			Quantity:        sale.Quantity, // Ensure your Sale model has a Quantity field
			UnitPrice:       formatCurrency(sale.UnitPrice), // Ensure UnitPrice is correctly formatted
			FormattedAmount: formatCurrency(sale.Amount),
			FormattedDate:   sale.Date.Format("Jan 2, 2006"),
		})
	}
	

	log.Println("Summarizing sales categories...")
	var categorySummary []SalesCategorySummary
	var chartValues []chart.Value

	for category, total := range categoryTotals {
		categorySummary = append(categorySummary, SalesCategorySummary{
			Category: category,
			Total:    formatCurrency(total),
		})
		chartValues = append(chartValues, chart.Value{
			Label: category,
			Value: total,
		})
	}

	log.Println("Generating sales chart...")
	chartImagePath, err := generateSalesChart(chartValues)
	if err != nil {
		log.Println("Error generating sales chart:", err)
		return "", fmt.Errorf("failed to generate sales chart: %w", err)
	}

	reportData := SalesReportData{
		Title:           "Sales Report",
		DateRange:       fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
		User:            user.Username,
		Email:           user.Email,
		Contact:         user.Contact,
		Summary:         fmt.Sprintf("Total sales recorded: %s", formatCurrency(totalAmount)),
		Sales:           formattedSales,
		CategorySummary: categorySummary,
		TotalAmount:     formatCurrency(totalAmount),
		ChartImagePath:  chartImagePath,
	}

	baseDir, _ := os.Getwd()
	templatePath := filepath.Join(baseDir, "pkg/reports/templates/sales_report_template.html")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Println("Error loading template:", err)
		return "", fmt.Errorf("failed to load template: %w", err)
	}

	var htmlBuffer bytes.Buffer
	if err := tmpl.Execute(&htmlBuffer, reportData); err != nil {
		log.Println("Error executing template:", err)
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	outputDir := filepath.Join(baseDir, "pkg/reports/generated")
	_ = os.MkdirAll(outputDir, os.ModePerm)

	reportFilename := fmt.Sprintf("sales_report_%d.pdf", time.Now().Unix())
	pdfFilePath := filepath.Join(outputDir, reportFilename)
	relativePath := filepath.Join("pkg/reports/generated", reportFilename)

	log.Println("Generating PDF report at:", pdfFilePath)
	cmd := exec.Command("wkhtmltopdf", "--enable-local-file-access", "-", pdfFilePath)
	cmd.Stdin = bytes.NewReader(htmlBuffer.Bytes())

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Println("Error generating PDF:", err, "Details:", stderr.String())
		return "", fmt.Errorf("failed to generate PDF: %v\nDetails: %s", err, stderr.String())
	}

	report := models.Report{
		ReportType:  "Sales",
		GeneratedAt: time.Now(),
		UserID:      userID,
		Name:        reportFilename,
		Content:     relativePath,
		StartDate:   startDate,
		EndDate:     endDate,
	}

	if err := db.Create(&report).Error; err != nil {
		log.Println("Error saving report to database:", err)
		return "", fmt.Errorf("failed to save report to database: %w", err)
	}

	log.Println("Sales report generated and saved successfully at:", pdfFilePath)
	return pdfFilePath, nil
}

func generateSalesChart(values []chart.Value) (string, error) {
	log.Println("Rendering sales chart...")
	baseDir, _ := os.Getwd()
	chartImagePath := filepath.Join(baseDir, "pkg/reports/generated/sales_chart.png")
	
	for i := range values {
		values[i].Label = fmt.Sprintf("%s\n(%s)", values[i].Label, formatCurrency(values[i].Value))
	}

	graph := chart.BarChart{
		Title: "Sales Breakdown by Category",
		TitleStyle: chart.Style{
			FontSize:  10, // Slightly larger for emphasis
			FontColor: chart.ColorBlack,
			Padding: chart.Box{
				Top:    1, // Adds space above the title
				Bottom: 20, // Adds space below to avoid overlap
				Left:   10,
				Right:  10,
			},
			TextWrap: chart.TextWrapWord, // Ensures text doesn’t overflow
		},
		Background: chart.Style{
			Padding: chart.Box{
				Top:    30,
				Bottom: 30,
				Left:   50,  // More space for y-axis labels
				Right:  30,
			},
		},
		Width:  800,
		Height: 500,
		BarWidth: 40,
		Bars:  values,
	}

	file, err := os.Create(chartImagePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if err := graph.Render(chart.PNG, file); err != nil {
		return "", err
	}

	log.Println("Sales chart saved at:", chartImagePath)
	return chartImagePath, nil
}


