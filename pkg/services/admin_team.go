package services

// The admin console's team: the first admin is created once through setup (while no admin exists);
// everyone after that joins through an emailed invite. The first admin (lowest ID) is protected for
// good: never removed, role never changed.

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"time"

	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services/email"

	"golang.org/x/crypto/bcrypt"
)

const adminInviteValidFor = 72 * time.Hour

var (
	ErrAdminSetupDone   = errors.New("setup is already done; ask an admin to invite you")
	ErrAdminFounder     = errors.New("this is the first admin account: it can't be removed and its role can't change")
	ErrAdminInviteGone  = errors.New("this invite link has expired or was already used")
	ErrAdminBadRole     = errors.New("role must be admin or viewer")
	ErrAdminSelf        = errors.New("you can't change or remove your own account here")
	ErrAdminExists      = errors.New("an admin with this email or username already exists")
	ErrAdminWeakPass    = errors.New("use a password of at least 10 characters")
	ErrAdminBadUsername = errors.New("choose a username of 3 to 32 letters, numbers, dots, dashes or underscores")
)

// AdminRoles are the console roles: admin can do everything, viewer can only look
var AdminRoles = []string{"admin", "viewer"}

func validAdminRole(r string) bool { return r == "admin" || r == "viewer" }

// AdminSetupRequired is true until the first admin exists
func AdminSetupRequired() bool {
	var n int64
	db.DB.Model(&models.Admin{}).Count(&n)
	return n == 0
}

// AdminFounderID is the first admin's ID (0 when there is none)
func AdminFounderID() uint {
	var first models.Admin
	if db.DB.Select("id").Order("id").First(&first).Error != nil {
		return 0
	}
	return first.ID
}

func checkAdminCredentials(username, email, password string) error {
	if len(password) < 10 {
		return ErrAdminWeakPass
	}
	if len(username) < 3 || len(username) > 32 || strings.Trim(username, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-") != "" {
		return ErrAdminBadUsername
	}
	var n int64
	db.DB.Model(&models.Admin{}).Where("email = ? OR username = ?", email, username).Count(&n)
	if n > 0 {
		return ErrAdminExists
	}
	return nil
}

func createAdmin(username, emailAddr, password, role string) (*models.Admin, string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}
	a := &models.Admin{Username: username, Email: emailAddr, Password: string(hash), Role: role}
	if err := db.DB.Create(a).Error; err != nil {
		return nil, "", err
	}
	token, err := generateAdminJWT(a.ID, a.Username)
	if err != nil {
		return nil, "", err
	}
	a.Password = ""
	return a, token, nil
}

// SetupFirstAdmin creates the first admin and signs them in. It only works while no admin exists.
func SetupFirstAdmin(username, emailAddr, password string) (*models.Admin, string, error) {
	username, emailAddr = strings.TrimSpace(username), strings.ToLower(strings.TrimSpace(emailAddr))
	if !AdminSetupRequired() {
		return nil, "", ErrAdminSetupDone
	}
	if !strings.Contains(emailAddr, "@") {
		return nil, "", errors.New("enter a valid email address")
	}
	if err := checkAdminCredentials(username, emailAddr, password); err != nil {
		return nil, "", err
	}
	return createAdmin(username, emailAddr, password, "admin")
}

func hashAdminToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

func adminConsoleURL() string {
	if u := strings.TrimRight(os.Getenv("ADMIN_CONSOLE_URL"), "/"); u != "" {
		return u
	}
	return "https://console.birdseye-poultry.com"
}

