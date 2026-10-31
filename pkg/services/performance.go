package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"birdseye-backend/pkg/broadcast"
	"birdseye-backend/pkg/models"

	"gorm.io/gorm"
)

var (
	ErrFeedType     = errors.New("feed type must be starter, grower, finisher, layers_mash or other")
	ErrFeedKg       = errors.New("enter how many kg of feed")
	ErrFeedNotFound = errors.New("feed record not found")
	ErrWeighSample  = errors.New("weigh at least 1 bird")
	ErrWeighWeight  = errors.New("enter the average weight in grams")
	ErrWeighFound   = errors.New("weigh-in not found")
)

// FeedLogs lists a batch's feed, newest first
func FeedLogs(db *gorm.DB, farmID, flockID uint) ([]models.FeedLog, error) {
	list := []models.FeedLog{}
	err := db.Where("farm_id = ? AND flock_id = ?", farmID, flockID).Order("date DESC, id DESC").Find(&list).Error
	return list, err
}

// AddFeed records feed given to a batch; a cost also files a Feed expense for the batch
func AddFeed(db *gorm.DB, farmID, userID, flockID uint, f *models.FeedLog) error {
	if !models.FeedTypes[f.FeedType] {
		return ErrFeedType
	}
	if f.Kg <= 0 {
		return ErrFeedKg
	}
	if f.Cost < 0 {
		return ErrSalePrice
	}
	if f.Date.IsZero() {
		f.Date = time.Now()
	}
	f.ID, f.FarmID, f.FlockID, f.ExpenseID = 0, farmID, flockID, nil
	f.Notes = strings.TrimSpace(f.Notes)
	return db.Transaction(func(tx *gorm.DB) error {
		var flock models.Flock
		if err := tx.Select("id", "name").Where("id = ? AND farm_id = ?", flockID, farmID).First(&flock).Error; err != nil {
			return ErrFlockNotFound
		}
		if f.Cost > 0 {
			expense := models.Expense{FarmID: farmID, FlockID: &flockID, CreatedByUserID: userID, Date: f.Date,
				Description: fmt.Sprintf("%.0f kg %s for %s", f.Kg, strings.ReplaceAll(f.FeedType, "_", " "), flock.Name),
				Amount:      f.Cost, Category: "Feed"}
			if err := tx.Create(&expense).Error; err != nil {
				return err
			}
			f.ExpenseID = &expense.ID
			broadcast.SendExpenseUpdate(farmID, "expense_added", expense)
		}
		return tx.Create(f).Error
	})
}

// DeleteFeed removes a feed record and its expense
func DeleteFeed(db *gorm.DB, farmID, id uint) error {
	var f models.FeedLog
	if err := db.Where("id = ? AND farm_id = ?", id, farmID).First(&f).Error; err != nil {
		return ErrFeedNotFound
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if f.ExpenseID != nil {
			if err := tx.Where("id = ? AND farm_id = ?", *f.ExpenseID, farmID).Delete(&models.Expense{}).Error; err != nil {
				return err
			}
			broadcast.SendExpenseUpdate(farmID, "expense_deleted", *f.ExpenseID)
		}
		return tx.Delete(&f).Error
	})
}

// WeighIns lists a batch's weigh-ins, newest first
func WeighIns(db *gorm.DB, farmID, flockID uint) ([]models.WeighIn, error) {
	list := []models.WeighIn{}
	err := db.Where("farm_id = ? AND flock_id = ?", farmID, flockID).Order("date DESC, id DESC").Find(&list).Error
	return list, err
}

// AddWeighIn records the average weight of a sample of the batch's birds
func AddWeighIn(db *gorm.DB, farmID, flockID uint, w *models.WeighIn) error {
	if w.SampleSize < 1 {
		return ErrWeighSample
	}
	if w.AvgWeightG <= 0 {
		return ErrWeighWeight
	}
	if w.Date.IsZero() {
		w.Date = time.Now()
	}
	var n int64
	db.Model(&models.Flock{}).Where("id = ? AND farm_id = ?", flockID, farmID).Count(&n)
	if n == 0 {
		return ErrFlockNotFound
	}
	w.ID, w.FarmID, w.FlockID = 0, farmID, flockID
	return db.Create(w).Error
}

