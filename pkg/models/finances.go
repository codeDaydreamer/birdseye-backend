package models

import (
	"time"
)

// FlockFinancialData stores financial data for each flock for a given time period
type FlockFinancialData struct {
	ID             uint      `gorm:"primaryKey"`
	FlockID        uint      `json:"flock_id"`
	UserID         uint      `json:"user_id"`
	PeriodStart    time.Time `json:"period_start"` // Start of the time period
	PeriodEnd      time.Time `json:"period_end"`   // End of the time period
	TotalRevenue   float64   `json:"total_revenue"`
	TotalExpenses  float64   `json:"total_expenses"`
	NetProfit      float64   `json:"net_profit"`
	ProfitMargin   float64   `json:"profit_margin"`
	CostPerUnit    float64   `json:"cost_per_unit"`
	ExpenseRatio   float64   `json:"expense_ratio"`
	InventoryCost  float64   `json:"inventory_cost"` // Optional if you want to store inventory cost separately
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func GetCurrentWeekPeriod() (time.Time, time.Time) {
	startOfWeek := time.Now().AddDate(0, 0, -int(time.Now().Weekday()))
	endOfWeek := startOfWeek.Add(7 * 24 * time.Hour)
	return startOfWeek, endOfWeek
}

func GetCurrentMonthPeriod() (time.Time, time.Time) {
	startOfMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Local)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)
	return startOfMonth, endOfMonth
}

func GetCurrentYearPeriod() (time.Time, time.Time) {
	startOfYear := time.Date(time.Now().Year(), 1, 1, 0, 0, 0, 0, time.Local)
	endOfYear := startOfYear.AddDate(1, 0, 0)
	return startOfYear, endOfYear
}

func GetCurrentDayPeriod() (time.Time, time.Time) {
	startOfDay := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)
	endOfDay := startOfDay.Add(24 * time.Hour)
	return startOfDay, endOfDay
}
