package services

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"

	"gorm.io/gorm"
)

// PlanFeature is a part of the app a plan can include or leave out
type PlanFeature struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// PlanFeatureCatalog lists the features the admin can switch on or off per plan. Each key guards a
// route group (see middlewares.RequireFeature); flocks, eggs, sales and expenses are always included.
var PlanFeatureCatalog = []PlanFeature{
	{"vaccinations", "Vaccinations", "Vaccination schedules and reminders"},
	{"inventory", "Inventory", "Feed, medicine and equipment stock"},
	{"budgets", "Budgets", "Monthly budgets per flock"},
	{"finances", "Flock finances", "Per-flock revenue, expense and profit tracking"},
	{"reports", "Reports", "Downloadable PDF reports"},
}

func isKnownFeature(key string) bool {
	for _, f := range PlanFeatureCatalog {
		if f.Key == key {
			return true
		}
	}
	return false
}

var (
	ErrPlanNotFound     = errors.New("plan not found")
	ErrPeriodNotOffered = errors.New("this plan isn't offered for that billing period")
	ErrPromoInvalid     = errors.New("promo code is not valid")
)

// PlanLimitError says which plan limit an action would exceed
type PlanLimitError struct {
	Limit string
	Max   int
}

func (e *PlanLimitError) Error() string {
	return fmt.Sprintf("your plan allows at most %d %s; upgrade your plan to add more", e.Max, e.Limit)
}