func DeleteWeighIn(db *gorm.DB, farmID, id uint) error {
	res := db.Where("id = ? AND farm_id = ?", id, farmID).Delete(&models.WeighIn{})
	if res.Error == nil && res.RowsAffected == 0 {
		return ErrWeighFound
	}
	return res.Error
}

// Performance is how a batch is doing: feed against weight (broilers), laying rate (layers) and money
type Performance struct {
	FeedKg        float64  `json:"feed_kg"`
	AvgWeightG    float64  `json:"avg_weight_g"`   // latest weigh-in
	WeighedAt     *string  `json:"weighed_at"`     // date of that weigh-in
	LiveWeightKg  float64  `json:"live_weight_kg"` // avg weight x birds now
	FCR           *float64 `json:"fcr"`            // kg feed per kg live weight (nil until weighed and fed)
	Costs         float64  `json:"costs"`          // the batch's expenses
	Revenue       float64  `json:"revenue"`        // the batch's sales
	CostPerKg     *float64 `json:"cost_per_kg"`    // costs per kg live weight
	EggsLast7     int      `json:"eggs_last_7"`
	LayingRate7   *float64 `json:"laying_rate_7"` // hen-day %: eggs in the last 7 days / (birds x 7)
	EggsPerBird   float64  `json:"eggs_per_bird"` // all eggs collected / birds placed
	FeedPerBirdKg float64  `json:"feed_per_bird_kg"`
}

// BatchPerformance works out a batch's figures from its feed, weigh-ins, eggs, sales and expenses
func BatchPerformance(db *gorm.DB, farmID, flockID uint) (*Performance, error) {
	var flock models.Flock
	if err := db.Where("id = ? AND farm_id = ?", flockID, farmID).First(&flock).Error; err != nil {
		return nil, ErrFlockNotFound
	}
	p := &Performance{}
	scalar := func(q *gorm.DB, expr string) float64 {
		var v float64
		q.Select("COALESCE(" + expr + ", 0)").Scan(&v)
		return v
	}
	p.FeedKg = scalar(db.Model(&models.FeedLog{}).Where("flock_id = ?", flockID), "SUM(kg)")
	p.Costs = scalar(db.Model(&models.Expense{}).Where("flock_id = ?", flockID), "SUM(amount)")
	p.Revenue = scalar(db.Model(&models.Sale{}).Where("flock_id = ?", flockID), "SUM(amount)")

	var last models.WeighIn
	if err := db.Where("flock_id = ?", flockID).Order("date DESC, id DESC").First(&last).Error; err == nil {
		p.AvgWeightG = last.AvgWeightG
		day := last.Date.Format("2006-01-02")
		p.WeighedAt = &day
		p.LiveWeightKg = last.AvgWeightG / 1000 * float64(flock.BirdCount)
		if p.FeedKg > 0 && p.LiveWeightKg > 0 {
			fcr := p.FeedKg / p.LiveWeightKg
			p.FCR = &fcr
		}
		if p.Costs > 0 && p.LiveWeightKg > 0 {
			c := p.Costs / p.LiveWeightKg
			p.CostPerKg = &c
		}
	}

	since := time.Now().AddDate(0, 0, -6).Format("2006-01-02")
	p.EggsLast7 = int(scalar(db.Model(&models.EggProduction{}).
		Where("flock_id = ? AND date_produced >= ?", flockID, since), "SUM(eggs_collected)"))
	if flock.BirdCount > 0 && p.EggsLast7 > 0 {
		r := float64(p.EggsLast7) / float64(flock.BirdCount*7) * 100
		p.LayingRate7 = &r
	}
	if flock.InitialBirdCount > 0 {
		all := scalar(db.Model(&models.EggProduction{}).Where("flock_id = ?", flockID), "SUM(eggs_collected)")
		p.EggsPerBird = all / float64(flock.InitialBirdCount)
		p.FeedPerBirdKg = p.FeedKg / float64(flock.InitialBirdCount)
	}
	return p, nil
}
