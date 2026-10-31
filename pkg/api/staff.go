package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services"

	"github.com/gin-gonic/gin"
)

// StaffHandler handles farm staff, invitations and the user's farm list
type StaffHandler struct{}

// SetupStaffRoutes registers the /farm routes
func SetupStaffRoutes(r *gin.Engine) {
	handler := &StaffHandler{}
	ownerOnly := middlewares.RequireFarmRole(models.FarmRoleOwner)
	ownerOrManager := middlewares.RequireFarmRole(models.FarmRoleOwner, models.FarmRoleManager)

	// Farms the signed-in user belongs to, for picking which one to work in
	r.GET("/farm/me", middlewares.UserAuth(), handler.GetMyFarms)
	r.POST("/farm/new", middlewares.UserAuth(), handler.CreateFarm)

	// Invite landing + acceptance: the token is the credential, sign-in is optional
	r.GET("/farm/invite/:token", handler.GetInvite)
	r.POST("/farm/invite/:token/accept", handler.AcceptInvite)

	farm := r.Group("/farm").Use(middlewares.FarmAuth())
	{
		farm.PUT("/", ownerOnly, handler.UpdateFarm)
		farm.GET("/staff", handler.ListStaff)
		farm.POST("/staff/invite", ownerOrManager, handler.InviteStaff)
		farm.PUT("/staff/:id/role", ownerOnly, handler.ChangeRole)
		farm.DELETE("/staff/:id", ownerOnly, handler.RemoveStaff)
		farm.DELETE("/invites/:id", ownerOrManager, handler.RevokeInvite)
	}
}

// staffErrorStatus maps staff service errors to HTTP statuses
func staffErrorStatus(err error) int {
	switch {
	case errors.Is(err, services.ErrInviteNotFound), errors.Is(err, services.ErrMemberNotFound):
		return http.StatusNotFound
	case errors.Is(err, services.ErrInviteExpired), errors.Is(err, services.ErrInviteUsed):
		return http.StatusGone
	case errors.Is(err, services.ErrRoleNotAllowed), errors.Is(err, services.ErrInviteEmailMismatch):
		return http.StatusForbidden
	case errors.Is(err, services.ErrAlreadyMember), errors.Is(err, services.ErrAccountExists),
		errors.Is(err, services.ErrUsernameTaken), errors.Is(err, services.ErrLastOwner):
		return http.StatusConflict
	case errors.Is(err, services.ErrInvalidRole), errors.Is(err, services.ErrInvalidEmail),
		errors.Is(err, services.ErrWeakPassword), errors.Is(err, services.ErrInvalidFarmType):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func respondStaffError(c *gin.Context, err error) {
	if respondPlanLimit(c, err) {
		return
	}
	status := staffErrorStatus(err)
	if status == http.StatusInternalServerError {
		log.Printf("Staff request failed: %v", err)
		c.JSON(status, gin.H{"error": "Something went wrong"})
		return
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

// GetMyFarms lists the farms the user is an active member of
func (h *StaffHandler) GetMyFarms(c *gin.Context) {
	farms, err := services.GetUserFarms(c.GetUint("user_id"))
	if err != nil {
		respondStaffError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"farms": farms})
}

// UpdateFarm changes the farm's name, location, type, or marks its setup as done; omitted fields stay
func (h *StaffHandler) UpdateFarm(c *gin.Context) {
	farmID, _ := farmContext(c)
	var req services.FarmDetails
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if err := services.UpdateFarm(farmID, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var farm models.Farm
	if err := db.DB.Omit("Subscription", "BillingInfo").First(&farm, farmID).Error; err != nil {
		respondStaffError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"farm_id": farmID, "name": farm.Name, "location": farm.Location,
		"enterprises": services.SplitEnterprises(farm.Enterprises), "setup_complete": farm.SetupCompletedAt != nil})
}

// CreateFarm starts a new farm owned by the signed-in user, who must not own one already
func (h *StaffHandler) CreateFarm(c *gin.Context) {
	var req services.FarmDetails
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if req.Name == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "farm name is required"})
		return
	}
	farm, err := services.CreateOwnFarm(c.GetUint("user_id"), req)
	if err != nil {
		respondStaffError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"farm_id": farm.ID, "name": farm.Name, "role": models.FarmRoleOwner})
}

// ListStaff returns the farm's members and pending invitations
func (h *StaffHandler) ListStaff(c *gin.Context) {
	farmID, _ := farmContext(c)
	members, invites, err := services.ListStaff(farmID)
	if err != nil {
		respondStaffError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"members": members, "invites": invites})
}

// InviteStaff emails an invitation to join the farm
func (h *StaffHandler) InviteStaff(c *gin.Context) {
	farmID, userID := farmContext(c)
	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	invite, err := services.InviteStaff(farmID, userID, c.GetString("farm_role"), req.Email, req.Role)
	if err != nil {
		respondStaffError(c, err)
		return
	}
	c.JSON(http.StatusCreated, invite)
}

// ChangeRole changes a member's role
func (h *StaffHandler) ChangeRole(c *gin.Context) {
	farmID, _ := farmContext(c)
	memberID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	member, err := services.ChangeStaffRole(farmID, uint(memberID), req.Role)
	if err != nil {
		respondStaffError(c, err)
		return
	}
	c.JSON(http.StatusOK, member)
}

// RemoveStaff takes a member off the farm
func (h *StaffHandler) RemoveStaff(c *gin.Context) {
	farmID, _ := farmContext(c)
	memberID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	if err := services.RemoveStaff(farmID, uint(memberID)); err != nil {
		respondStaffError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Staff member removed"})
}

// RevokeInvite cancels a pending invitation
func (h *StaffHandler) RevokeInvite(c *gin.Context) {
	farmID, _ := farmContext(c)
	inviteID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	if err := services.RevokeInvite(farmID, uint(inviteID)); err != nil {
		respondStaffError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Invitation revoked"})
}

// GetInvite shows an invitation's farm and role before it's accepted
func (h *StaffHandler) GetInvite(c *gin.Context) {
	info, err := services.GetInviteInfo(c.Param("token"))
	if err != nil {
		respondStaffError(c, err)
		return
	}
	c.JSON(http.StatusOK, info)
}

// AcceptInvite joins the farm: a signed-in user joins as themselves, otherwise username + password
// create a new account (returned with a login token)
func (h *StaffHandler) AcceptInvite(c *gin.Context) {
	var callerID uint
	if header := c.GetHeader("Authorization"); header != "" {
		userID, err := middlewares.ParseUserToken(header)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token, please log in again"})
			return
		}
		callerID = userID
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if callerID == 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username and password are required"})
			return
		}
	}

	user, token, member, err := services.AcceptInvite(c.Param("token"), callerID, req.Username, req.Password)
	if err != nil {
		respondStaffError(c, err)
		return
	}

	resp := gin.H{"farm_id": member.FarmID, "role": member.Role, "user": userJSON(user)}
	if token != "" {
		resp["token"] = token
	}
	c.JSON(http.StatusOK, resp)
}
