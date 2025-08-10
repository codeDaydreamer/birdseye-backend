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

	"birdseye-backend/pkg/models"

	"github.com/wcharczuk/go-chart/v2"
	"gorm.io/gorm"
)

type EggProductionSummary struct {
	FlockName    string
	TotalEggs    int
	TotalRevenue string
}

type FormattedEggProduction struct {
	FlockName     string
	EggsCollected int
	PricePerUnit  string
	TotalRevenue  string
	FormattedDate string
}

type EggProductionReportData struct {
	Title          string
	DateRange      string
	User           string
	Email          string
	Contact        string
	Summary        string
	EggProductions []FormattedEggProduction
	FlockSummaries []EggProductionSummary
	TotalEggs      int
	TotalRevenue   string
	ChartImagePath string
}

func GenerateEggProductionReport(db *gorm.DB, userID uint, startDate, endDate time.Time) (string, error) {
	log.Println("Starting egg production report generation...")

	var flockTotals = make(map[string]int)
	var revenueTotals = make(map[string]float64)
	var totalEggs int
	var totalRevenue float64

	log.Println("Fetching user details...")
	user, err := models.GetUserByID(userID)
	if err != nil {
		log.Println("Error retrieving user details:", err)
		return "", fmt.Errorf("failed to retrieve user details: %w", err)
	}

	log.Printf("Fetching egg production records for user %d between %s and %s", userID, startDate, endDate)

	// Struct to hold query results
	var productions []struct {
		FlockName     string
		EggsCollected int
		PricePerUnit  float64
		TotalRevenue  float64
		DateProduced  time.Time
	}

	// Perform the JOIN to fetch flock names
	if err := db.Table("egg_productions").
		Select("flocks.name AS flock_name, egg_productions.eggs_collected, egg_productions.price_per_unit, egg_productions.date_produced, (egg_productions.eggs_collected * egg_productions.price_per_unit) AS total_revenue").
		Joins("JOIN flocks ON flocks.id = egg_productions.flock_id").
		Where("egg_productions.user_id = ? AND egg_productions.date_produced BETWEEN ? AND ?", userID, startDate, endDate).
		Scan(&productions).Error; err != nil {
		log.Println("Error fetching egg production records:", err)
		return "", fmt.Errorf("failed to fetch egg production records: %w", err)
	}

	log.Printf("Total records fetched: %d", len(productions))

	var formattedProductions []FormattedEggProduction
	for _, production := range productions {
		totalEggs += production.EggsCollected
		totalRevenue += production.TotalRevenue
		flockTotals[production.FlockName] += production.EggsCollected
		revenueTotals[production.FlockName] += production.TotalRevenue

		formattedProductions = append(formattedProductions, FormattedEggProduction{
			FlockName:     production.FlockName,
			EggsCollected: production.EggsCollected,
			PricePerUnit:  formatCurrency(production.PricePerUnit),
			TotalRevenue:  formatCurrency(production.TotalRevenue),
			FormattedDate: production.DateProduced.Format("Jan 2, 2006"),
		})
	}

	log.Println("Summarizing flock production...")
	var flockSummaries []EggProductionSummary
	var chartValues []chart.Value

	for flock, eggs := range flockTotals {
		flockSummaries = append(flockSummaries, EggProductionSummary{
			FlockName:    flock,
			TotalEggs:    eggs,
			TotalRevenue: formatCurrency(revenueTotals[flock]),
		})
		found := false
		for i := range chartValues {
			if chartValues[i].Label == flock {
				chartValues[i].Value += float64(eggs)
				found = true
				break
			}
		}
		if !found {
			chartValues = append(chartValues, chart.Value{
				Label: flock,
				Value: float64(eggs),
			})
		}
	}

	log.Printf("Total flocks: %d", len(flockTotals))

	log.Printf("Total chart values: %d", len(chartValues))
	log.Println("Chart Values Debug:", chartValues)
	for _, v := range chartValues {
		log.Printf("Label: %s, Value: %.2f", v.Label, v.Value)
	}

	log.Println("Generating egg production chart...")
	chartImagePath, err := generateEggProductionChart(chartValues)
	if err != nil {
		log.Println("Error generating egg production chart:", err)
		return "", fmt.Errorf("failed to generate egg production chart: %w", err)
	}

	reportData := EggProductionReportData{
		Title:          "Egg Production Report",
		DateRange:      fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
		User:           user.Username,
		Email:          user.Email,
		Contact:        user.PhoneNumber,
		Summary:        fmt.Sprintf("Total eggs collected: %d", totalEggs),
		EggProductions: formattedProductions,
		FlockSummaries: flockSummaries,
		TotalEggs:      totalEggs,
		TotalRevenue:   formatCurrency(totalRevenue),
		ChartImagePath: chartImagePath,
	}

	baseDir, _ := os.Getwd()
	templatePath := filepath.Join(baseDir, "pkg/reports/templates/egg_production_report_template.html")
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

	reportFilename := fmt.Sprintf("egg_production_report_%d.pdf", time.Now().Unix())
	pdfFilePath := filepath.Join(outputDir, reportFilename)
	relativePath := filepath.Join("pkg/reports/generated", reportFilename)

	log.Println("Generating PDF report using WeasyPrint...")
	cmd := exec.Command("weasyprint", "-", pdfFilePath)
	cmd.Stdin = bytes.NewReader(htmlBuffer.Bytes())

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Println("Error generating PDF:", err, "Details:", stderr.String())
		return "", fmt.Errorf("failed to generate PDF: %v\nDetails: %s", err, stderr.String())
	}

	report := models.Report{
		ReportType:  "Egg Production",
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

	log.Println("Egg production report generated successfully:", pdfFilePath)
	return pdfFilePath, nil
}


func generateEggProductionChart(values []chart.Value) (string, error) {
    log.Println("Rendering egg production chart...")

    if len(values) == 0 {
        values = []chart.Value{{Label: "No Data", Value: 0}}
    }

    for i := range values {
        values[i].Label = fmt.Sprintf("%s\n(%d eggs)", values[i].Label, int(values[i].Value))
    }

    var maxValue float64
    for _, v := range values {
        if v.Value > maxValue {
            maxValue = v.Value
        }
    }
    if maxValue == 0 {
        maxValue = 1
    }

    baseDir, _ := os.Getwd()
    outputDir := filepath.Join(baseDir, "pkg/reports/generated")

    // Ensure directory exists before creating file
    if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
        return "", fmt.Errorf("failed to create output directory: %w", err)
    }

    chartImagePath := filepath.Join(outputDir, "egg_production_chart.png")

    file, err := os.Create(chartImagePath)
    if err != nil {
        return "", fmt.Errorf("failed to create chart image file: %w", err)
    }
    defer file.Close()

    graph := chart.BarChart{
        Title: "Egg Production by Flock",
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
        Width:    800,
        Height:   500,
        BarWidth: 40,
        Bars:     values,
        YAxis: chart.YAxis{
            Range: &chart.ContinuousRange{
                Min: 0,
                Max: maxValue * 1.1,
            },
        },
    }

    if err := graph.Render(chart.PNG, file); err != nil {
        return "", fmt.Errorf("failed to render chart: %w", err)
    }

    log.Println("Egg production chart saved at:", chartImagePath)
    return chartImagePath, nil
}
