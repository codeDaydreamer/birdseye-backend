package models

// InventoryItem represents an item in the 
// InventoryItem represents an item in the inventory
type InventoryItem struct {
    ID          uint    `gorm:"primaryKey;autoIncrement" json:"id"`
    ItemName    string  `gorm:"type:varchar(255);not null" json:"item_name"`
    Quantity    int     `gorm:"not null" json:"quantity"`
    ReorderLevel int    `gorm:"not null" json:"reorder_level"`
    CostPerUnit float64 `gorm:"not null" json:"cost_per_unit"`
    UserID      uint    `gorm:"not null" json:"user_id"`
    FlockID     uint    `gorm:"not null;index" json:"flock_id"` // Foreign key reference to Flock

    // Relationship
    Flock       Flock   `gorm:"foreignKey:FlockID;constraint:OnDelete:CASCADE;" json:"flock"` // Define relationship with Flock
}

// TableName overrides the default table name
func (InventoryItem) TableName() string {
    return "inventory_items"
}
