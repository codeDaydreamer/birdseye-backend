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

type InventorySummary struct {
	ItemName  string
	Quantity  int
	TotalCost string
}

type InventoryReportData struct {
	Title          string
	DateRange      string
	User           string
	Email          string
	Contact        string
	Summary        string
	InventoryItems []InventorySummary
	TotalValue     string
	ChartImagePath string
}

func GenerateInventoryReport(db *gorm.DB, userID uint, startDate, endDate time.Time) (string, error) {
	log.Println("Starting inventory report generation...")

	var inventoryItems []models.InventoryItem
	var totalValue float64

	log.Println("Fetching user details...")
	user, err := models.GetUserByID(userID)
	if err != nil {
		log.Println("Error retrieving user details:", err)
		return "", fmt.Errorf("failed to retrieve user details: %w", err)
	}

	log.Println("Fetching inventory items from the database...")
	if err := db.Where("user_id = ?", userID).Find(&inventoryItems).Error; err != nil {
		log.Println("Error fetching inventory items:", err)
		return "", fmt.Errorf("failed to fetch inventory items: %w", err)
	}

	var formattedInventory []InventorySummary
	var chartValues []chart.Value

	for _, item := range inventoryItems {
		itemTotalCost := float64(item.Quantity) * item.CostPerUnit
		totalValue += itemTotalCost

		formattedInventory = append(formattedInventory, InventorySummary{
			ItemName:  item.ItemName,
			Quantity:  item.Quantity,
			TotalCost: formatCurrency(itemTotalCost),
		})

		chartValues = append(chartValues, chart.Value{
			Label: item.ItemName,
			Value: float64(item.Quantity), // ✅ Uses Quantity instead of CostPerUnit
		})
		
	}

	log.Println("Generating inventory chart...")
	chartImagePath, err := generateInventoryChart(chartValues)
	if err != nil {
		log.Println("Error generating inventory chart:", err)
		return "", fmt.Errorf("failed to generate inventory chart: %w", err)
	}

	reportData := InventoryReportData{
		Title:          "Inventory Report",
		DateRange:      fmt.Sprintf("As of %s", endDate.Format("2006-01-02")),
		User:           user.Username,
		Email:          user.Email,
		Contact:        user.PhoneNumber,
		Summary:        fmt.Sprintf("Total inventory value: %s", formatCurrency(totalValue)),
		InventoryItems: formattedInventory,
		TotalValue:     formatCurrency(totalValue),
		ChartImagePath: chartImagePath,
	}

	baseDir, _ := os.Getwd()
	templatePath := filepath.Join(baseDir, "pkg/reports/templates/inventory_report_template.html")
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

	reportFilename := fmt.Sprintf("inventory_report_%d.pdf", time.Now().Unix())
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
		ReportType:  "Inventory",
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

	log.Println("Inventory report generated and saved successfully at:", pdfFilePath)
	return pdfFilePath, nil
}

func generateInventoryChart(values []chart.Value) (string, error) {
	log.Println("Rendering inventory chart...")
	baseDir, _ := os.Getwd()
	chartImagePath := filepath.Join(baseDir, "pkg/reports/generated/inventory_chart.png")

	graph := chart.BarChart{
		Title: "Inventory Levels",
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

	log.Println("Inventory chart saved at:", chartImagePath)
	return chartImagePath, nil
}