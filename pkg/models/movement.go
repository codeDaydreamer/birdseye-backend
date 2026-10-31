package models

import "time"

// Kinds of bird movement. Delta is positive for birds coming in and negative for birds going out;
// an adjust can be either (a recount).
const (
	MovePlaced      = "placed"
	MoveDied        = "died"
	MoveCulled      = "culled"
	MoveSold        = "sold"
	MoveTransferIn  = "transfer_in"
	MoveTransferOut = "transfer_out"
	MoveAdjust      = "adjust"
)

// DeathCauses are the reasons a farmer can give for deaths
var DeathCauses = map[string]bool{
	"disease": true, "predator": true, "heat_cold": true, "smothering": true, "injury": true, "unknown": true,
}

// BirdMovement is one change to a batch's bird count. A batch's counts are always the sum of these.
type BirdMovement struct {
	ID              uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	FarmID          uint      `json:"farm_id" gorm:"index;not null"`
	FlockID         uint      `json:"flock_id" gorm:"index;not null"`
	Type            string    `json:"type" gorm:"type:varchar(16);not null"`
	Delta           int       `json:"delta" gorm:"not null"`
	Date            time.Time `json:"date" gorm:"type:date;not null"`
	Cause           string    `json:"cause" gorm:"type:varchar(20)"`
	Notes           string    `json:"notes" gorm:"type:varchar(255)"`
	SaleID          *uint     `json:"sale_id" gorm:"index"` // set for "sold": the sale it came from
	RelatedFlockID  *uint     `json:"related_flock_id"`     // the other batch of a transfer
	PairID          *uint     `json:"pair_id" gorm:"index"` // a transfer's other half
	CreatedByUserID uint      `json:"created_by_user_id" gorm:"index"`
	CreatedAt       time.Time `json:"created_at"`
}

// FlockTotals are a batch's counts worked out from its movements
type FlockTotals struct {
	Initial   int // placed + moved in
	Birds     int
	Deaths    int
	Culled    int
	Sold      int
	Mortality float64 // deaths as % of Initial
}

// TotalsFromMovements adds up a batch's movements. Only deaths count towards mortality: birds sold,
// culled or moved out are not losses.
func TotalsFromMovements(moves []BirdMovement) FlockTotals {
	var t FlockTotals
	for _, m := range moves {
		t.Birds += m.Delta
		switch m.Type {
		case MovePlaced, MoveTransferIn:
			t.Initial += m.Delta
		case MoveDied:
			t.Deaths -= m.Delta
		case MoveCulled:
			t.Culled -= m.Delta
		case MoveSold:
			t.Sold -= m.Delta
		}
	}
	if t.Initial > 0 {
		t.Mortality = float64(t.Deaths) / float64(t.Initial) * 100
	}
	return t
}