// GetFarmPlan returns the farm's current plan, or nil when it has none (then nothing is limited)
func GetFarmPlan(farmID uint) (*models.Plan, error) {
	var farm models.Farm
	if err := db.DB.Omit("Subscription", "BillingInfo").Select("id", "plan_id").First(&farm, farmID).Error; err != nil {
		return nil, err
	}
	if farm.PlanID == nil {
		return nil, nil
	}
	var plan models.Plan
	if err := db.DB.First(&plan, *farm.PlanID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &plan, nil
}

// FarmHasFeature reports whether the farm's plan includes a feature (farms without a plan get all)
func FarmHasFeature(farmID uint, key string) (bool, error) {
	plan, err := GetFarmPlan(farmID)
	if err != nil || plan == nil {
		return plan == nil, err
	}
	return plan.Features[key], nil
}

// CheckFlockLimit fails if the farm can't add another flock on its plan
func CheckFlockLimit(farmID uint) error {
	plan, err := GetFarmPlan(farmID)
	if err != nil || plan == nil || plan.MaxFlocks <= 0 {
		return err
	}
	var count int64
	db.DB.Model(&models.Flock{}).Where("farm_id = ?", farmID).Count(&count)
	if int(count) >= plan.MaxFlocks {
		return &PlanLimitError{Limit: "flocks", Max: plan.MaxFlocks}
	}
	return nil
}

// CheckBirdLimit fails if the farm's birds would exceed its plan once a flock has newBirdCount
// birds (excludeFlockID is that flock's current row when updating, 0 when creating)
func CheckBirdLimit(farmID uint, newBirdCount int, excludeFlockID uint) error {
	plan, err := GetFarmPlan(farmID)
	if err != nil || plan == nil || plan.MaxBirds <= 0 {
		return err
	}
	var others int64
	db.DB.Model(&models.Flock{}).Where("farm_id = ? AND id <> ?", farmID, excludeFlockID).
		Select("COALESCE(SUM(bird_count), 0)").Scan(&others)
	if int(others)+newBirdCount > plan.MaxBirds {
		return &PlanLimitError{Limit: "birds", Max: plan.MaxBirds}
	}
	return nil
}

// CheckStaffLimit fails if the farm can't take on another member (pending invites count too)
func CheckStaffLimit(farmID uint, countPendingInvites bool) error {
	plan, err := GetFarmPlan(farmID)
	if err != nil || plan == nil || plan.MaxStaff <= 0 {
		return err
	}
	var members, invites int64
	db.DB.Model(&models.FarmMember{}).Where("farm_id = ? AND status = ?", farmID, "active").Count(&members)
	if countPendingInvites {
		db.DB.Model(&models.FarmInvite{}).
			Where("farm_id = ? AND status = ? AND expires_at > ?", farmID, "pending", time.Now()).Count(&invites)
	}
	if int(members+invites) >= plan.MaxStaff {
		return &PlanLimitError{Limit: "staff members", Max: plan.MaxStaff}
	}
	return nil
}

// Quote is the price of a plan for a billing period after any promo code
type Quote struct {
	PlanID         uint    `json:"plan_id"`
	PlanName       string  `json:"plan_name"`
	BillingPeriod  string  `json:"billing_period"`
	ListPrice      float64 `json:"list_price"`
	PromoCode      string  `json:"promo_code,omitempty"`
	PromoCodeID    *uint   `json:"-"`
	DiscountAmount float64 `json:"discount_amount"`
	Amount         float64 `json:"amount"`
	Currency       string  `json:"currency"`
}

// findUsablePromo loads a promo code that can be applied to the plan right now
func findUsablePromo(code string, planID uint) (*models.PromoCode, error) {
	var promo models.PromoCode
	if err := db.DB.Where("code = ?", strings.ToUpper(strings.TrimSpace(code))).First(&promo).Error; err != nil {
		return nil, ErrPromoInvalid
	}
	now := time.Now()
	switch {
	case !promo.IsActive,
		promo.ValidFrom != nil && now.Before(*promo.ValidFrom),
		promo.ValidUntil != nil && now.After(*promo.ValidUntil),
		promo.MaxRedemptions > 0 && promo.RedemptionCount >= promo.MaxRedemptions,
		promo.PlanID != nil && *promo.PlanID != planID:
		return nil, ErrPromoInvalid
	}
	return &promo, nil
}

// QuotePlan prices a plan for a billing period, applying a promo code if one is given
func QuotePlan(planID uint, period, promoCode string) (*Quote, error) {
	if period != models.BillingMonthly && period != models.BillingYearly {
		return nil, ErrPeriodNotOffered
	}
	var plan models.Plan
	if err := db.DB.Where("id = ? AND is_active = ?", planID, true).First(&plan).Error; err != nil {
		return nil, ErrPlanNotFound
	}
	price := plan.PriceFor(period)
	if price <= 0 {
		return nil, ErrPeriodNotOffered
	}

	settings, err := models.GetPlatformSettings()
	if err != nil {
		return nil, err
	}

	quote := &Quote{
		PlanID:        plan.ID,
		PlanName:      plan.Name,
		BillingPeriod: period,
		ListPrice:     price,
		Amount:        price,
		Currency:      settings.Currency,
	}

	if strings.TrimSpace(promoCode) != "" {
		promo, err := findUsablePromo(promoCode, plan.ID)
		if err != nil {
			return nil, err
		}
		discount := promo.DiscountValue
		if promo.DiscountType == models.DiscountPercent {
			discount = price * promo.DiscountValue / 100
		}
		discount = math.Min(math.Round(discount), price)
		quote.PromoCode = promo.Code
		quote.PromoCodeID = &promo.ID
		quote.DiscountAmount = discount
		quote.Amount = price - discount
	}

	return quote, nil
}

// ApplySuccessfulPayment moves the farm onto the plan it paid for and extends PaidUntil by the
// period, from whichever is later: now or the end of what's already paid
func ApplySuccessfulPayment(payment *models.Payment) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var farm models.Farm
		if err := tx.Omit("Subscription", "BillingInfo").First(&farm, payment.FarmID).Error; err != nil {
			return err
		}

		start := time.Now()
		if farm.PaidUntil != nil && farm.PaidUntil.After(start) {
			start = *farm.PaidUntil
		}
		paidUntil := start.AddDate(0, 1, 0)
		if payment.BillingPeriod == models.BillingYearly {
			paidUntil = start.AddDate(1, 0, 0)
		}

		updates := map[string]interface{}{"paid_until": paidUntil, "payment_status": "paid"}
		if payment.PlanID != nil {
			updates["plan_id"] = *payment.PlanID
		}
		if payment.BillingPeriod != "" {
			updates["billing_period"] = payment.BillingPeriod
		}
		if err := tx.Model(&models.Farm{}).Where("id = ?", farm.ID).Updates(updates).Error; err != nil {
			return err
		}

		if payment.PromoCodeID != nil {
			return tx.Model(&models.PromoCode{}).Where("id = ?", *payment.PromoCodeID).
				UpdateColumn("redemption_count", gorm.Expr("redemption_count + 1")).Error
		}
		return nil
	})
}

// ---- Admin management ----

// ListPlans returns plans in display order; activeOnly hides ones not offered to farms
func ListPlans(activeOnly bool) ([]models.Plan, error) {
	plans := []models.Plan{}
	q := db.DB.Order("sort_order, id")
	if activeOnly {
		q = q.Where("is_active = ?", true)
	}
	err := q.Find(&plans).Error
	return plans, err
}

