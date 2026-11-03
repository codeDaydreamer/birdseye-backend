package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"birdseye-backend/pkg/appconfig"
	"birdseye-backend/pkg/services"
	"birdseye-backend/pkg/services/email"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/gin-gonic/gin"
)

// Integration keys (Resend, VAPID, Firebase, Google, Dynapay) managed from the console. Secret values
// never leave the server: the console sees whether each key is set, where it comes from and its last
// 4 characters. Changes need the admin's password and are recorded by adminAudit (without the values).
func setupAdminConfigRoutes(admin *gin.RouterGroup) {
	admin.GET("/config", handleAdminGetConfig)
	admin.PUT("/config", handleAdminSaveConfig)
	admin.POST("/config/vapid", handleAdminGenerateVAPID)
	admin.POST("/config/test/email", handleAdminTestEmail)
	admin.POST("/config/test/dynapay", handleAdminTestDynapay)
}

// HandleVAPIDPublicKey serves the Web Push public key so the web app needn't be rebuilt when it changes
func HandleVAPIDPublicKey(c *gin.Context) {
	key := appconfig.Get("VAPID_PUBLIC_KEY")
	if key == "" || appconfig.Get("VAPID_PRIVATE_KEY") == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "push notifications aren't set up"})
		return
	}
	c.Header("Cache-Control", "no-cache")
	c.JSON(http.StatusOK, gin.H{"public_key": key})
}

func currentAdminID(c *gin.Context) uint {
	v, _ := c.Get("user_id")
	id, _ := v.(uint)
	return id
}

// confirmPassword checks the password in the request; it writes the error response itself
func confirmPassword(c *gin.Context, password string) bool {
	if strings.TrimSpace(password) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enter your password to confirm"})
		return false
	}
	if err := services.VerifyAdminPassword(currentAdminID(c), password); err != nil {
		time.Sleep(500 * time.Millisecond) // slow down guessing
		c.JSON(http.StatusUnauthorized, gin.H{"error": "That password is wrong"})
		return false
	}
	return true
}

func handleAdminGetConfig(c *gin.Context) {
	type fieldView struct {
		appconfig.Field
		Source    string     `json:"source"` // console, env or ""
		Preview   string     `json:"preview"`
		UpdatedAt *time.Time `json:"updated_at,omitempty"`
	}
	type groupView struct {
		ID         string      `json:"id"`
		Title      string      `json:"title"`
		About      string      `json:"about"`
		Configured bool        `json:"configured"`
		Fields     []fieldView `json:"fields"`
	}
	out := []groupView{}
	for _, g := range appconfig.Groups {
		gv := groupView{ID: g.ID, Title: g.Title, About: g.About, Configured: true}
		for _, f := range g.Fields {
			v := appconfig.Get(f.Key)
			preview := appconfig.Preview(f, v)
			if f.Key == "FIREBASE_SERVICE_ACCOUNT_JSON" && v != "" {
				var sa struct {
					ProjectID   string `json:"project_id"`
					ClientEmail string `json:"client_email"`
				}
				if json.Unmarshal([]byte(v), &sa) == nil && sa.ClientEmail != "" {
					preview = sa.ClientEmail
				}
			}
			optional := f.Key == "EMAIL_FROM" || f.Key == "VAPID_SUBSCRIBER" || f.Key == "GOOGLE_ALLOWED_CLIENT_IDS"
			if v == "" && !optional {
				gv.Configured = false
			}
			gv.Fields = append(gv.Fields, fieldView{Field: f, Source: appconfig.Source(f.Key), Preview: preview, UpdatedAt: appconfig.UpdatedAt(f.Key)})
		}
		if g.ID == "fcm" && appconfig.Get("FIREBASE_SERVICE_ACCOUNT_JSON") == "" && appconfig.Get("FIREBASE_SERVICE_ACCOUNT_FILE") != "" {
			gv.Configured = true
			gv.Fields[0].Source, gv.Fields[0].Preview = "env", "file: "+appconfig.Get("FIREBASE_SERVICE_ACCOUNT_FILE")
		}
		out = append(out, gv)
	}
	c.JSON(http.StatusOK, gin.H{"groups": out})
}

func handleAdminSaveConfig(c *gin.Context) {
	var req struct {
		Password string            `json:"password"`
		Set      map[string]string `json:"set"`
		Clear    []string          `json:"clear"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || (len(req.Set) == 0 && len(req.Clear) == 0) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nothing to save"})
		return
	}
	if !confirmPassword(c, req.Password) {
		return
	}
	if req.Set == nil {
		req.Set = map[string]string{}
	}
	if err := appconfig.Save(req.Set, req.Clear, currentAdminID(c)); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	handleAdminGetConfig(c)
}

// handleAdminGenerateVAPID makes and saves a new Web Push key pair (the private key is never shown)
func handleAdminGenerateVAPID(c *gin.Context) {
	var req struct {
		Password string `json:"password"`
	}
	_ = c.ShouldBindJSON(&req)
	if !confirmPassword(c, req.Password) {
		return
	}
	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Couldn't generate keys"})
		return
	}
	if err := appconfig.Save(map[string]string{"VAPID_PUBLIC_KEY": pub, "VAPID_PRIVATE_KEY": priv}, nil, currentAdminID(c)); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	handleAdminGetConfig(c)
}

func handleAdminTestEmail(c *gin.Context) {
	var req struct {
		To string `json:"to" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Enter an email address"})
		return
	}
	body := `<p>This is a test email from the Birdseye admin console.</p><p>If you can read it, Resend is set up correctly.</p>`
	if err := email.SendEmail(req.To, "Birdseye test email", body); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Sent to " + req.To})
}

func handleAdminTestDynapay(c *gin.Context) {
	if err := services.Dynapay.Ping(); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Connected: Dynapay accepted the API key"})
}
