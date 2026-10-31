package models

import (
	"math"
	"testing"
	"time"
)

func moves(rows ...BirdMovement) []BirdMovement { return rows }

func TestTotalsSoldBirdsAreNotDeaths(t *testing.T) {
	got := TotalsFromMovements(moves(
		BirdMovement{Type: MovePlaced, Delta: 500},
		BirdMovement{Type: MoveDied, Delta: -10},
		BirdMovement{Type: MoveSold, Delta: -100},
	))
	if got.Birds != 390 || got.Initial != 500 || got.Deaths != 10 || got.Sold != 100 {
		t.Fatalf("got %+v", got)
	}
	if math.Abs(got.Mortality-2) > 1e-9 {
		t.Fatalf("mortality = %v, want 2 (the old formula gave 22)", got.Mortality)
	}
}

func TestTotalsTransfersCulledAndAdjust(t *testing.T) {
	got := TotalsFromMovements(moves(
		BirdMovement{Type: MovePlaced, Delta: 200},
		BirdMovement{Type: MoveTransferIn, Delta: 50},
		BirdMovement{Type: MoveTransferOut, Delta: -30},
		BirdMovement{Type: MoveCulled, Delta: -5},
		BirdMovement{Type: MoveAdjust, Delta: -2},
		BirdMovement{Type: MoveDied, Delta: -25},
	))
	if got.Initial != 250 || got.Birds != 188 || got.Culled != 5 || got.Deaths != 25 {
		t.Fatalf("got %+v", got)
	}
	if math.Abs(got.Mortality-10) > 1e-9 {
		t.Fatalf("mortality = %v, want 10", got.Mortality)
	}
}

func TestTotalsEmpty(t *testing.T) {
	if got := TotalsFromMovements(nil); got != (FlockTotals{}) {
		t.Fatalf("got %+v", got)
	}
}

func TestFlockStage(t *testing.T) {
	cases := []struct {
		kind string
		days int
		want string
	}{
		{ProductionLayers, 0, "chick"},
		{ProductionLayers, 41, "chick"},
		{ProductionLayers, 42, "grower"},
		{ProductionLayers, 125, "grower"},
		{ProductionLayers, 126, "laying"},
		{ProductionLayers, 504, "spent"},
		{ProductionKienyeji, 130, "laying"},
		{ProductionBreeders, 60, "grower"},
		{ProductionBroilers, 20, "chick"},
		{ProductionBroilers, 21, "finisher"},
		{ProductionChicks, 41, "chick"},
		{ProductionChicks, 42, "grower"},
	}
	for _, c := range cases {
		if got := FlockStage(c.kind, c.days, false); got != c.want {
			t.Errorf("%s at %d days = %q, want %q", c.kind, c.days, got, c.want)
		}
	}
	if got := FlockStage(ProductionLayers, 200, true); got != "closed" {
		t.Errorf("closed batch stage = %q", got)
	}
}

func TestFillComputedAgeFromPlacement(t *testing.T) {
	now := time.Date(2026, 10, 31, 15, 0, 0, 0, time.UTC)
	// Bought as 2-week-old chicks 10 days ago: 24 days old today
	f := Flock{ProductionType: ProductionLayers, Status: FlockActive,
		PlacedAt: time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC), AgeAtPlacementDays: 14}
	f.FillComputed(now)
	if f.AgeDays != 24 || f.AgeWeeks != 3 || f.Stage != "chick" {
		t.Fatalf("got age %d days / %d weeks, stage %s", f.AgeDays, f.AgeWeeks, f.Stage)
	}
	if want := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC); !f.HatchDate.Equal(want) {
		t.Fatalf("hatch date %v, want %v", f.HatchDate, want)
	}

	// A closed batch stops ageing on the day it closed
	closed := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
	f.Status, f.ClosedAt = FlockClosed, &closed
	f.FillComputed(now)
	if f.AgeDays != 18 || f.Stage != "closed" {
		t.Fatalf("closed: age %d, stage %s", f.AgeDays, f.Stage)
	}
}
