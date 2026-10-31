package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Sale struct {
	ID              uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	FarmID          uint      `json:"farm_id" gorm:"index;not null"`   // tenant
	CreatedByUserID uint      `json:"created_by_user_id" gorm:"index"` // staff member who logged this record
	FlockID         *uint     `json:"flock_id" gorm:"index"`           // the batch it came from; nil = the whole farm (manure, feed...)
	RefNo           string    `json:"ref_no" gorm:"type:varchar(50);unique;not null"`
	Product         string    `json:"product" gorm:"type:varchar(100);not null"`
	Category        string    `json:"category" gorm:"type:varchar(50);not null"`
	Description     string    `json:"description" gorm:"type:varchar(255);not null"`
	ProductType     string    `json:"product_type" gorm:"type:varchar(16);not null;default:other"`
	Unit            string    `json:"unit" gorm:"type:varchar(12)"`
	Quantity        int       `json:"quantity" gorm:"not null"`
	WeightKg        float64   `json:"weight_kg"` // meat: total weight sold; unit_price is then per kg
	Birds           int       `json:"birds"`     // meat: how many birds were slaughtered for it
	UnitPrice       float64   `json:"unit_price" gorm:"not null"`
	Amount          float64   `json:"amount" gorm:"not null"`
	Date            time.Time `json:"date" gorm:"not null"`
	SaleType        string    `json:"sale_type" gorm:"type:varchar(50);not null"`
	Status          string    `json:"status" gorm:"type:varchar(20);not null;default:'pending'"` // ✅ Added status field
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Flock *Flock `json:"flock,omitempty" gorm:"foreignKey:FlockID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// SaleProduct describes what selling a product does: its category label, the units it's sold in, and
// whether it takes eggs or birds out of stock
type SaleProduct struct {
	Label string
	Units []string
	Stock string // "eggs", "birds", "meat" (birds slaughtered) or "" for none
}

// EggsPerTray is how many eggs a tray holds
const EggsPerTray = 30

// SaleProducts are the things a farm can sell, by product_type
var SaleProducts = map[string]SaleProduct{
	"eggs":          {"Egg Sales", []string{"tray", "piece"}, "eggs"},
	"hatching_eggs": {"Hatching Eggs", []string{"tray", "piece"}, "eggs"},
	"chicks":        {"Chicks", []string{"bird"}, "birds"},
	"live_birds":    {"Live Birds", []string{"bird"}, "birds"},
	"spent_hens":    {"Spent Hens", []string{"bird"}, "birds"},
	"meat":          {"Meat", []string{"kg"}, "meat"},
	"manure":        {"Manure", []string{"sack", "kg"}, ""},
	"feed":          {"Feed", []string{"kg", "bag"}, ""},
	"other":         {"Other", []string{"item"}, ""},
}

// EggsMoved is how many eggs the sale takes out of stock (0 for anything but eggs)
func (s *Sale) EggsMoved() int {
	if SaleProducts[s.ProductType].Stock != "eggs" {
		return 0
	}
	if s.Unit == "tray" {
		return s.Quantity * EggsPerTray
	}
	return s.Quantity
}

// BirdsMoved is how many birds the sale takes out of its batch (0 for anything but birds and meat)
func (s *Sale) BirdsMoved() int {
	switch SaleProducts[s.ProductType].Stock {
	case "birds":
		return s.Quantity
	case "meat":
		return s.Birds
	}
	return 0
}

// GenerateRefNo generates a unique reference number for a farm's sales
func GenerateRefNo(farmID uint) string {
	timestamp := time.Now().Format("20060102150405")                // YYYYMMDDHHMMSS
	randomNum := fmt.Sprintf("%04d", time.Now().Nanosecond()%10000) // 4-digit random number
	return fmt.Sprintf("REF-%d-%s-%s", farmID, timestamp, randomNum)
}

// Hook to generate RefNo before creating a sale
func (s *Sale) BeforeCreate(tx *gorm.DB) (err error) {
	s.RefNo = GenerateRefNo(s.FarmID)
	if s.Status == "" {
		s.Status = "pending..." // Default status
	}
	return
}
