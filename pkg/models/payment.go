package models

import (
	
	"time"

	

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Payment struct {
	ID                 string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID             uint       `gorm:"index;not null" json:"user_id"`
	Amount             float64    `gorm:"not null" json:"amount"`
	PhoneNumber        string     `gorm:"type:varchar(20);not null" json:"phone_number"`
	Status             string     `gorm:"default:'initiated'" json:"status"`
	Reference          string     `gorm:"type:varchar(255);not null" json:"reference"`
	PaymentID          *string    `gorm:"type:varchar(191);uniqueIndex" json:"payment_id"`
	MpesaReference     string     `gorm:"type:varchar(255)" json:"mpesa_reference"`
	MerchantRequestID  string     `gorm:"type:varchar(255)" json:"merchant_request_id"`
	CheckoutRequestID  string     `gorm:"type:varchar(255)" json:"checkout_request_id"`
	ResultDescription  string     `gorm:"type:varchar(255)" json:"result_description"` 
	CallbackReceivedAt *time.Time `json:"callback_received_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// BeforeCreate hook to generate UUID manually
func (p *Payment) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return
}
