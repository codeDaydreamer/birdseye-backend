package services

import (
	"errors"
	"strings"

	"birdseye-backend/pkg/models"

	"gorm.io/gorm"
)

var (
	ErrContactNotFound = errors.New("contact not found")
	ErrContactName     = errors.New("enter the name")
	ErrContactKind     = errors.New("kind must be customer or supplier")
)

// ListContacts lists the farm's customers or suppliers (kind "" = both), with what customers owe (their
// pending sales) and what has been spent with suppliers
func ListContacts(db *gorm.DB, farmID uint, kind string) ([]models.Contact, error) {
	list := []models.Contact{}
	q := db.Where("farm_id = ?", farmID)
	if kind != "" {
		q = q.Where("kind = ?", kind)
	}
	if err := q.Order("name").Find(&list).Error; err != nil {
		return nil, err
	}
	type row struct {
		ID    uint
		Total float64
	}
	var owing, spent []row
	db.Model(&models.Sale{}).Select("customer_id AS id, COALESCE(SUM(amount), 0) AS total").
		Where("farm_id = ? AND customer_id IS NOT NULL AND status <> ?", farmID, "paid").
		Group("customer_id").Scan(&owing)
	db.Model(&models.Expense{}).Select("supplier_id AS id, COALESCE(SUM(amount), 0) AS total").
		Where("farm_id = ? AND supplier_id IS NOT NULL", farmID).Group("supplier_id").Scan(&spent)
	o, sp := map[uint]float64{}, map[uint]float64{}
	for _, r := range owing {
		o[r.ID] = r.Total
	}
	for _, r := range spent {
		sp[r.ID] = r.Total
	}
	for i := range list {
		list[i].Owing, list[i].Spent = o[list[i].ID], sp[list[i].ID]
	}
	return list, nil
}

// SaveContact creates or updates one of the farm's contacts
func SaveContact(db *gorm.DB, farmID uint, c *models.Contact) error {
	c.Name = strings.TrimSpace(c.Name)
	c.Phone = strings.TrimSpace(c.Phone)
	switch {
	case c.Name == "":
		return ErrContactName
	case c.Kind != models.ContactCustomer && c.Kind != models.ContactSupplier:
		return ErrContactKind
	}
	if c.Phone != "" {
		phone, err := NormalizePhone(c.Phone)
		if err != nil {
			return err
		}
		c.Phone = phone
	}
	c.FarmID = farmID
	if c.ID == 0 {
		return db.Create(c).Error
	}
	res := db.Model(&models.Contact{}).Where("id = ? AND farm_id = ?", c.ID, farmID).
		Select("kind", "name", "phone", "location", "notes").Updates(c)
	if res.Error == nil && res.RowsAffected == 0 {
		var n int64
		db.Model(&models.Contact{}).Where("id = ? AND farm_id = ?", c.ID, farmID).Count(&n)
		if n == 0 {
			return ErrContactNotFound
		}
	}
	return res.Error
}

// DeleteContact removes a contact; their sales and expenses stay, just unlinked
func DeleteContact(db *gorm.DB, farmID, id uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ? AND farm_id = ?", id, farmID).Delete(&models.Contact{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrContactNotFound
		}
		if err := tx.Model(&models.Sale{}).Where("customer_id = ? AND farm_id = ?", id, farmID).
			Update("customer_id", nil).Error; err != nil {
			return err
		}
		return tx.Model(&models.Expense{}).Where("supplier_id = ? AND farm_id = ?", id, farmID).
			Update("supplier_id", nil).Error
	})
}

// Statement is everything with one contact: a customer's sales or a supplier's expenses, with totals
type Statement struct {
	Contact  models.Contact   `json:"contact"`
	Sales    []models.Sale    `json:"sales,omitempty"`
	Expenses []models.Expense `json:"expenses,omitempty"`
	Billed   float64          `json:"billed"` // customers: all sales
	Paid     float64          `json:"paid"`
	Owing    float64          `json:"owing"`
	Spent    float64          `json:"spent"` // suppliers: all expenses
}

func ContactStatement(db *gorm.DB, farmID, id uint) (*Statement, error) {
	var c models.Contact
	if err := db.Where("id = ? AND farm_id = ?", id, farmID).First(&c).Error; err != nil {
		return nil, ErrContactNotFound
	}
	st := &Statement{Contact: c}
	if c.Kind == models.ContactCustomer {
		st.Sales = []models.Sale{}
		db.Where("farm_id = ? AND customer_id = ?", farmID, id).Order("date DESC, id DESC").Find(&st.Sales)
		for _, s := range st.Sales {
			st.Billed += s.Amount
			if s.Status == "paid" {
				st.Paid += s.Amount
			} else {
				st.Owing += s.Amount
			}
		}
	} else {
		st.Expenses = []models.Expense{}
		db.Where("farm_id = ? AND supplier_id = ?", farmID, id).Order("date DESC, id DESC").Find(&st.Expenses)
		for _, e := range st.Expenses {
			st.Spent += e.Amount
		}
	}
	st.Contact.Owing, st.Contact.Spent = st.Owing, st.Spent
	return st, nil
}

// ContactOfKind treats a missing/0 id as none (setting it nil) and otherwise checks the farm has a contact of
// that kind with the id
func ContactOfKind(db *gorm.DB, id **uint, farmID uint, kind string) bool {
	if *id == nil || **id == 0 {
		*id = nil
		return true
	}
	var n int64
	db.Model(&models.Contact{}).Where("id = ? AND farm_id = ? AND kind = ?", **id, farmID, kind).Count(&n)
	return n > 0
}
