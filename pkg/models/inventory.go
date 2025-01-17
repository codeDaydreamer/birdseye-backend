package models

// InventoryItem represents an item in the inventory
type InventoryItem struct {
    ID          uint    `gorm:"primaryKey" json:"id"`
    ItemName    string  `gorm:"type:varchar(255);not null" json:"item_name"`
    Quantity    int     `gorm:"not null" json:"quantity"`
    ReorderLevel int    `gorm:"not null" json:"reorder_level"`
    CostPerUnit float64 `gorm:"not null" json:"cost_per_unit"`
    UserID      uint    `gorm:"not null" json:"user_id"` // Add the UserID field for the user association
}

// TableName overrides the default table name
func (InventoryItem) TableName() string {
    return "inventory_items"
}
