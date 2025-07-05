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

	if len(flocks) == 0 {
		log.Println("No flocks found for user.")
		return "", fmt.Errorf("no flocks found for user")
	}

	var totalMortalityRate float64
	for _, flock := range flocks {
		if flock.BirdCount == 0 {
			log.Printf("Skipping flock %s with zero birds", flock.Name)
			continue
		}

		totalBirds += flock.BirdCount
		flockSummaries = append(flockSummaries, FlockSummary{
			Name:          flock.Name,
			BirdCount:     flock.BirdCount,
			MortalityRate: flock.MortalityRate,
			Revenue:       formatCurrency(flock.Revenue),
			Expenses:      formatCurrency(flock.Expenses),
		})

		totalMortalityRate += flock.MortalityRate
	}

	if len(flockSummaries) == 0 {
		log.Println("All flocks have zero birds; report generation aborted.")
		return "", fmt.Errorf("all flocks have zero birds, cannot generate report")
	}

	// Compute average mortality rate safely
	avgMortalityRate := 0.0
	if len(flockSummaries) > 0 {
		avgMortalityRate = math.Round((totalMortalityRate / float64(len(flockSummaries))) * 10) / 10
	}



	// Report Data Struct
	reportData := FlockReportData{
		Title:           "Flock Report",
		DateRange:       fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
		User:            user.Username,
		Email:           user.Email,
		Contact:         user.PhoneNumber,
		Summary:         fmt.Sprintf("Total birds: %d", totalBirds),
		Flocks:          flockSummaries,
		TotalBirds:      totalBirds,
		AvgMortalityRate: avgMortalityRate,
	}

	// Template Processing
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

	// Generate PDF
	outputDir := filepath.Join(baseDir, "pkg/reports/generated")
	_ = os.MkdirAll(outputDir, os.ModePerm)
	reportFilename := fmt.Sprintf("flock_report_%d.pdf", time.Now().Unix())
	pdfFilePath := filepath.Join(outputDir, reportFilename)
	relativePath := filepath.Join("pkg/reports/generated", reportFilename)

	log.Println("Generating PDF report...")
	cmd := exec.Command("weasyprint", "-", pdfFilePath)
	cmd.Stdin = bytes.NewReader(htmlBuffer.Bytes())

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Println("Error generating PDF:", err, "Details:", stderr.String())
		return "", fmt.Errorf("failed to generate PDF: %v\nDetails: %s", err, stderr.String())
	}

	// Save Report in Database
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