func validatePlan(plan *models.Plan) error {
	plan.Name = strings.TrimSpace(plan.Name)
	if plan.Name == "" {
		return errors.New("plan name is required")
	}
	if plan.MonthlyPrice < 0 || plan.YearlyPrice < 0 || plan.MaxStaff < 0 || plan.MaxFlocks < 0 || plan.MaxBirds < 0 {
		return errors.New("prices and limits can't be negative")
	}
	if plan.MonthlyPrice == 0 && plan.YearlyPrice == 0 {
		return errors.New("set a monthly or yearly price")
	}
	features := models.FeatureSet{}
	for key, on := range plan.Features {
		if !isKnownFeature(key) {
			return fmt.Errorf("unknown feature %q", key)
		}
		features[key] = on
	}
	plan.Features = features
	return nil
}

// SavePlan creates or updates a plan; marking it default unmarks the others
func SavePlan(plan *models.Plan) error {
	if err := validatePlan(plan); err != nil {
		return err
	}
	return db.DB.Transaction(func(tx *gorm.DB) error {
		if plan.IsDefault {
			if err := tx.Model(&models.Plan{}).Where("id <> ?", plan.ID).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Save(plan).Error
	})
}

// DeletePlan removes an unused plan; one that farms are on is deactivated instead
func DeletePlan(planID uint) (deactivated bool, err error) {
	var inUse int64
	db.DB.Model(&models.Farm{}).Where("plan_id = ?", planID).Count(&inUse)
	if inUse > 0 {
		return true, db.DB.Model(&models.Plan{}).Where("id = ?", planID).
			Updates(map[string]interface{}{"is_active": false, "is_default": false}).Error
	}
	res := db.DB.Delete(&models.Plan{}, planID)
	if res.RowsAffected == 0 {
		return false, ErrPlanNotFound
	}
	return false, res.Error
}

// ListPromoCodes returns all promo codes, newest first
func ListPromoCodes() ([]models.PromoCode, error) {
	codes := []models.PromoCode{}
	err := db.DB.Order("created_at DESC").Find(&codes).Error
	return codes, err
}

// SavePromoCode creates or updates a promo code
func SavePromoCode(promo *models.PromoCode) error {
	promo.Code = strings.ToUpper(strings.TrimSpace(promo.Code))
	switch {
	case promo.Code == "":
		return errors.New("code is required")
	case promo.DiscountType != models.DiscountPercent && promo.DiscountType != models.DiscountFixed:
		return errors.New("discount type must be percent or fixed")
	case promo.DiscountValue <= 0:
		return errors.New("discount must be more than 0")
	case promo.DiscountType == models.DiscountPercent && promo.DiscountValue > 100:
		return errors.New("a percentage discount can't exceed 100")
	case promo.MaxRedemptions < 0:
		return errors.New("max redemptions can't be negative")
	}
	if promo.PlanID != nil {
		var count int64
		db.DB.Model(&models.Plan{}).Where("id = ?", *promo.PlanID).Count(&count)
		if count == 0 {
			return ErrPlanNotFound
		}
	}
	err := db.DB.Save(promo).Error
	if err != nil && strings.Contains(err.Error(), "Duplicate") {
		return errors.New("a promo code with that code already exists")
	}
	return err
}

// UpdatePlatformSettings saves the trial and billing settings
func UpdatePlatformSettings(in models.PlatformSettings) (*models.PlatformSettings, error) {
	if in.TrialDays < 0 || in.TrialDays > 365 || in.GracePeriodDays < 0 || in.GracePeriodDays > 60 {
		return nil, errors.New("trial must be 0-365 days and grace period 0-60 days")
	}
	settings, err := models.GetPlatformSettings()
	if err != nil {
		return nil, err
	}
	settings.TrialEnabled = in.TrialEnabled
	settings.TrialDays = in.TrialDays
	settings.GracePeriodDays = in.GracePeriodDays
	if c := strings.ToUpper(strings.TrimSpace(in.Currency)); c != "" {
		settings.Currency = c
	}
	return settings, db.DB.Save(settings).Error
}

// AdminUpdateFarmBilling lets the admin extend a farm's trial or move it to another plan
func AdminUpdateFarmBilling(farmID uint, trialEndsAt *time.Time, planID *uint) error {
	updates := map[string]interface{}{}
	if trialEndsAt != nil {
		updates["trial_ends_at"] = *trialEndsAt
	}
	if planID != nil {
		var count int64
		db.DB.Model(&models.Plan{}).Where("id = ?", *planID).Count(&count)
		if count == 0 {
			return ErrPlanNotFound
		}
		updates["plan_id"] = *planID
	}
	if len(updates) == 0 {
		return errors.New("nothing to update")
	}
	res := db.DB.Model(&models.Farm{}).Where("id = ?", farmID).Updates(updates)
	if res.Error == nil && res.RowsAffected == 0 {
		return errors.New("farm not found")
	}
	if res.Error == nil {
		_, err := GetFarmBillingStatus(farmID)
		return err
	}
	return res.Error
}
