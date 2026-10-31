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
