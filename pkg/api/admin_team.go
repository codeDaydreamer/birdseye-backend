package api

import (
	"errors"
	"net/http"
	"strconv"

	"birdseye-backend/pkg/services"

	"github.com/gin-gonic/gin"
)

// The console's Team page: admins and open invites. Viewers can see it (GET) but the middleware
// stops them changing anything.
func setupAdminTeamRoutes(admin *gin.RouterGroup) {
	admin.GET("/team", handleAdminTeam)
	admin.POST("/team/invites", handleAdminInvite)
	admin.DELETE("/team/invites/:id", handleAdminRevokeInvite)
	admin.PATCH("/team/:id", handleAdminSetRole)
	admin.DELETE("/team/:id", handleAdminRemove)
}

func adminTeamError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrAdminFounder), errors.Is(err, services.ErrAdminSelf):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrAdminInviteGone):
		c.JSON(http.StatusGone, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrAdminExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	}
}

func callerAdminID(c *gin.Context) uint {
	id, _ := c.Get("user_id")
	v, _ := id.(uint)
	return v
}

func teamIDParam(c *gin.Context) (uint, bool) {
	n, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return uint(n), true
}

func handleAdminTeam(c *gin.Context) {
	admins, invites := services.AdminTeam()
	role, _ := c.Get("role")
	c.JSON(http.StatusOK, gin.H{"admins": admins, "invites": invites, "founder_id": services.AdminFounderID(),
		"roles": services.AdminRoles, "me": callerAdminID(c), "my_role": role})
}

func handleAdminInvite(c *gin.Context) {
	var in struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	inviter, err := services.GetAdminByID(callerAdminID(c))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sign in again"})
		return
	}
	inv, link, emailed, err := services.InviteAdmin(inviter, in.Email, in.Role)
	if err != nil {
		adminTeamError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"invite": inv, "link": link, "emailed": emailed})
}

func handleAdminRevokeInvite(c *gin.Context) {
	id, ok := teamIDParam(c)
	if !ok {
		return
	}
	if err := services.RevokeAdminInvite(id); err != nil {
		adminTeamError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func handleAdminSetRole(c *gin.Context) {
	id, ok := teamIDParam(c)
	if !ok {
		return
	}
	var in struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	a, err := services.SetAdminRole(callerAdminID(c), id, in.Role)
	if err != nil {
		adminTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"admin": a})
}

func handleAdminRemove(c *gin.Context) {
	id, ok := teamIDParam(c)
	if !ok {
		return
	}
	if err := services.RemoveAdmin(callerAdminID(c), id); err != nil {
		adminTeamError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func handleAdminInviteInfo(c *gin.Context) {
	inv, err := services.AdminInviteInfo(c.Param("token"))
	if err != nil {
		adminTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"email": inv.Email, "role": inv.Role})
}

func handleAdminAcceptInvite(c *gin.Context) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	admin, token, err := services.AcceptAdminInvite(c.Param("token"), in.Username, in.Password)
	if err != nil {
		adminTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "admin": admin})
}
