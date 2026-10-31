package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"os"
	"strings"
	"time"

	"birdseye-backend/pkg/broadcast"
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services/email"

	"gorm.io/gorm"
)

const inviteValidFor = 7 * 24 * time.Hour

var (
	ErrInvalidRole         = errors.New("role must be owner, manager or worker")
	ErrRoleNotAllowed      = errors.New("you can't give that role")
	ErrInvalidEmail        = errors.New("invalid email address")
	ErrAlreadyMember       = errors.New("this person is already on the farm's staff")
	ErrInviteNotFound      = errors.New("invitation not found")
	ErrInviteExpired       = errors.New("invitation has expired")
	ErrInviteUsed          = errors.New("invitation is no longer valid")
	ErrInviteEmailMismatch = errors.New("this invitation was sent to a different email address")
	ErrAccountExists       = errors.New("an account with this email already exists, log in to accept")
	ErrUsernameTaken       = errors.New("username is already taken")
	ErrWeakPassword        = errors.New("password must be at least 8 characters")
	ErrMemberNotFound      = errors.New("staff member not found")
	ErrLastOwner           = errors.New("a farm must keep at least one owner")
	ErrInvalidFarmType     = errors.New("what you keep must be from layers, broilers, kienyeji, breeders, chicks")
)

// StaffMember is a farm member as shown on the staff page
type StaffMember struct {
	ID        uint      `json:"id"` // FarmMember ID
	UserID    uint      `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"joined_at"`
}

// FarmInviteInfo is what the invite landing page shows before accepting
type FarmInviteInfo struct {
	FarmName      string    `json:"farm_name"`
	Email         string    `json:"email"`
	Role          string    `json:"role"`
	ExpiresAt     time.Time `json:"expires_at"`
	AccountExists bool      `json:"account_exists"`
}

func validRole(role string) bool {
	return role == models.FarmRoleOwner || role == models.FarmRoleManager || role == models.FarmRoleWorker
}

// hashInviteToken is what's stored, so a database leak doesn't expose usable invite links
func hashInviteToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func inviteURL(token string) string {
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "https://app.birdseye-poultry.com"
	}
	return fmt.Sprintf("%s/invite/%s", strings.TrimRight(frontendURL, "/"), token)
}

// InviteStaff creates an invitation and emails its link. Owners can invite any role; managers can
// only invite workers. Any earlier pending invite for the same email is replaced.
func InviteStaff(farmID, inviterID uint, inviterRole, emailAddr, role string) (*models.FarmInvite, error) {
	if !validRole(role) {
		return nil, ErrInvalidRole
	}
	if inviterRole != models.FarmRoleOwner && role != models.FarmRoleWorker {
		return nil, ErrRoleNotAllowed
	}
	addr, err := mail.ParseAddress(strings.TrimSpace(emailAddr))
	if err != nil {
		return nil, ErrInvalidEmail
	}
	emailAddr = strings.ToLower(addr.Address)

	var activeCount int64
	db.DB.Model(&models.FarmMember{}).
		Joins("JOIN users ON users.id = farm_members.user_id").
		Where("farm_members.farm_id = ? AND farm_members.status = ? AND LOWER(users.email) = ?", farmID, "active", emailAddr).
		Count(&activeCount)
	if activeCount > 0 {
		return nil, ErrAlreadyMember
	}

	if err := CheckStaffLimit(farmID, true); err != nil {
		return nil, err
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(raw)

	invite := models.FarmInvite{
		FarmID:    farmID,
		Email:     emailAddr,
		Role:      role,
		Token:     hashInviteToken(token),
		InvitedBy: inviterID,
		Status:    "pending",
		ExpiresAt: time.Now().Add(inviteValidFor),
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.FarmInvite{}).
			Where("farm_id = ? AND email = ? AND status = ?", farmID, emailAddr, "pending").
			Update("status", "revoked").Error; err != nil {
			return err
		}
		return tx.Create(&invite).Error
	})
	if err != nil {
		return nil, err
	}

	var farm models.Farm
	db.DB.Omit("Subscription", "BillingInfo").First(&farm, farmID)
	var inviter models.User
	db.DB.First(&inviter, inviterID)

	go func() {
		if err := email.SendFarmInviteEmail(emailAddr, farm.Name, inviter.Username, role, inviteURL(token)); err != nil {
			log.Printf("Failed to send farm invite email to %s: %v", emailAddr, err)
		}
	}()

	return &invite, nil
}

