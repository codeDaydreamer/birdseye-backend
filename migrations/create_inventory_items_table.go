package migrations

import (
	"birdseye-backend/pkg/models"
	"gorm.io/gorm"
	"log"
)

// Migrate runs the migration to create the inventory_items table
func Migrate(db *gorm.DB) error {
	// Check if the table exists, and create it if not
	if !db.Migrator().HasTable(&models.InventoryItem{}) {
		if err := db.AutoMigrate(&models.InventoryItem{}); err != nil {
			log.Fatalf("Error migrating database: %v", err)
			return err
		}
		log.Println("InventoryItems table created successfully.")
	} else {
		log.Println("InventoryItems table already exists.")
	}

	return nil
}
