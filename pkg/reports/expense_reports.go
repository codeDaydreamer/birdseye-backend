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
	"github.com/dustin/go-humanize"
)

type ExpenseCategorySummary struct {
	Category string
	Total    string
}

type FormattedExpense struct {
	Category        string
	Description     string
	FormattedAmount string
	FormattedDate   string // Store formatted date as a string
}

type ExpenseReportData struct {
	Title            string
	DateRange        string
	User             string
	Email            string
	Contact          string
	Summary          string
	Expenses         []FormattedExpense
	CategorySummary  []ExpenseCategorySummary
	TotalAmount      string
	ChartImagePath   string
}
func GenerateExpenseReport(db *gorm.DB, userID uint, startDate, endDate time.Time) (string, error) {
	log.Println("Starting expense report generation...")

	// Fetch expenses and calculate totals per category.
	var expenses []models.Expense
	var categoryTotals = make(map[string]float64)
	var totalAmount float64

	log.Println("Fetching user details...")
	user, err := models.GetUserByID(userID)
	if err != nil {
		log.Println("Error retrieving user details:", err)
		return "", fmt.Errorf("failed to retrieve user details: %w", err)
	}

	log.Println("Fetching expenses from the database...")
	if err := db.Where("user_id = ? AND date BETWEEN ? AND ?", userID, startDate, endDate).Find(&expenses).Error; err != nil {
		log.Println("Error fetching expenses:", err)
		return "", fmt.Errorf("failed to fetch expenses: %w", err)
	}

	// Format expense data for report.
	var formattedExpenses []FormattedExpense
	for _, expense := range expenses {
		totalAmount += expense.Amount
		categoryTotals[expense.Category] += expense.Amount
	
		formattedExpenses = append(formattedExpenses, FormattedExpense{
			Category:        expense.Category,
			Description:     expense.Description,
			FormattedAmount: formatCurrency(expense.Amount),
			FormattedDate:   expense.Date.Format("Jan 2, 2006"),
		})
	}

	// Prepare category summary for chart.
	log.Println("Summarizing expense categories...")
	var categorySummary []ExpenseCategorySummary
	var chartValues []chart.Value
	for category, total := range categoryTotals {
		categorySummary = append(categorySummary, ExpenseCategorySummary{
			Category: category,
			Total:    formatCurrency(total),
		})
		chartValues = append(chartValues, chart.Value{
			Label: category,
			Value: total,
		})
	}

	// Ensure there's data for the chart (default to 0 if none exists).
	if len(chartValues) == 0 {
		chartValues = append(chartValues, chart.Value{
			Label: "No Data",
			Value: 0,
		})
	}

	log.Println("Generating expense chart...")
	chartImagePath, err := generateExpenseChart(chartValues)
	if err != nil {
		log.Println("Error generating expense chart:", err)
		return "", fmt.Errorf("failed to generate expense chart: %w", err)
	}

	// Prepare report data.
	reportData := ExpenseReportData{
		Title:           "Expense Report",
		DateRange:       fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
		User:            user.Username,
		Email:           user.Email,
		Contact:         user.PhoneNumber,
		Summary:         fmt.Sprintf("Total expenses recorded: %s", formatCurrency(totalAmount)),
		Expenses:        formattedExpenses,
		CategorySummary: categorySummary,
		TotalAmount:     formatCurrency(totalAmount),
		ChartImagePath:  chartImagePath,
	}

	// Load the HTML template.
	baseDir, _ := os.Getwd()
	templatePath := filepath.Join(baseDir, "pkg/reports/templates/expense_report_template.html")
	log.Println("Loading template from:", templatePath)
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Println("Error loading template:", err)
		return "", fmt.Errorf("failed to load template: %w", err)
	}

	// Execute the template.
	var htmlBuffer bytes.Buffer
	log.Println("Executing template...")
	if err := tmpl.Execute(&htmlBuffer, reportData); err != nil {
		log.Println("Error executing template:", err)
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Create the output directory and generate PDF.
	outputDir := filepath.Join(baseDir, "pkg/reports/generated")
	_ = os.MkdirAll(outputDir, os.ModePerm)

	reportFilename := fmt.Sprintf("expense_report_%d.pdf", time.Now().Unix())
	pdfFilePath := filepath.Join(outputDir, reportFilename)
	relativePath := filepath.Join("pkg/reports/generated", reportFilename)

	// Use WeasyPrint for PDF generation.
	log.Println("Generating PDF report at:", pdfFilePath)
	cmd := exec.Command("weasyprint", "-", pdfFilePath)
	cmd.Stdin = bytes.NewReader(htmlBuffer.Bytes())

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Println("Error generating PDF:", err, "Details:", stderr.String())
		return "", fmt.Errorf("failed to generate PDF: %v\nDetails: %s", err, stderr.String())
	}

	// Save report details to the database.
	log.Println("Saving report details to database...")
	report := models.Report{
		ReportType:  "Expense",
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

	log.Println("Expense report generated and saved successfully at:", pdfFilePath)
	return pdfFilePath, nil
}

func generateExpenseChart(values []chart.Value) (string, error) {
    log.Println("Rendering expense chart...")

    // Always ensure at least one value exists
    if len(values) == 0 {
        values = []chart.Value{{Label: "No Data", Value: 0}}
    }

    // Add detailed labels (e.g., include currency)
    for i := range values {
        values[i].Label = fmt.Sprintf("%s\n($%.2f)", values[i].Label, values[i].Value)
    }

    // Find max value, default to 1 if all zeros
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

    // Ensure the output directory exists before creating the file
    if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
        return "", fmt.Errorf("failed to create output directory: %w", err)
    }

    chartImagePath := filepath.Join(outputDir, "expense_chart.png")

    file, err := os.Create(chartImagePath)
    if err != nil {
        return "", fmt.Errorf("failed to create chart image file: %w", err)
    }
    defer file.Close()

    graph := chart.BarChart{
        Title:    "Expenses by Category",
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
                Max: maxValue * 1.1, // add headroom
            },
        },
    }

    if err := graph.Render(chart.PNG, file); err != nil {
        return "", fmt.Errorf("failed to render chart: %w", err)
    }

    log.Println("Expense chart saved at:", chartImagePath)
    return chartImagePath, nil
}

func formatCurrency(amount float64) string {
	return fmt.Sprintf("KES %s", humanize.Commaf(amount))
}
