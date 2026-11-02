package services

import (
	"testing"
	"time"

	"birdseye-backend/pkg/models"
)

func TestMonthlyFinancials(t *testing.T) {
	on := func(m time.Month, d int) time.Time { return time.Date(2026, m, d, 15, 0, 0, 0, time.UTC) }
	one, two := uint(1), uint(2)
	sales := []models.Sale{
		{FlockID: &one, Amount: 1000, Category: "Egg Sales", Date: on(10, 31)}, // last day of the month, afternoon
		{FlockID: &one, Amount: 500, Category: "Bird Sales", Date: on(11, 1)},
		{FlockID: &two, Amount: 300, Category: "Egg Sales", Date: on(11, 2)},
		{FlockID: nil, Amount: 999, Category: "Manure", Date: on(11, 2)},                  // farm-wide: skipped
		{FlockID: &one, Amount: 777, Date: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)}, // last year: skipped
	}
	expenses := []models.Expense{
		{FlockID: &one, Amount: 400, Date: on(10, 5)},
		{FlockID: nil, Amount: 50, Date: on(10, 5)},
	}
	stored := []models.FlocksFinancialData{{ID: 7, FlockID: 2, Month: 11, Year: 2026, Budget: 2500}}

	got := MonthlyFinancials(3, 2026, sales, expenses, stored)
	if len(got) != 3 {
		t.Fatalf("want 3 rows, got %+v", got)
	}
	oct, nov1, nov2 := got[0], got[1], got[2]
	if oct.FlockID != 1 || oct.Month != 10 || oct.Revenue != 1000 || oct.EggSales != 1000 || oct.Expenses != 400 || oct.NetRevenue != 600 || oct.FarmID != 3 {
		t.Fatalf("october: %+v", oct)
	}
	if nov1.FlockID != 1 || nov1.Month != 11 || nov1.Revenue != 500 || nov1.EggSales != 0 {
		t.Fatalf("november batch 1: %+v", nov1)
	}
	if nov2.FlockID != 2 || nov2.Revenue != 300 || nov2.Budget != 2500 || nov2.ID != 7 {
		t.Fatalf("november batch 2: %+v", nov2)
	}
}
