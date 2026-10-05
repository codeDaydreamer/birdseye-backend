package models

import (
	"fmt"
	"log"

	"birdseye-backend/pkg/db"

	"gorm.io/gorm"
)

// farmScopedModels are the tables that were scoped by user_id and now also carry farm_id
var farmScopedModels = []interface{}{
	&Flock{}, &Budget{}, &EggProduction{}, &EggAdjustment{}, &Expense{},
	&FlocksFinancialData{}, &InventoryItem{}, &Sale{}, &Vaccination{},
	&Notification{}, &Payment{}, &PushSubscription{}, &Report{},
	&BillingInfo{}, &Subscription{},
}

// auditedModels additionally get created_by_user_id backfilled from user_id
var auditedModels = []interface{}{&Sale{}, &Expense{}, &Vaccination{}}

func tableName(m interface{}) (string, error) {
	stmt := &gorm.Statement{DB: db.DB}
	if err := stmt.Parse(m); err != nil {
		return "", err
	}
	return stmt.Schema.Table, nil
}

// BackfillFarms gives every User without a farm its own Farm + owner FarmMember, then
// populates farm_id on all user-scoped tables. Idempotent: safe to run on every boot.
func BackfillFarms() error {
	var users []User
	if err := db.DB.
		Where("id NOT IN (?)", db.DB.Model(&FarmMember{}).Select("user_id").Where("role = ?", FarmRoleOwner)).
		Find(&users).Error; err != nil {
		return err
	}

	for _, user := range users {
		err := db.DB.Transaction(func(tx *gorm.DB) error {
			farm := Farm{
				Name:          fmt.Sprintf("%s's Farm", user.Username),
				OwnerUserID:   user.ID,
				PaymentStatus: user.PaymentStatus,
			}
			if err := tx.Omit("Subscription", "BillingInfo").Create(&farm).Error; err != nil {
				return err
			}
			return tx.Create(&FarmMember{FarmID: farm.ID, UserID: user.ID, Role: FarmRoleOwner}).Error
		})
		if err != nil {
			log.Printf("Failed to create farm for user %d: %v", user.ID, err)
			continue
		}
		log.Printf("Created farm for user %d", user.ID)
	}

	for _, m := range farmScopedModels {
		table, err := tableName(m)
		if err != nil {
			return err
		}
		res := db.DB.Exec(fmt.Sprintf(
			"UPDATE %s t JOIN farm_members fm ON fm.user_id = t.user_id AND fm.role = ? "+
				"SET t.farm_id = fm.farm_id WHERE t.farm_id IS NULL OR t.farm_id = 0",
			table), FarmRoleOwner)
		if res.Error != nil {
			return fmt.Errorf("backfilling farm_id on %s: %w", table, res.Error)
		}
		if res.RowsAffected > 0 {
			log.Printf("Backfilled farm_id on %d %s rows", res.RowsAffected, table)
		}
	}

	for _, m := range auditedModels {
		table, err := tableName(m)
		if err != nil {
			return err
		}
		if err := db.DB.Exec(fmt.Sprintf(
			"UPDATE %s SET created_by_user_id = user_id WHERE created_by_user_id IS NULL OR created_by_user_id = 0",
			table)).Error; err != nil {
			return fmt.Errorf("backfilling created_by_user_id on %s: %w", table, err)
		}
	}

	// Never keep raw card data around (PCI): drop the legacy plaintext columns
	for _, col := range []string{"card_number", "cvv", "expiry_date"} {
		if db.DB.Migrator().HasColumn(&BillingInfo{}, col) {
			if err := db.DB.Migrator().DropColumn(&BillingInfo{}, col); err != nil {
				return fmt.Errorf("dropping billing_infos.%s: %w", col, err)
			}
			log.Printf("Dropped billing_infos.%s", col)
		}
	}

	return nil
}
