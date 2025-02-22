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

type FinancialCategorySummary struct {
	Category string
	Total    string
}

type FormattedFlockFinancial struct {
	FlockID    uint
	FlockName  string
	Month      string
	Year       int
	Revenue    string
	EggSales   string
	Expenses   string
	NetRevenue string
}

type FinancialReportData struct {
	Title           string
	DateRange       string
	Year            int
	User            string
	Email           string
	Contact         string
	Summary         string
	FinancialData   []FormattedFlockFinancial
	TotalRevenue    string
	TotalEggSales   string
	TotalExpenses   string
	TotalNetRevenue string
	ChartImagePath  string
}

func getMonthName(month int) string {
	months := []string{
		"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}
	if month >= 1 && month <= 12 {
		return months[month-1]
	}
	return "Unknown"
}

func getReportDateRange(month, year int) (time.Time, time.Time) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 0, 21) // First 21 days of the month
	return startDate, endDate
}

func GenerateFinancialReport(db *gorm.DB, userID uint) (string, error) {
	log.Println("Starting financial report generation...")

	currentYear := time.Now().Year()

	var financialData []models.FlocksFinancialData
	var totalRevenue, totalEggSales, totalExpenses, totalNetRevenue float64

	log.Println("Fetching user details...")
	user, err := models.GetUserByID(userID)
	if err != nil {
		log.Println("Error retrieving user details:", err)
		return "", fmt.Errorf("failed to retrieve user details: %w", err)
	}

	log.Println("Fetching financial data for all flocks of the user...")
	if err := db.Where("user_id = ? AND year = ?", userID, currentYear).Find(&financialData).Error; err != nil {
		log.Println("Error fetching financial data:", err)
		return "", fmt.Errorf("failed to fetch financial data: %w", err)
	}

	var formattedFinancialData []FormattedFlockFinancial
	var chartValues []chart.Value
	var startDate, endDate time.Time

	for _, data := range financialData {
		var flock models.Flock
		if err := db.Where("id = ?", data.FlockID).First(&flock).Error; err != nil {
			log.Println("Error fetching flock details:", err)
			continue
		}

		totalRevenue += data.Revenue
		totalEggSales += data.EggSales
		totalExpenses += data.Expenses
		totalNetRevenue += data.NetRevenue

		formattedFinancialData = append(formattedFinancialData, FormattedFlockFinancial{
			FlockID:    flock.ID,
			FlockName:  flock.Name,
			Month:      getMonthName(data.Month),
			Year:       data.Year,
			Revenue:    formatCurrency(data.Revenue),
			EggSales:   formatCurrency(data.EggSales),
			Expenses:   formatCurrency(data.Expenses),
			NetRevenue: formatCurrency(data.NetRevenue),
		})

		chartValues = append(chartValues, chart.Value{
			Label: fmt.Sprintf("%s (%s)", getMonthName(data.Month), flock.Name),
			Value: data.NetRevenue,
		})

		if startDate.IsZero() || data.Month < int(startDate.Month()) {
			startDate, endDate = getReportDateRange(data.Month, data.Year)
		}
	}

	log.Println("Generating financial chart...")
	chartImagePath, err := generateFinancialChart(chartValues)
	if err != nil {
		log.Println("Error generating financial chart:", err)
		return "", fmt.Errorf("failed to generate financial chart: %w", err)
	}

	reportData := FinancialReportData{
		Title:         "Financial Report",
		DateRange:     fmt.Sprintf("%s %d - %s %d", startDate.Format("Jan 2"), startDate.Year(), endDate.Format("Jan 2"), endDate.Year()),
		Year:          currentYear,
		User:          user.Username,
		Email:         user.Email,
		Contact:       user.Contact,
		Summary:       fmt.Sprintf("Total revenue: %s, Egg sales: %s, Expenses: %s, Net revenue: %s", formatCurrency(totalRevenue), formatCurrency(totalEggSales), formatCurrency(totalExpenses), formatCurrency(totalNetRevenue)),
		FinancialData:   formattedFinancialData,
		TotalRevenue:    formatCurrency(totalRevenue),
		TotalEggSales:   formatCurrency(totalEggSales),
		TotalExpenses:   formatCurrency(totalExpenses),
		TotalNetRevenue: formatCurrency(totalNetRevenue),
		ChartImagePath:  chartImagePath,
	}

	baseDir, _ := os.Getwd()
	templatePath := filepath.Join(baseDir, "pkg/reports/templates/financial_report_template.html")
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

	reportFilename := fmt.Sprintf("financial_report_%d.pdf", time.Now().Unix())
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
		ReportType:  "Financial",
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

	log.Println("Financial report generated and saved successfully at:", pdfFilePath)
	return pdfFilePath, nil
}


func generateFinancialChart(values []chart.Value) (string, error) {
	log.Println("Rendering financial chart...")
	baseDir, _ := os.Getwd()
	chartImagePath := filepath.Join(baseDir, "pkg/reports/generated/financial_chart.png")

	for i := range values {
		values[i].Label = fmt.Sprintf("%s\n(%s)", values[i].Label, formatCurrency(values[i].Value))
	}

	graph := chart.BarChart{
		Title: "Financial Breakdown by Net Revenue",
		TitleStyle: chart.Style{
			FontSize:  10, 
			FontColor: chart.ColorBlack,
			Padding: chart.Box{
				Top:    1,
				Bottom: 20,
				Left:   10,
				Right:  10,
			},
			TextWrap: chart.TextWrapWord,
		},
		Background: chart.Style{
			Padding: chart.Box{
				Top:    30,
				Bottom: 30,
				Left:   50,  
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

	log.Println("Financial chart saved at:", chartImagePath)
	return chartImagePath, nil
}
