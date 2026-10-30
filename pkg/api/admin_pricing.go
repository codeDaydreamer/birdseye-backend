package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services"

	"github.com/gin-gonic/gin"
)

// setupAdminPricingRoutes registers the admin's pricing controls on the (already admin-only) group
func setupAdminPricingRoutes(admin *gin.RouterGroup) {
	admin.GET("/plan-features", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"features": services.PlanFeatureCatalog})
	})
	admin.GET("/plans", handleAdminListPlans)
	admin.POST("/plans", handleAdminSavePlan)
	admin.PUT("/plans/:id", handleAdminSavePlan)
	admin.DELETE("/plans/:id", handleAdminDeletePlan)

	admin.GET("/promo-codes", handleAdminListPromoCodes)
	admin.POST("/promo-codes", handleAdminSavePromoCode)
	admin.PUT("/promo-codes/:id", handleAdminSavePromoCode)

	admin.GET("/settings", handleAdminGetSettings)
	admin.PUT("/settings", handleAdminUpdateSettings)

	admin.PUT("/farms/:id/billing", handleAdminUpdateFarmBilling)
}

// SetupPlanRoutes registers the public list of plans (for the pricing page and the billing screen)
func SetupPlanRoutes(r *gin.Engine) {
	r.GET("/plans", func(c *gin.Context) {
		plans, err := services.ListPlans(true)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load plans"})
			return
		}
		settings, err := models.GetPlatformSettings()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load plans"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"plans":    plans,
			"features": services.PlanFeatureCatalog,
			"currency": settings.Currency,
			"trial":    gin.H{"enabled": settings.TrialEnabled, "days": settings.TrialDays},
		})
	})
}

func pathID(c *gin.Context) (uint, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return 0, false
	}
	return uint(id), true
}

func handleAdminListPlans(c *gin.Context) {
	plans, err := services.ListPlans(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load plans"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"plans": plans})
}

// handleAdminSavePlan creates (POST) or updates (PUT /:id) a plan. An update applies the body on
// top of the stored plan, so fields left out (e.g. is_active) keep their current value.
func handleAdminSavePlan(c *gin.Context) {
	plan := models.Plan{IsActive: true}
	if c.Param("id") != "" {
		id, ok := pathID(c)
		if !ok {
			return
		}
		if err := dbFirst(&plan, id); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
			return
		}
	}
	id, createdAt := plan.ID, plan.CreatedAt
	if err := c.ShouldBindJSON(&plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan"})
		return
	}
	plan.ID, plan.CreatedAt = id, createdAt

	if err := services.SavePlan(&plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, plan)
}

func handleAdminDeletePlan(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	deactivated, err := services.DeletePlan(id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrPlanNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	if deactivated {
		c.JSON(http.StatusOK, gin.H{"message": "Farms are on this plan, so it was deactivated instead of deleted", "deactivated": true})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Plan deleted"})
}

func handleAdminListPromoCodes(c *gin.Context) {
	codes, err := services.ListPromoCodes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load promo codes"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"promo_codes": codes})
}

// handleAdminSavePromoCode creates (POST) or updates (PUT /:id) a promo code. Deactivate a code
// with is_active=false rather than deleting it, so past payments keep their reference.
func handleAdminSavePromoCode(c *gin.Context) {
	promo := models.PromoCode{IsActive: true}
	if c.Param("id") != "" {
		id, ok := pathID(c)
		if !ok {
			return
		}
		if err := dbFirst(&promo, id); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Promo code not found"})
			return
		}
	}
	id, redemptions, createdAt := promo.ID, promo.RedemptionCount, promo.CreatedAt
	if err := c.ShouldBindJSON(&promo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid promo code"})
		return
	}
	// Clients can't change the ID or the redemption count
	promo.ID, promo.RedemptionCount, promo.CreatedAt = id, redemptions, createdAt

	if err := services.SavePromoCode(&promo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, promo)
}

func handleAdminGetSettings(c *gin.Context) {
	settings, err := models.GetPlatformSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load settings"})
		return
	}
	c.JSON(http.StatusOK, settings)
}

func handleAdminUpdateSettings(c *gin.Context) {
	var in models.PlatformSettings
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid settings"})
		return
	}
	settings, err := services.UpdatePlatformSettings(in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, settings)
}

// handleAdminUpdateFarmBilling extends a farm's trial and/or moves it to another plan
func handleAdminUpdateFarmBilling(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req struct {
		TrialEndsAt *time.Time `json:"trial_ends_at"`
		PlanID      *uint      `json:"plan_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if err := services.AdminUpdateFarmBilling(id, req.TrialEndsAt, req.PlanID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log.Printf("Admin updated billing for farm %d", id)
	c.JSON(http.StatusOK, farmBillingFields(id))
}

// requireFeature blocks a farm-scoped route group when the farm's plan doesn't include the feature.
// Chain after middlewares.FarmAuth.
func requireFeature(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, err := services.FarmHasFeature(c.GetUint("farm_id"), key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check plan"})
			c.Abort()
			return
		}
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Your plan doesn't include this feature", "feature": key, "upgrade_required": true})
			c.Abort()
			return
		}
		c.Next()
	}
}

// respondPlanLimit answers a plan limit error with 403 + upgrade_required; reports whether it did
func respondPlanLimit(c *gin.Context, err error) bool {
	var limitErr *services.PlanLimitError
	if errors.As(err, &limitErr) {
		c.JSON(http.StatusForbidden, gin.H{"error": limitErr.Error(), "limit": limitErr.Limit, "upgrade_required": true})
		return true
	}
	return false
}
