package models

import "time"

// Contact kinds
const (
	ContactCustomer = "customer"
	ContactSupplier = "supplier"
)

// Contact is someone the farm sells to (customer) or buys from (supplier)
type Contact struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	FarmID    uint      `json:"farm_id" gorm:"index;not null"`
	Kind      string    `json:"kind" gorm:"type:varchar(16);not null;index"`
	Name      string    `json:"name" gorm:"type:varchar(120);not null"`
	Phone     string    `json:"phone" gorm:"type:varchar(20)"`
	Location  string    `json:"location" gorm:"type:varchar(120)"`
	Notes     string    `json:"notes" gorm:"type:varchar(255)"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Worked out for lists: customers' unpaid sales, suppliers' total spend
	Owing float64 `json:"owing" gorm:"-"`
	Spent float64 `json:"spent" gorm:"-"`
}
