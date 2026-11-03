// Package appconfig holds the integration keys an admin can manage from the console: Resend, Web Push
// (VAPID), Firebase, Google sign-in and Dynapay. A value saved in the console (encrypted in the
// app_settings table) wins over the environment variable of the same name, so a server can start from
// .env and be reconfigured later without a restart.
//
// Values are encrypted with AES-256-GCM. The key is SETTINGS_ENCRYPTION_KEY (32 bytes, base64) when set,
// otherwise it is derived from JWT_SECRET; changing whichever is in use means re-entering the keys.
//
// Bootstrap settings (database, JWT_SECRET, PORT) are not here: the server needs them to start.
package appconfig

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
)

// Field describes one managed key
type Field struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Secret      bool   `json:"secret"`
	Multiline   bool   `json:"multiline,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Help        string `json:"help,omitempty"`
}

// Group is a set of fields that configure one integration; Reload runs after any of them change
type Group struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	About  string  `json:"about"`
	Fields []Field `json:"fields"`
}

// Groups lists every key the console can manage, in display order
var Groups = []Group{
	{ID: "email", Title: "Email (Resend)", About: "Sign-up codes, password resets, invites, reminders and receipts.", Fields: []Field{
		{Key: "RESEND_API_KEY", Label: "Resend API key", Secret: true, Placeholder: "re_...", Help: "resend.com > API keys. Needs sending access for the birdseye-poultry.com domain."},
		{Key: "EMAIL_FROM", Label: "Send as", Placeholder: "Birdseye <noreply@birdseye-poultry.com>", Help: "Must be on a domain verified in Resend."},
	}},
	{ID: "push", Title: "Browser push (VAPID)", About: "Notifications in the web app. Changing the key pair signs every browser out of push until they open the app again.", Fields: []Field{
		{Key: "VAPID_PUBLIC_KEY", Label: "Public key", Placeholder: "B..."},
		{Key: "VAPID_PRIVATE_KEY", Label: "Private key", Secret: true},
		{Key: "VAPID_SUBSCRIBER", Label: "Contact", Placeholder: "mailto:support@birdseye-poultry.com", Help: "Push services contact this address about problems."},
	}},
	{ID: "fcm", Title: "Mobile push (Firebase)", About: "Notifications in the Android app.", Fields: []Field{
		{Key: "FIREBASE_SERVICE_ACCOUNT_JSON", Label: "Service account JSON", Secret: true, Multiline: true, Placeholder: `{"type": "service_account", ...}`,
			Help: "Firebase console > Project settings > Service accounts > Generate new private key. Paste the whole file."},
	}},
	{ID: "google", Title: "Google sign-in", About: "\"Continue with Google\" in the web and mobile apps.", Fields: []Field{
		{Key: "GOOGLE_CLIENT_ID", Label: "Web client ID", Placeholder: "....apps.googleusercontent.com"},
		{Key: "GOOGLE_CLIENT_SECRET", Label: "Web client secret", Secret: true},
		{Key: "GOOGLE_REDIRECT_URL", Label: "Redirect URL", Placeholder: "https://server.birdseye-poultry.com/auth/google/callback"},
		{Key: "GOOGLE_ALLOWED_CLIENT_IDS", Label: "Other allowed client IDs", Placeholder: "android-id.apps.googleusercontent.com", Help: "Comma-separated: the Android app's client IDs."},
	}},
	{ID: "dynapay", Title: "Payments (Dynapay)", About: "M-Pesa subscription payments. Get these from the Dynapay dashboard: Developers > API keys and Webhooks.", Fields: []Field{
		{Key: "DYNAPAY_API_URL", Label: "API URL", Placeholder: "https://dynapay.birdseye-poultry.com"},
		{Key: "DYNAPAY_CLIENT_ID", Label: "Client ID", Placeholder: "dp_live_..."},
		{Key: "DYNAPAY_CLIENT_SECRET", Label: "Client secret", Secret: true},
		{Key: "DYNAPAY_WEBHOOK_SECRET", Label: "Webhook signing secret", Secret: true, Help: "Shown once when the webhook endpoint is set in Dynapay."},
	}},
}

var (
	mu      sync.RWMutex
	values  = map[string]string{} // decrypted console values
	updated = map[string]time.Time{}
	reloads = map[string][]func(){}
	aead    cipher.AEAD
)

// FieldByKey finds a managed field and its group
func FieldByKey(key string) (*Field, *Group) {
	for gi := range Groups {
		for fi := range Groups[gi].Fields {
			if Groups[gi].Fields[fi].Key == key {
				return &Groups[gi].Fields[fi], &Groups[gi]
			}
		}
	}
	return nil, nil
}

// Get returns the console value for key, or the environment variable when none is saved
func Get(key string) string {
	mu.RLock()
	v, ok := values[key]
	mu.RUnlock()
	if ok {
		return v
	}
	return os.Getenv(key)
}

// Source says where key's value comes from: "console", "env" or "" (not set)
func Source(key string) string {
	mu.RLock()
	_, ok := values[key]
	mu.RUnlock()
	switch {
	case ok:
		return "console"
	case os.Getenv(key) != "":
		return "env"
	}
	return ""
}

// UpdatedAt is when key was last saved from the console
func UpdatedAt(key string) *time.Time {
	mu.RLock()
	defer mu.RUnlock()
	if t, ok := updated[key]; ok {
		return &t
	}
	return nil
}

// OnChange registers fn to run after any key in group changes
func OnChange(group string, fn func()) {
	mu.Lock()
	reloads[group] = append(reloads[group], fn)
	mu.Unlock()
}

func cipherKey() []byte {
	if raw := strings.TrimSpace(os.Getenv("SETTINGS_ENCRYPTION_KEY")); raw != "" {
		if k, err := base64.StdEncoding.DecodeString(raw); err == nil && len(k) == 32 {
			return k
		}
		log.Fatal("SETTINGS_ENCRYPTION_KEY must be 32 bytes, base64 encoded (openssl rand -base64 32)")
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET or SETTINGS_ENCRYPTION_KEY is required to encrypt console settings")
	}
	k := sha256.Sum256([]byte("birdseye-console-settings:" + secret))
	return k[:]
}

func encrypt(plain string) (string, error) {
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(aead.Seal(nonce, nonce, []byte(plain), nil)), nil
}

func decrypt(enc string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil || len(raw) < aead.NonceSize() {
		return "", errors.New("corrupt value")
	}
	plain, err := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], nil)
	if err != nil {
		return "", errors.New("can't decrypt (was the encryption key changed?)")
	}
	return string(plain), nil
}

// Load sets up encryption and reads saved values; call after the database is migrated
func Load() {
	block, err := aes.NewCipher(cipherKey())
	if err != nil {
		log.Fatal(err)
	}
	if aead, err = cipher.NewGCM(block); err != nil {
		log.Fatal(err)
	}
	var rows []models.AppSetting
	if err := db.DB.Find(&rows).Error; err != nil {
		log.Printf("Warning: can't read console settings: %v", err)
		return
	}
	mu.Lock()
	defer mu.Unlock()
	for _, r := range rows {
		v, err := decrypt(r.Value)
		if err != nil {
			log.Printf("Warning: console setting %s ignored: %v; the environment value is used instead", r.Key, err)
			continue
		}
		values[r.Key], updated[r.Key] = v, r.UpdatedAt
	}
	if len(rows) > 0 {
		log.Printf("Loaded %d console settings", len(values))
	}
}

// Save stores set (key -> new value) and removes clear (back to the environment value), then reloads
// the affected integrations. Unknown keys are refused.
func Save(set map[string]string, clear []string, adminID uint) error {
	groups := map[string]bool{}
	for k, v := range set {
		f, g := FieldByKey(k)
		if f == nil {
			return fmt.Errorf("unknown setting %s", k)
		}
		if f.Multiline {
			set[k] = strings.TrimSpace(v)
		} else {
			set[k] = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(v, "\n", ""), "\r", ""))
		}
		if set[k] == "" {
			return fmt.Errorf("%s is empty: use clear to remove it", f.Label)
		}
		groups[g.ID] = true
	}
	for _, k := range clear {
		f, g := FieldByKey(k)
		if f == nil {
			return fmt.Errorf("unknown setting %s", k)
		}
		groups[g.ID] = true
	}
	if err := validate(set); err != nil {
		return err
	}

	now := time.Now()
	tx := db.DB.Begin()
	for k, v := range set {
		enc, err := encrypt(v)
		if err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Save(&models.AppSetting{Key: k, Value: enc, UpdatedBy: adminID, UpdatedAt: now}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	if len(clear) > 0 {
		if err := tx.Where("`key` IN ?", clear).Delete(&models.AppSetting{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}

	mu.Lock()
	for k, v := range set {
		values[k], updated[k] = v, now
	}
	for _, k := range clear {
		delete(values, k)
		delete(updated, k)
	}
	var fns []func()
	for g := range groups {
		fns = append(fns, reloads[g]...)
	}
	mu.Unlock()
	for _, fn := range fns {
		fn()
	}
	return nil
}

// validate catches obvious paste mistakes before they break an integration
func validate(set map[string]string) error {
	for k, v := range set {
		switch k {
		case "RESEND_API_KEY":
			if !strings.HasPrefix(v, "re_") {
				return errors.New("a Resend API key starts with re_")
			}
		case "VAPID_SUBSCRIBER":
			if !strings.HasPrefix(v, "mailto:") && !strings.HasPrefix(v, "https://") {
				return errors.New("the push contact must start with mailto: or https://")
			}
		case "FIREBASE_SERVICE_ACCOUNT_JSON":
			if !strings.Contains(v, `"private_key"`) || !strings.Contains(v, `"client_email"`) {
				return errors.New("that doesn't look like a Firebase service account file (no private_key/client_email)")
			}
		case "GOOGLE_REDIRECT_URL", "DYNAPAY_API_URL":
			if !strings.HasPrefix(v, "https://") && !strings.HasPrefix(v, "http://localhost") && !strings.HasPrefix(v, "http://127.0.0.1") {
				return errors.New("URLs must use https://")
			}
		}
	}
	return nil
}

// Preview is what the console may show of a value: secrets only reveal their last 4 characters
func Preview(f Field, v string) string {
	if v == "" {
		return ""
	}
	if !f.Secret {
		return v
	}
	if len(v) <= 8 {
		return "••••"
	}
	return "••••" + v[len(v)-4:]
}
