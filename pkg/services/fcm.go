package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"birdseye-backend/pkg/models"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

// FCM sends push notifications to the mobile app through Firebase Cloud Messaging (HTTP v1 API),
// authenticating with a Firebase service account
type FCM struct {
	db        *gorm.DB
	projectID string
	client    *http.Client
}

// NewFCM builds a sender from the service account JSON downloaded from the Firebase console
// (Project settings > Service accounts > Generate new private key)
func NewFCM(db *gorm.DB, serviceAccountJSON []byte) (*FCM, error) {
	creds, err := google.CredentialsFromJSON(context.Background(), serviceAccountJSON,
		"https://www.googleapis.com/auth/firebase.messaging")
	if err != nil {
		return nil, fmt.Errorf("reading Firebase service account: %w", err)
	}
	if creds.ProjectID == "" {
		return nil, fmt.Errorf("the Firebase service account has no project_id")
	}
	client := oauth2.NewClient(context.Background(), creds.TokenSource)
	client.Timeout = 15 * time.Second
	return &FCM{db: db, projectID: creds.ProjectID, client: client}, nil
}

// SaveDeviceToken records the phone's token for the user, taking it over from whoever had it before
func SaveDeviceToken(db *gorm.DB, userID uint, token, platform string) error {
	if platform == "" {
		platform = "android"
	}
	var existing models.DeviceToken
	err := db.Where("token = ?", token).First(&existing).Error
	if err == nil {
		return db.Model(&existing).Updates(map[string]interface{}{
			"user_id": userID, "platform": platform, "updated_at": time.Now(),
		}).Error
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	return db.Create(&models.DeviceToken{UserID: userID, Token: token, Platform: platform}).Error
}

// DeleteDeviceToken forgets the phone's token for the user (on sign-out)
func DeleteDeviceToken(db *gorm.DB, userID uint, token string) error {
	return db.Where("user_id = ? AND token = ?", userID, token).Delete(&models.DeviceToken{}).Error
}

// SendToUser pushes a notification to every phone the user is signed in on
func (f *FCM) SendToUser(userID uint, title, body, url string) {
	f.sendToUsers([]uint{userID}, title, body, url)
}

// SendToFarm pushes a notification to the phones of the farm's active members
func (f *FCM) SendToFarm(farmID uint, title, body, url string) {
	var userIDs []uint
	if err := f.db.Model(&models.FarmMember{}).
		Where("farm_id = ? AND status = ?", farmID, "active").
		Pluck("user_id", &userIDs).Error; err != nil {
		log.Printf("FCM: failed to load members of farm %d: %v", farmID, err)
		return
	}
	f.sendToUsers(userIDs, title, body, url)
}

func (f *FCM) sendToUsers(userIDs []uint, title, body, url string) {
	if len(userIDs) == 0 {
		return
	}
	var tokens []models.DeviceToken
	if err := f.db.Where("user_id IN ?", userIDs).Find(&tokens).Error; err != nil {
		log.Printf("FCM: failed to load device tokens: %v", err)
		return
	}
	for _, t := range tokens {
		if err := f.send(t.Token, title, body, url); err == errTokenGone {
			// The app was uninstalled or the token rotated; stop sending to it
			f.db.Delete(&models.DeviceToken{}, t.ID)
		} else if err != nil {
			log.Printf("FCM: send to user %d failed: %v", t.UserID, err)
		}
	}
}

var errTokenGone = fmt.Errorf("device token is no longer registered")

func (f *FCM) send(token, title, body, url string) error {
	// "notification" lets Android show it while the app is closed; "data" tells the app where to go on tap
	msg := map[string]interface{}{
		"message": map[string]interface{}{
			"token":        token,
			"notification": map[string]string{"title": title, "body": body},
			"data":         map[string]string{"title": title, "body": body, "url": url},
			"android": map[string]interface{}{
				"priority":     "high",
				"notification": map[string]string{"channel_id": "birdseye_alerts"},
			},
		},
	}
	payload, _ := json.Marshal(msg)
	endpoint := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", f.projectID)
	resp, err := f.client.Post(endpoint, "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	// 404 UNREGISTERED, or 400 for a malformed/foreign token: the token will never work again
	if resp.StatusCode == http.StatusNotFound ||
		(resp.StatusCode == http.StatusBadRequest && bytes.Contains(respBody, []byte("registration token"))) {
		return errTokenGone
	}
	return fmt.Errorf("FCM returned %d: %s", resp.StatusCode, respBody)
}