// InviteAdmin creates an invite, emails its link, and returns the invite with the link (shown once)
func InviteAdmin(inviter *models.Admin, emailAddr, role string) (*models.AdminInvite, string, bool, error) {
	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))
	if !strings.Contains(emailAddr, "@") {
		return nil, "", false, errors.New("enter a valid email address")
	}
	if !validAdminRole(role) {
		return nil, "", false, ErrAdminBadRole
	}
	var n int64
	db.DB.Model(&models.Admin{}).Where("email = ?", emailAddr).Count(&n)
	if n > 0 {
		return nil, "", false, ErrAdminExists
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", false, err
	}
	token := hex.EncodeToString(raw)
	now := time.Now()
	// An earlier open invite for the same person stops working
	db.DB.Model(&models.AdminInvite{}).Where("email = ? AND accepted_at IS NULL AND revoked_at IS NULL", emailAddr).Update("revoked_at", now)
	inv := &models.AdminInvite{Email: emailAddr, Role: role, TokenHash: hashAdminToken(token), InvitedByID: inviter.ID, ExpiresAt: now.Add(adminInviteValidFor)}
	if err := db.DB.Create(inv).Error; err != nil {
		return nil, "", false, err
	}
	link := adminConsoleURL() + "/invite/" + token
	emailed := email.SendAdminInviteEmail(emailAddr, inviter.Username, role, link, int(adminInviteValidFor.Hours()/24)) == nil
	return inv, link, emailed, nil
}

func findAdminInvite(token string) (*models.AdminInvite, error) {
	var inv models.AdminInvite
	if token == "" || db.DB.Where("token_hash = ?", hashAdminToken(token)).First(&inv).Error != nil ||
		inv.AcceptedAt != nil || inv.RevokedAt != nil || time.Now().After(inv.ExpiresAt) {
		return nil, ErrAdminInviteGone
	}
	return &inv, nil
}

// AdminInviteInfo says who an invite is for, so the accept page can show it
func AdminInviteInfo(token string) (*models.AdminInvite, error) { return findAdminInvite(token) }

// AcceptAdminInvite creates the invited admin and signs them in
func AcceptAdminInvite(token, username, password string) (*models.Admin, string, error) {
	inv, err := findAdminInvite(token)
	if err != nil {
		return nil, "", err
	}
	username = strings.TrimSpace(username)
	if err := checkAdminCredentials(username, inv.Email, password); err != nil {
		return nil, "", err
	}
	now := time.Now()
	// Claim the invite first so a link can't be used twice at the same moment
	res := db.DB.Model(&models.AdminInvite{}).Where("id = ? AND accepted_at IS NULL AND revoked_at IS NULL", inv.ID).Update("accepted_at", now)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil, "", ErrAdminInviteGone
	}
	return createAdmin(username, inv.Email, password, inv.Role)
}

// AdminTeam lists the admins and the open invites
func AdminTeam() ([]models.Admin, []models.AdminInvite) {
	admins := []models.Admin{}
	db.DB.Select("id, username, email, role, created_at, updated_at").Order("id").Find(&admins)
	invites := []models.AdminInvite{}
	db.DB.Where("accepted_at IS NULL AND revoked_at IS NULL AND expires_at > ?", time.Now()).Order("created_at DESC").Find(&invites)
	return admins, invites
}

// RevokeAdminInvite stops an open invite from working
func RevokeAdminInvite(id uint) error {
	return db.DB.Model(&models.AdminInvite{}).Where("id = ? AND accepted_at IS NULL", id).Update("revoked_at", time.Now()).Error
}

func otherAdmin(callerID, id uint) (*models.Admin, error) {
	if id == callerID {
		return nil, ErrAdminSelf
	}
	if id == AdminFounderID() {
		return nil, ErrAdminFounder
	}
	return GetAdminByID(id)
}

// SetAdminRole changes another admin's role (never the first admin's)
func SetAdminRole(callerID, id uint, role string) (*models.Admin, error) {
	if !validAdminRole(role) {
		return nil, ErrAdminBadRole
	}
	a, err := otherAdmin(callerID, id)
	if err != nil {
		return nil, err
	}
	if err := db.DB.Model(a).Update("role", role).Error; err != nil {
		return nil, err
	}
	a.Password = ""
	return a, nil
}

// RemoveAdmin deletes another admin (never the first admin). Their tokens stop working at once
// because the console checks the admin still exists on every request.
func RemoveAdmin(callerID, id uint) error {
	a, err := otherAdmin(callerID, id)
	if err != nil {
		return err
	}
	return db.DB.Delete(a).Error
}
