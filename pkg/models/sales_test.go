package models

import "testing"

func TestSaleStockMoved(t *testing.T) {
	cases := []struct {
		sale        Sale
		eggs, birds int
	}{
		{Sale{ProductType: "eggs", Unit: "tray", Quantity: 5}, 150, 0},
		{Sale{ProductType: "eggs", Unit: "piece", Quantity: 12}, 12, 0},
		{Sale{ProductType: "hatching_eggs", Unit: "tray", Quantity: 2}, 60, 0},
		{Sale{ProductType: "chicks", Unit: "bird", Quantity: 100}, 0, 100},
		{Sale{ProductType: "spent_hens", Unit: "bird", Quantity: 40}, 0, 40},
		{Sale{ProductType: "meat", Unit: "kg", Quantity: 10, Birds: 10, WeightKg: 25}, 0, 10},
		{Sale{ProductType: "manure", Unit: "sack", Quantity: 4}, 0, 0},
	}
	for _, c := range cases {
		if e, b := c.sale.EggsMoved(), c.sale.BirdsMoved(); e != c.eggs || b != c.birds {
			t.Errorf("%s %d %s: eggs %d birds %d, want %d / %d", c.sale.ProductType, c.sale.Quantity,
				c.sale.Unit, e, b, c.eggs, c.birds)
		}
	}
}
