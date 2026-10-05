package models

type PushSubscription struct {
	ID         uint   `gorm:"primaryKey;autoIncrement"`
	UserID     uint   `json:"user_id"`
	FarmID uint `json:"farm_id" gorm:"index"` // tenant; backfilled from user_id
	Endpoint   string `json:"endpoint"`
	P256DH     string `json:"p256dh"`
	Auth       string `json:"auth"`
}
