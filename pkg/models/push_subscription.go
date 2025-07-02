package models

type PushSubscription struct {
	ID         uint   `gorm:"primaryKey;autoIncrement"`
	UserID     uint   `json:"user_id"`
	Endpoint   string `json:"endpoint"`
	P256DH     string `json:"p256dh"`
	Auth       string `json:"auth"`
}
