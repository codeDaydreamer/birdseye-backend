package models



// FlockFinancialData stores monthly financial data for a flock
type FlocksFinancialData struct {
    ID         uint    `json:"id" gorm:"primaryKey;autoIncrement"`
    FlockID    uint    `json:"flock_id" gorm:"index;not null"`
    UserID     uint    `json:"user_id" gorm:"index;not null"`
    Month      int     `json:"month" gorm:"not null"`
    Year       int     `json:"year" gorm:"not null"`
    Revenue    float64 `json:"revenue" gorm:"not null"`
    EggSales   float64 `json:"egg_sales" gorm:"not null"`
    Expenses   float64 `json:"expenses" gorm:"not null"`
    NetRevenue float64 `json:"net_revenue" gorm:"not null"`
}

