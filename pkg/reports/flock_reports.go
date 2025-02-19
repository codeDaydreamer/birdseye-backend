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
	 "math"
	"birdseye-backend/pkg/models"

	"github.com/wcharczuk/go-chart/v2"
	"gorm.io/gorm"
)

type FlockSummary struct {
	Name          string
	BirdCount     int
	MortalityRate float64
	Revenue       string
	Expenses      string
}

type FlockReportData struct {
	Title          string
	DateRange      string
	User           string
	Email          string
	Contact        string
	Summary        string
	Flocks         []FlockSummary
	TotalBirds     int
	ChartImagePath string
	AvgMortalityRate float64
}

func GenerateFlockReport(db *gorm.DB, userID uint, startDate, endDate time.Time) (string, error) {
	log.Println("Starting flock report generation...")

	var totalBirds int
	var flockSummaries []FlockSummary
	var chartValues []chart.Value

	log.Println("Fetching user details...")
	user, err := models.GetUserByID(userID)
	if err != nil {
		log.Println("Error retrieving user details:", err)
		return "", fmt.Errorf("failed to retrieve user details: %w", err)
	}

	log.Printf("Fetching flocks for user %d", userID)
	var flocks []models.Flock
	if err := db.Where("user_id = ?", userID).Find(&flocks).Error; err != nil {
		log.Println("Error fetching flocks:", err)
		return "", fmt.Errorf("failed to fetch flocks: %w", err)
	}

	for _, flock := range flocks {
		totalBirds += flock.BirdCount
		flockSummaries = append(flockSummaries, FlockSummary{
			Name:          flock.Name,
			BirdCount:     flock.BirdCount,
			MortalityRate: flock.MortalityRate,
			Revenue:       formatCurrency(flock.Revenue),
			Expenses:      formatCurrency(flock.Expenses),
		})
		chartValues = append(chartValues, chart.Value{
			Label: flock.Name,
			Value: float64(flock.BirdCount),
		})
	}
	var totalMortalityRate float64
for _, flock := range flockSummaries {
    totalMortalityRate += flock.MortalityRate
}


// Compute average mortality rate
avgMortalityRate := 0.0
if len(flockSummaries) > 0 {
    avgMortalityRate = math.Round((totalMortalityRate / float64(len(flockSummaries))) * 10) / 10
}



	log.Println("Generating flock mortality chart...")
	chartImagePath, err := generateFlockMortalityChart(chartValues)
	if err != nil {
		log.Println("Error generating flock chart:", err)
		return "", fmt.Errorf("failed to generate flock chart: %w", err)
	}

	reportData := FlockReportData{
		Title:          "Flock Report",
		DateRange:      fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
		User:           user.Username,
		Email:          user.Email,
		Contact:        user.Contact,
		Summary:        fmt.Sprintf("Total birds: %d", totalBirds),
		Flocks:         flockSummaries,
		TotalBirds:     totalBirds,
		ChartImagePath: chartImagePath,
		AvgMortalityRate: avgMortalityRate,
	}

	baseDir, _ := os.Getwd()
	templatePath := filepath.Join(baseDir, "pkg/reports/templates/flock_report_template.html")
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

	reportFilename := fmt.Sprintf("flock_report_%d.pdf", time.Now().Unix())
	pdfFilePath := filepath.Join(outputDir, reportFilename)
	relativePath := filepath.Join("pkg/reports/generated", reportFilename)

	log.Println("Generating PDF report...")
	cmd := exec.Command("wkhtmltopdf", "--enable-local-file-access", "-", pdfFilePath)
	cmd.Stdin = bytes.NewReader(htmlBuffer.Bytes())

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Println("Error generating PDF:", err, "Details:", stderr.String())
		return "", fmt.Errorf("failed to generate PDF: %v\nDetails: %s", err, stderr.String())
	}

	report := models.Report{
		ReportType:  "Flock",
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

	log.Println("Flock report generated successfully:", pdfFilePath)
	return pdfFilePath, nil
}

func generateFlockMortalityChart(values []chart.Value) (string, error) {
	log.Println("Rendering flock bird count chart...")

	if len(values) == 0 {
		log.Println("Chart generation skipped: No data available")
		return "", fmt.Errorf("invalid data range; cannot be zero")
	}

	baseDir, _ := os.Getwd()
	chartImagePath := filepath.Join(baseDir, "pkg/reports/generated/flock_mortality_chart.png")

	graph := chart.BarChart{
		Title:    "Flock Bird Count",
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
		Width:    800,
		Height:   500,
		BarWidth: 40,
		Bars:     values,
	}

	file, err := os.Create(chartImagePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if err := graph.Render(chart.PNG, file); err != nil {
		return "", err
	}

	log.Println("Flock bird count  saved at:", chartImagePath)
	return chartImagePath, nil
}
