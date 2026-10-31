package services

import (
	"fmt"

	"birdseye-backend/pkg/models"

	"gorm.io/gorm"
)

// EggStock is a batch's (or the farm's) eggs: collected, lost to adjustments (breakage, eaten, given away),
// sold, set in an incubator, and what's left
type EggStock struct {
	FlockID   uint   `json:"flock_id,omitempty"`
	FlockName string `json:"flock_name,omitempty"`
	Collected int    `json:"collected"`
	Adjusted  int    `json:"adjusted"`
	Sold      int    `json:"sold"`
	Set       int    `json:"set"` // put in an incubator to hatch
	OnHand    int    `json:"on_hand"`
}

// eggSoldSQL counts a sale's eggs: trays hold models.EggsPerTray
var eggSoldSQL = fmt.Sprintf(
	"COALESCE(SUM(CASE WHEN unit = 'tray' THEN quantity * %d ELSE quantity END), 0)", models.EggsPerTray)

// EggStockByFlock works out every batch's eggs on hand from collections, adjustments and egg sales.
// excludeSaleID leaves one sale out (to check an edit of it against the stock it would replace).
func EggStockByFlock(tx *gorm.DB, farmID uint, excludeSaleID uint) ([]EggStock, error) {
	var flocks []models.Flock
	if err := tx.Select("id", "name").Where("farm_id = ?", farmID).Order("id").Find(&flocks).Error; err != nil {
		return nil, err
	}
	type row struct {
		FlockID uint
		N       int
	}
	sumBy := func(q *gorm.DB, column, expr string) (map[uint]int, error) {
		var rows []row
		if err := q.Select(column + " AS flock_id, " + expr + " AS n").Group(column).Scan(&rows).Error; err != nil {
			return nil, err
		}
		m := map[uint]int{}
		for _, r := range rows {
			m[r.FlockID] = r.N
		}
		return m, nil
	}
	sum := func(q *gorm.DB, expr string) (map[uint]int, error) {
		return sumBy(q, "flock_id", expr)
	}
	collected, err := sum(tx.Model(&models.EggProduction{}).Where("farm_id = ?", farmID),
		"COALESCE(SUM(eggs_collected), 0)")
	if err != nil {
		return nil, err
	}
	adjusted, err := sum(tx.Model(&models.EggAdjustment{}).Where("farm_id = ?", farmID),
		"COALESCE(SUM(quantity), 0)")
	if err != nil {
		return nil, err
	}
	sold, err := sum(tx.Model(&models.Sale{}).
		Where("farm_id = ? AND product_type IN ? AND id <> ? AND flock_id IS NOT NULL",
			farmID, eggProductTypes(), excludeSaleID), eggSoldSQL)
	if err != nil {
		return nil, err
	}

	set, err := sumBy(tx.Model(&models.EggSetting{}).Where("farm_id = ? AND source = 'own'", farmID),
		"source_flock_id", "COALESCE(SUM(eggs_set), 0)")
	if err != nil {
		return nil, err
	}

	stock := make([]EggStock, 0, len(flocks))
	for _, f := range flocks {
		s := EggStock{FlockID: f.ID, FlockName: f.Name, Collected: collected[f.ID], Adjusted: adjusted[f.ID],
			Sold: sold[f.ID], Set: set[f.ID]}
		s.OnHand = s.Collected - s.Adjusted - s.Sold - s.Set
		stock = append(stock, s)
	}
	return stock, nil
}

// FarmEggStock adds the batches up
func FarmEggStock(flocks []EggStock) EggStock {
	var t EggStock
	for _, f := range flocks {
		t.Collected += f.Collected
		t.Adjusted += f.Adjusted
		t.Sold += f.Sold
		t.Set += f.Set
		t.OnHand += f.OnHand
	}
	return t
}

func eggProductTypes() []string {
	var types []string
	for k, p := range models.SaleProducts {
		if p.Stock == "eggs" {
			types = append(types, k)
		}
	}
	return types
}

// NotEnoughEggsError is returned when a sale needs more eggs than the batch has on hand
type NotEnoughEggsError struct {
	Flock  string
	OnHand int
}

func (e *NotEnoughEggsError) Error() string {
	n := e.OnHand
	if n < 0 {
		n = 0
	}
	return fmt.Sprintf("You have %d eggs on hand for %s. Record today's collection first.", n, e.Flock)
}
