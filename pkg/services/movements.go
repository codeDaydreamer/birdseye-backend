package services

import (
	"errors"
	"fmt"
	"time"

	"birdseye-backend/pkg/broadcast"
	"birdseye-backend/pkg/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrFlockNotFound    = errors.New("flock not found")
	ErrMovementNotFound = errors.New("movement not found")
	ErrMovementLocked   = errors.New("this entry comes from a sale or the batch's placement; change that instead")
	ErrInvalidMovement  = errors.New("type must be died, culled, adjust or transfer")
	ErrInvalidQuantity  = errors.New("enter how many birds")
	ErrInvalidCause     = errors.New("cause must be disease, predator, heat_cold, smothering, injury or unknown")
	ErrSameFlock        = errors.New("choose a different batch to move the birds to")
	ErrFlockClosed      = errors.New("this batch is closed")
)

// NotEnoughBirdsError is returned when a movement would take a batch below zero birds
type NotEnoughBirdsError struct {
	Flock string
	Have  int
}

func (e *NotEnoughBirdsError) Error() string {
	return fmt.Sprintf("%s only has %d birds", e.Flock, e.Have)
}

// lockFlock loads the farm's batch and locks its row until the transaction ends, so two movements at once
// can't both pass the "enough birds" check
func lockFlock(tx *gorm.DB, flockID, farmID uint) (*models.Flock, error) {
	var flock models.Flock
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND farm_id = ?", flockID, farmID).First(&flock).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFlockNotFound
	}
	return &flock, err
}

// RecomputeFlockTotals rewrites the batch's stored counts from its movement log
func RecomputeFlockTotals(tx *gorm.DB, flockID uint) (models.FlockTotals, error) {
	var moves []models.BirdMovement
	if err := tx.Where("flock_id = ?", flockID).Find(&moves).Error; err != nil {
		return models.FlockTotals{}, err
	}
	t := models.TotalsFromMovements(moves)
	err := tx.Model(&models.Flock{}).Where("id = ?", flockID).Updates(map[string]interface{}{
		"initial_bird_count": t.Initial,
		"bird_count":         t.Birds,
		"deaths":             t.Deaths,
		"culled":             t.Culled,
		"sold":               t.Sold,
		"mortality_rate":     t.Mortality,
	}).Error
	return t, err
}

// ApplyMovement records one movement on a batch inside tx and updates its counts. It refuses to take the
// batch below zero and checks the plan's bird limit when birds come in.
func ApplyMovement(tx *gorm.DB, m *models.BirdMovement) error {
	flock, err := lockFlock(tx, m.FlockID, m.FarmID)
	if err != nil {
		return err
	}
	if flock.BirdCount+m.Delta < 0 {
		return &NotEnoughBirdsError{Flock: flock.Name, Have: flock.BirdCount}
	}
	if m.Delta > 0 && m.Type != models.MovePlaced {
		if err := CheckBirdLimit(m.FarmID, flock.BirdCount+m.Delta, flock.ID); err != nil {
			return err
		}
	}
	if m.Date.IsZero() {
		m.Date = time.Now()
	}
	if err := tx.Create(m).Error; err != nil {
		return err
	}
	_, err = RecomputeFlockTotals(tx, flock.ID)
	return err
}

// TransferBirds moves birds from one of the farm's batches to another (e.g. chicks to the laying house)
func TransferBirds(tx *gorm.DB, farmID, userID, fromID, toID uint, qty int, date time.Time, notes string) error {
	if fromID == toID {
		return ErrSameFlock
	}
	to, err := lockFlock(tx, toID, farmID)
	if err != nil {
		return err
	}
	if to.Status == models.FlockClosed {
		return ErrFlockClosed
	}
	out := models.BirdMovement{FarmID: farmID, FlockID: fromID, Type: models.MoveTransferOut, Delta: -qty,
		Date: date, Notes: notes, RelatedFlockID: &toID, CreatedByUserID: userID}
	if err := ApplyMovement(tx, &out); err != nil {
		return err
	}
	in := models.BirdMovement{FarmID: farmID, FlockID: toID, Type: models.MoveTransferIn, Delta: qty,
		Date: date, Notes: notes, RelatedFlockID: &fromID, PairID: &out.ID, CreatedByUserID: userID}
	if err := ApplyMovement(tx, &in); err != nil {
		return err
	}
	return tx.Model(&out).Update("pair_id", in.ID).Error
}

// DeleteMovement undoes a death, cull, recount or transfer (both halves). Placements and sales can't be
// undone here: they follow the batch and the sale.
func DeleteMovement(tx *gorm.DB, farmID, movementID uint) (affected []uint, err error) {
	var m models.BirdMovement
	if err := tx.Where("id = ? AND farm_id = ?", movementID, farmID).First(&m).Error; err != nil {
		return nil, ErrMovementNotFound
	}
	if m.Type == models.MovePlaced || m.Type == models.MoveSold {
		return nil, ErrMovementLocked
	}
	rows := []models.BirdMovement{m}
	if m.PairID != nil {
		var pair models.BirdMovement
		if err := tx.First(&pair, *m.PairID).Error; err == nil {
			rows = append(rows, pair)
		}
	}
	for _, r := range rows {
		flock, err := lockFlock(tx, r.FlockID, farmID)
		if err != nil {
			return nil, err
		}
		// Undoing an addition can't leave the batch with fewer than zero birds (e.g. they were sold since)
		if flock.BirdCount-r.Delta < 0 {
			return nil, &NotEnoughBirdsError{Flock: flock.Name, Have: flock.BirdCount}
		}
		if err := tx.Delete(&models.BirdMovement{}, r.ID).Error; err != nil {
			return nil, err
		}
		if _, err := RecomputeFlockTotals(tx, r.FlockID); err != nil {
			return nil, err
		}
		affected = append(affected, r.FlockID)
	}
	return affected, nil
}

// BroadcastFlocks sends each batch's fresh record to the farm's open apps (after the transaction commits)
func BroadcastFlocks(db *gorm.DB, farmID uint, flockIDs ...uint) {
	for _, id := range flockIDs {
		var flock models.Flock
		if err := db.Where("id = ? AND farm_id = ?", id, farmID).First(&flock).Error; err == nil {
			broadcast.SendFlockUpdate(farmID, "flock_updated", flock)
		}
	}
}
