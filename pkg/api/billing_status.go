package api

import (
	"encoding/json"
	"log"

	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services"

	"github.com/gin-gonic/gin"
)

// farmBillingFields are a farm's billing fields, under the same names the client and admin console
// read from the user object (payment_status, is_trial_active, trial_ends_at, ...)
func farmBillingFields(farmID uint) gin.H {
	status, err := services.GetFarmBillingStatus(farmID)
	if err != nil {
		log.Printf("Failed to get billing status for farm %d: %v", farmID, err)
		return gin.H{}
	}

	var subscription models.Subscription
	db.DB.Where("farm_id = ?", farmID).Limit(1).Find(&subscription)
	var billingInfo models.BillingInfo
	db.DB.Where("farm_id = ?", farmID).Limit(1).Find(&billingInfo)

	price, err := services.SubscriptionPrice(farmID)
	if err != nil {
		price = services.StandardPrice
	}

	return gin.H{
		"subscription_price":         price,
		"payment_status":             status.PaymentStatus,
		"is_trial_active":            status.IsTrialActive,
		"trial_ends_at":              status.TrialEndsAt,
		"paid_until":                 status.PaidUntil,
		"needs_subscription_renewal": status.NeedsSubscriptionRenewal,
		"subscription":               subscription,
		"billing_info":               billingInfo,
	}
}

// userJSON turns a user into a mutable map (User.MarshalJSON already drops the password hash)
func userJSON(user *models.User) gin.H {
	out := gin.H{}
	raw, err := json.Marshal(user)
	if err == nil {
		json.Unmarshal(raw, &out)
	}
	return out
}

// userPayload is the user object returned at login and by /auth/me: the user, their farms, and the
// billing state of the farm they're working in (X-Farm-ID, else their only farm)
func userPayload(c *gin.Context, user *models.User) gin.H {
	out := userJSON(user)

	var farms []struct {
		FarmID uint   `json:"farm_id"`
		Name   string `json:"name"`
		Role   string `json:"role"`
	}
	db.DB.Table("farm_members").
		Select("farm_members.farm_id, farms.name, farm_members.role").
		Joins("JOIN farms ON farms.id = farm_members.farm_id").
		Where("farm_members.user_id = ? AND farm_members.status = ?", user.ID, "active").
		Scan(&farms)
	out["farms"] = farms

	member, err := middlewares.ResolveFarmMember(user.ID, c.GetHeader("X-Farm-ID"))
	if err != nil {
		return out // several farms and none chosen yet: the client picks one from "farms"
	}
	out["farm_id"] = member.FarmID
	out["farm_role"] = member.Role
	for k, v := range farmBillingFields(member.FarmID) {
		out[k] = v
	}
	return out
}

// adminUserPayload is a user as the admin console shows it, with their own farm's billing state
func adminUserPayload(user *models.User) gin.H {
	out := userJSON(user)
	farm, err := services.GetOwnedFarm(user.ID)
	if err != nil {
		return out
	}
	out["farm_id"] = farm.ID
	out["farm_name"] = farm.Name
	for k, v := range farmBillingFields(farm.ID) {
		out[k] = v
	}
	return out
}
