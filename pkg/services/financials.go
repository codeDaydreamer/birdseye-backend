package services

import (
	"sort"
	"time"

	"birdseye-backend/pkg/models"

	"gorm.io/gorm"
)

type flockMonth struct {
	flock uint
	month int
}

// MonthlyFinancials adds up each batch's sales and its own expenses per month of the given year.
// Sales and expenses without a batch (farm-wide) are skipped, as are other years. Budgets are
// carried over from the stored rows, the only figure on them that people set by hand.
func MonthlyFinancials(farmID uint, year int, sales []models.Sale, expenses []models.Expense,
	stored []models.FlocksFinancialData) []models.FlocksFinancialData {
	rows := map[flockMonth]*models.FlocksFinancialData{}
	row := func(flock uint, month int) *models.FlocksFinancialData {
		k := flockMonth{flock, month}
		if rows[k] == nil {
			rows[k] = &models.FlocksFinancialData{FlockID: flock, FarmID: farmID, Month: month, Year: year}
		}
		return rows[k]
	}
	for _, s := range sales {
		if s.FlockID == nil || s.Date.Year() != year {
			continue
		}
		r := row(*s.FlockID, int(s.Date.Month()))
		r.Revenue += s.Amount
		if s.Category == "Egg Sales" {
			r.EggSales += s.Amount
		}
	}
	for _, e := range expenses {
		if e.FlockID == nil || e.Date.Year() != year {
			continue
		}
		row(*e.FlockID, int(e.Date.Month())).Expenses += e.Amount
	}
	for _, s := range stored {
		if s.Year == year && s.Budget != 0 {
			r := row(s.FlockID, s.Month)
			r.ID, r.Budget = s.ID, s.Budget
		}
	}

	out := make([]models.FlocksFinancialData, 0, len(rows))
	for _, r := range rows {
		r.NetRevenue = r.Revenue - r.Expenses
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FlockID != out[j].FlockID {
			return out[i].FlockID < out[j].FlockID
		}
		return out[i].Month < out[j].Month
	})
	return out
}

// FarmMonthlyFinancials works out /flock-financial live for this year, so it is always current
// (it used to be a cache refreshed only when the batch list was loaded, and only for the current
// month, so every new month started at zero)
func FarmMonthlyFinancials(db *gorm.DB, farmID uint) ([]models.FlocksFinancialData, error) {
	year := time.Now().Year()
	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0)

	var sales []models.Sale
	if err := db.Where("farm_id = ? AND flock_id IS NOT NULL AND date >= ? AND date < ?", farmID, start, end).
		Find(&sales).Error; err != nil {
		return nil, err
	}
	var expenses []models.Expense
	if err := db.Where("farm_id = ? AND flock_id IS NOT NULL AND date >= ? AND date < ?", farmID, start, end).
		Find(&expenses).Error; err != nil {
		return nil, err
	}
	var stored []models.FlocksFinancialData
	if err := db.Where("farm_id = ? AND year = ?", farmID, year).Find(&stored).Error; err != nil {
		return nil, err
	}
	return MonthlyFinancials(farmID, year, sales, expenses, stored), nil
}