// findPendingInvite loads a still-usable invite by its raw token
func findPendingInvite(tx *gorm.DB, token string) (*models.FarmInvite, error) {
	var invite models.FarmInvite
	if err := tx.Where("token = ?", hashInviteToken(token)).First(&invite).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInviteNotFound
		}
		return nil, err
	}
	if invite.Status != "pending" {
		return nil, ErrInviteUsed
	}
	if time.Now().After(invite.ExpiresAt) {
		tx.Model(&invite).Update("status", "expired")
		return nil, ErrInviteExpired
	}
	return &invite, nil
}

// GetInviteInfo returns what the invite landing page needs, without accepting it
func GetInviteInfo(token string) (*FarmInviteInfo, error) {
	invite, err := findPendingInvite(db.DB, token)
	if err != nil {
		return nil, err
	}

	var farm models.Farm
	if err := db.DB.Omit("Subscription", "BillingInfo").First(&farm, invite.FarmID).Error; err != nil {
		return nil, err
	}

	var accounts int64
	db.DB.Model(&models.User{}).Where("LOWER(email) = ?", invite.Email).Count(&accounts)

	return &FarmInviteInfo{
		FarmName:      farm.Name,
		Email:         invite.Email,
		Role:          invite.Role,
		ExpiresAt:     invite.ExpiresAt,
		AccountExists: accounts > 0,
	}, nil
}

// AcceptInvite adds a user to the invite's farm. A signed-in caller (callerUserID != 0) must own the
// invited email; otherwise a new account is created from username/password (the invite proves the
// email, so it starts verified). Returns the user and, for a new account, a login token.
func AcceptInvite(token string, callerUserID uint, username, password string) (*models.User, string, *models.FarmMember, error) {
	var user models.User
	var member models.FarmMember
	newAccount := false

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		invite, err := findPendingInvite(tx, token)
		if err != nil {
			return err
		}

		if callerUserID != 0 {
			if err := tx.First(&user, callerUserID).Error; err != nil {
				return err
			}
			if !strings.EqualFold(user.Email, invite.Email) {
				return ErrInviteEmailMismatch
			}
		} else {
			var accounts int64
			tx.Model(&models.User{}).Where("LOWER(email) = ?", invite.Email).Count(&accounts)
			if accounts > 0 {
				return ErrAccountExists
			}

			username = strings.TrimSpace(username)
			if username == "" {
				return errors.New("username is required")
			}
			if len(password) < 8 {
				return ErrWeakPassword
			}
			var taken int64
			tx.Model(&models.User{}).Where("username = ?", username).Count(&taken)
			if taken > 0 {
				return ErrUsernameTaken
			}

			user = models.User{
				Username:      username,
				Email:         invite.Email,
				Password:      password,
				EmailVerified: true,
			}
			if err := user.HashPassword(); err != nil {
				return err
			}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
			newAccount = true
		}

		if err := CheckStaffLimit(invite.FarmID, false); err != nil {
			return err
		}

		// Reuse the row if they were on this farm before (one row per farm/user pair)
		inviter := invite.InvitedBy
		err = tx.Where("farm_id = ? AND user_id = ?", invite.FarmID, user.ID).First(&member).Error
		switch {
		case err == nil:
			if member.Status == "active" {
				return ErrAlreadyMember
			}
			member.Role, member.Status, member.InvitedBy = invite.Role, "active", &inviter
			if err := tx.Save(&member).Error; err != nil {
				return err
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			member = models.FarmMember{FarmID: invite.FarmID, UserID: user.ID, Role: invite.Role, InvitedBy: &inviter, Status: "active"}
			if err := tx.Create(&member).Error; err != nil {
				return err
			}
		default:
			return err
		}

		return tx.Model(invite).Update("status", "accepted").Error
	})
	if err != nil {
		return nil, "", nil, err
	}

	loginToken := ""
	if newAccount {
		if loginToken, err = generateJWT(user); err != nil {
			return nil, "", nil, err
		}
	}
	return &user, loginToken, &member, nil
}

