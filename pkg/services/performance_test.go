package services

import (
	"math"
	"testing"
	"time"

	"birdseye-backend/pkg/models"
)

func TestWeightLeftKg(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC) }
	weighIns := []models.WeighIn{{Date: day(7), AvgWeightG: 500}, {Date: day(21), AvgWeightG: 1500}}
	sale := uint(9)
	moves := []models.BirdMovement{
		{Type: models.MoveSold, Delta: -100, Date: day(25)},               // 100 x 1.5 kg
		{Type: models.MoveCulled, Delta: -10, Date: day(10)},              // 10 x 0.5 kg
		{Type: models.MoveSold, Delta: -20, Date: day(26), SaleID: &sale}, // meat sale: its own kg
	}
	got := WeightLeftKg(moves, weighIns, map[uint]float64{9: 37})
	if want := 150.0 + 5 + 37; math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %v, want %v", got, want)
	}
	if WeightLeftKg(moves, nil, nil) != 0 {
		t.Fatal("no weigh-ins should give 0")
	}
}

func TestShareCosts(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC) }
	moves := []models.BirdMovement{
		{FlockID: 1, Delta: 300, Date: day(1)},
		{FlockID: 2, Delta: 100, Date: day(10)},
		{FlockID: 1, Delta: -100, Date: day(20)}, // sold
	}
	expenses := []models.Expense{
		{Amount: 1000, Date: day(5)},                                     // only batch 1 alive: all of it
		{Amount: 4000, Date: day(15)},                                    // 300 vs 100: 3000 / 1000
		{Amount: 900, Date: day(25)},                                     // 200 vs 100: 600 / 300
		{Amount: 500, Date: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)}, // before any birds: not shared
	}
	got := ShareCosts(expenses, moves)
	if math.Abs(got[1]-4600) > 1e-9 || math.Abs(got[2]-1300) > 1e-9 {
		t.Fatalf("got %v", got)
	}
}
