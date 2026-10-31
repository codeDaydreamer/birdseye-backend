package models

import (
	"math"
	"testing"
	"time"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestEggSettingRates(t *testing.T) {
	set := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	candled := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	hatched := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)
	s := EggSetting{SetDate: set, EggsSet: 200, Clear: 20, Rotten: 10, CandledAt: &candled,
		Hatched: 150, Cull: 6, HatchedAt: &hatched, CostPerEgg: 30}
	s.FillComputed(time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC))

	if !s.DueDate.Equal(time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("due %v", s.DueDate)
	}
	if s.Day != 22 { // counted to the hatch day, not today
		t.Errorf("day %d", s.Day)
	}
	if !near(s.FertilityRate, 90) || !near(s.HatchOfSet, 75) || !near(s.HatchOfFertile, 150.0/180*100) {
		t.Errorf("rates %v %v %v", s.FertilityRate, s.HatchOfSet, s.HatchOfFertile)
	}
	if s.GoodChicks != 144 || !near(s.CostPerChick, 30*200/144.0) {
		t.Errorf("good %d cost %v", s.GoodChicks, s.CostPerChick)
	}
}

func TestEggSettingBeforeCandling(t *testing.T) {
	s := EggSetting{SetDate: time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC), EggsSet: 100}
	s.FillComputed(time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC))
	if s.Day != 7 || s.FertilityRate != 0 || s.HatchOfSet != 0 {
		t.Errorf("day %d fertility %v", s.Day, s.FertilityRate)
	}
}