// ListStaff returns the farm's active members and its pending invitations
func ListStaff(farmID uint) ([]StaffMember, []models.FarmInvite, error) {
	members := []StaffMember{}
	if err := db.DB.Table("farm_members").
		Select("farm_members.id, farm_members.user_id, users.username, users.email, farm_members.role, farm_members.created_at").
		Joins("JOIN users ON users.id = farm_members.user_id").
		Where("farm_members.farm_id = ? AND farm_members.status = ?", farmID, "active").
		Order("farm_members.created_at").
		Scan(&members).Error; err != nil {
		return nil, nil, err
	}

	invites := []models.FarmInvite{}
	if err := db.DB.Where("farm_id = ? AND status = ? AND expires_at > ?", farmID, "pending", time.Now()).
		Order("created_at DESC").Find(&invites).Error; err != nil {
		return nil, nil, err
	}

	return members, invites, nil
}

// activeOwnerCount counts the farm's active owners, for the last-owner guard
func activeOwnerCount(tx *gorm.DB, farmID uint) int64 {
	var n int64
	tx.Model(&models.FarmMember{}).
		Where("farm_id = ? AND role = ? AND status = ?", farmID, models.FarmRoleOwner, "active").
		Count(&n)
	return n
}

// ChangeStaffRole changes a member's role, refusing to demote the farm's last owner
func ChangeStaffRole(farmID, memberID uint, role string) (*models.FarmMember, error) {
	if !validRole(role) {
		return nil, ErrInvalidRole
	}

	var member models.FarmMember
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND farm_id = ? AND status = ?", memberID, farmID, "active").First(&member).Error; err != nil {
			return ErrMemberNotFound
		}
		if member.Role == models.FarmRoleOwner && role != models.FarmRoleOwner && activeOwnerCount(tx, farmID) <= 1 {
			return ErrLastOwner
		}
		member.Role = role
		return tx.Save(&member).Error
	})
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// RemoveStaff takes a member off the farm (kept as "removed" so their records keep their author),
// refusing to remove the last owner, and drops their live connection to the farm
func RemoveStaff(farmID, memberID uint) error {
	var member models.FarmMember
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND farm_id = ? AND status = ?", memberID, farmID, "active").First(&member).Error; err != nil {
			return ErrMemberNotFound
		}
		if member.Role == models.FarmRoleOwner && activeOwnerCount(tx, farmID) <= 1 {
			return ErrLastOwner
		}
		return tx.Model(&member).Update("status", "removed").Error
	})
	if err != nil {
		return err
	}

	broadcast.DisconnectFarmMember(farmID, member.UserID)
	return nil
}

// RevokeInvite cancels a pending invitation
func RevokeInvite(farmID, inviteID uint) error {
	res := db.DB.Model(&models.FarmInvite{}).
		Where("id = ? AND farm_id = ? AND status = ?", inviteID, farmID, "pending").
		Update("status", "revoked")
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrInviteNotFound
	}
	return nil
}

// FarmSummary is one of the user's farms, for the farm switcher
type FarmSummary struct {
	FarmID        uint     `json:"farm_id"`
	Name          string   `json:"name"`
	Role          string   `json:"role"`
	Location      string   `json:"location"`
	EnterpriseCSV string   `json:"-" gorm:"column:enterprises"`
	Enterprises   []string `json:"enterprises" gorm:"-"`
	SetupComplete bool     `json:"setup_complete"`
}

// SplitEnterprises turns the stored comma list into a list (never nil)
func SplitEnterprises(csv string) []string {
	list := []string{}
	for _, e := range strings.Split(csv, ",") {
		if e = strings.TrimSpace(e); e != "" {
			list = append(list, e)
		}
	}
	return list
}

