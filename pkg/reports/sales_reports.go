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
	var salesByDate = make(map[string]float64)
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
		dateKey := sale.Date.Format("2006-01-02") // Format as YYYY-MM-DD
		salesByDate[dateKey] += sale.Amount // Aggregate sales by date
	
		formattedSales = append(formattedSales, FormattedSale{
			RefNo:           sale.RefNo,
			Product:         sale.Product,
			Category:        sale.Category,
			Description:     sale.Description,
			Quantity:        sale.Quantity,
			UnitPrice:       formatCurrency(sale.UnitPrice),
			FormattedAmount: formatCurrency(sale.Amount),
			FormattedDate:   sale.Date.Format("Jan 2, 2006"),
		})
	}

	log.Println("Generating sales trend chart...")
	chartImagePath, err := generateSalesTrendChart(salesByDate, startDate, endDate)
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
	cmd := exec.Command("weasyprint", "-", pdfFilePath)
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

func generateSalesTrendChart(salesByDate map[string]float64, startDate, endDate time.Time) (string, error) {
	log.Println("Rendering sales trend chart...")
	baseDir, _ := os.Getwd()
	chartImagePath := filepath.Join(baseDir, "pkg/reports/generated/sales_trend_chart.png")

	// Ensure at least one default point if no sales exist
	var xValues []time.Time
	var yValues []float64
	currentDate := startDate

	for !currentDate.After(endDate) {
		dateStr := currentDate.Format("2006-01-02")
		xValues = append(xValues, currentDate)
		yValues = append(yValues, salesByDate[dateStr])
		currentDate = currentDate.AddDate(0, 0, 1) // Move to next day
	}

	// Prevent empty chart rendering
	if len(xValues) == 0 {
		xValues = append(xValues, startDate)
		yValues = append(yValues, 0)
	}

	graph := chart.Chart{
		Title: "Sales Trend Over Time",
		TitleStyle: chart.Style{
			FontSize:  12,
			FontColor: chart.ColorBlack,
		},
		Width:  800,
		Height: 500,
		Series: []chart.Series{
			chart.TimeSeries{
				Name:    "Sales Amount",
				XValues: xValues,
				YValues: yValues,
				Style: chart.Style{
				
					StrokeColor: chart.ColorBlue,
					StrokeWidth: 2,
				},
			},
		},
		XAxis: chart.XAxis{
			Name: "Date",
			Style: chart.Style{
				
			},
			TickPosition: chart.TickPositionBetweenTicks,
		},
		YAxis: chart.YAxis{
			Name: "Sales Amount",
			Style: chart.Style{
				
			},
		},
	}

	file, err := os.Create(chartImagePath)
	if err != nil {
		return "", fmt.Errorf("failed to create chart file: %w", err)
	}
	defer file.Close()

	if err := graph.Render(chart.PNG, file); err != nil {
		return "", fmt.Errorf("failed to render chart: %w", err)
	}

	log.Println("Sales trend chart saved at:", chartImagePath)
	return chartImagePath, nil
}
