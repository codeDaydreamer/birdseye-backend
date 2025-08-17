package models

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CustomTime is a wrapper around time.Time to control JSON output and DB storage
type CustomTime struct {
	time.Time
}

// MarshalJSON ensures UTC and RFC3339 format in JSON output
func (ct CustomTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + ct.UTC().Format(time.RFC3339) + `"`), nil
}

// UnmarshalJSON parses a JSON time string in RFC3339 format
func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	parsed, err := time.Parse(`"`+time.RFC3339+`"`, string(b))
	if err != nil {
		return err
	}
	ct.Time = parsed.UTC()
	return nil
}

// Value makes CustomTime implement the driver.Valuer interface (for DB writes)
func (ct CustomTime) Value() (driver.Value, error) {
	if ct.IsZero() {
		return nil, nil
	}
	return ct.UTC(), nil
}

// Scan makes CustomTime implement the sql.Scanner interface (for DB reads)
func (ct *CustomTime) Scan(value interface{}) error {
	t, ok := value.(time.Time)
	if !ok {
		return fmt.Errorf("cannot convert %v to time.Time", value)
	}
	ct.Time = t.UTC()
	return nil
}

// Payment model
type Payment struct {
	ID                 string      `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID             uint        `gorm:"index;not null" json:"user_id"`
	Amount             float64     `gorm:"not null" json:"amount"`
	PhoneNumber        string      `gorm:"type:varchar(20);not null" json:"phone_number"`
	Status             string      `gorm:"default:'initiated'" json:"status"`

	Reference          string      `gorm:"type:varchar(255);uniqueIndex" json:"reference"`
	TxRef *string `gorm:"type:varchar(255);uniqueIndex" json:"tx_ref"`
	PaymentID          *string     `gorm:"type:varchar(191);uniqueIndex" json:"payment_id"`

	// Daraja (M-Pesa) fields
	MpesaReference     string      `gorm:"type:varchar(255)" json:"mpesa_reference"`
	MerchantRequestID  string      `gorm:"type:varchar(255)" json:"merchant_request_id"`
	CheckoutRequestID  string      `gorm:"type:varchar(255)" json:"checkout_request_id"`
	ResultDescription  string      `gorm:"type:varchar(255)" json:"result_description"`
	CallbackReceivedAt *CustomTime `json:"callback_received_at"`

	Gateway            string      `gorm:"type:varchar(50)" json:"gateway"`
	PaidAt             CustomTime  `json:"paid_at"`
	CreatedAt          CustomTime  `json:"created_at"`
	UpdatedAt          CustomTime  `json:"updated_at"`
}

// BeforeCreate hook to generate UUID and set UTC timestamps
func (p *Payment) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	p.CreatedAt = CustomTime{now}
	p.UpdatedAt = CustomTime{now}

	if p.PaidAt.IsZero() {
		p.PaidAt = CustomTime{now}
	}

	return
}

// BeforeUpdate hook to ensure UpdatedAt uses UTC
func (p *Payment) BeforeUpdate(tx *gorm.DB) (err error) {
	p.UpdatedAt = CustomTime{time.Now().UTC()}
	return
}