// GetUserFarms lists the farms the user is an active member of
func GetUserFarms(userID uint) ([]FarmSummary, error) {
	farms := []FarmSummary{}
	err := db.DB.Table("farm_members").
		Select("farm_members.farm_id, farms.name, farm_members.role, farms.location, farms.enterprises, "+
			"farms.setup_completed_at IS NOT NULL AS setup_complete").
		Joins("JOIN farms ON farms.id = farm_members.farm_id").
		Where("farm_members.user_id = ? AND farm_members.status = ?", userID, "active").
		Order("farm_members.farm_id").
		Scan(&farms).Error
	for i := range farms {
		farms[i].Enterprises = SplitEnterprises(farms[i].EnterpriseCSV)
	}
	return farms, err
}

// FarmDetails is what an owner can change about their farm; nil fields are left as they are
type FarmDetails struct {
	Name          *string   `json:"name"`
	Location      *string   `json:"location"`
	Enterprises   *[]string `json:"enterprises"`
	SetupComplete *bool     `json:"setup_complete"`
}

// farmUpdates validates the details and turns them into column updates
func (d FarmDetails) farmUpdates() (map[string]interface{}, error) {
	updates := map[string]interface{}{}
	if d.Name != nil {
		name := strings.TrimSpace(*d.Name)
		if name == "" || len(name) > 100 {
			return nil, errors.New("farm name must be 1-100 characters")
		}
		updates["name"] = name
	}
	if d.Location != nil {
		location := strings.TrimSpace(*d.Location)
		if len(location) > 120 {
			return nil, errors.New("location must be at most 120 characters")
		}
		updates["location"] = location
	}
	if d.Enterprises != nil {
		seen := map[string]bool{}
		kept := []string{}
		for _, e := range *d.Enterprises {
			e = strings.ToLower(strings.TrimSpace(e))
			if !models.ProductionTypes[e] {
				return nil, ErrInvalidFarmType
			}
			if !seen[e] {
				seen[e] = true
				kept = append(kept, e)
			}
		}
		updates["enterprises"] = strings.Join(kept, ",")
	}
	if d.SetupComplete != nil {
		if *d.SetupComplete {
			updates["setup_completed_at"] = time.Now()
		} else {
			updates["setup_completed_at"] = nil
		}
	}
	return updates, nil
}

// UpdateFarm changes the farm's name, location, type or setup state
func UpdateFarm(farmID uint, details FarmDetails) error {
	updates, err := details.farmUpdates()
	if err != nil || len(updates) == 0 {
		return err
	}
	return db.DB.Model(&models.Farm{}).Where("id = ?", farmID).Updates(updates).Error
}

// CreateOwnFarm sets up another farm owned by the user. Their first owned farm gets the free trial; any
// later one starts unpaid.
func CreateOwnFarm(userID uint, details FarmDetails) (*models.Farm, error) {
	updates, err := details.farmUpdates()
	if err != nil {
		return nil, err
	}
	var farm *models.Farm
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		var owned int64
		if err := tx.Model(&models.Farm{}).Where("owner_user_id = ?", userID).Count(&owned).Error; err != nil {
			return err
		}
		var user models.User
		if err := tx.First(&user, userID).Error; err != nil {
			return err
		}
		if farm, err = models.CreateFarmForUser(tx, &user); err != nil {
			return err
		}
		// Only a person's first farm gets the free trial (user decision 2026-10-31): later farms start unpaid
		// and are activated in Billing, so new farms can't be used to keep trialling for free
		if owned > 0 {
			now := time.Now()
			farm.TrialEndsAt, farm.PaymentStatus = now, "unpaid"
			if err := tx.Model(farm).Updates(map[string]interface{}{"trial_ends_at": now, "payment_status": "unpaid"}).Error; err != nil {
				return err
			}
		}
		if len(updates) > 0 {
			return tx.Model(farm).Updates(updates).Error
		}
		return nil
	})
	return farm, err
}
