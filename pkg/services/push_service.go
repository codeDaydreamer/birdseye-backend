package services

import (
    "encoding/json"
    "log"
    "net/http"

    "birdseye-backend/pkg/models"
    "github.com/SherClockHolmes/webpush-go"
    "gorm.io/gorm"
)

type PushService struct {
    db *gorm.DB
}

func NewPushService(db *gorm.DB) *PushService {
    return &PushService{db: db}
}

// Save or update a push subscription
func (s *PushService) SaveSubscription(sub *models.PushSubscription) error {
    var existing models.PushSubscription
    if err := s.db.Where("user_id = ?", sub.UserID).First(&existing).Error; err == nil {
        // Update existing subscription
        existing.Endpoint = sub.Endpoint
        existing.P256DH = sub.P256DH
        existing.Auth = sub.Auth
        return s.db.Save(&existing).Error
    }

    // Save new subscription
    return s.db.Create(sub).Error
}

// WebPush sends Web Push notifications to users' registered devices
type WebPush struct {
    db         *gorm.DB
    publicKey  string
    privateKey string
    subscriber string
}

func NewWebPush(db *gorm.DB, publicKey, privateKey, subscriber string) *WebPush {
    return &WebPush{db: db, publicKey: publicKey, privateKey: privateKey, subscriber: subscriber}
}

// SendToUser pushes a notification to every device the user has subscribed
func (w *WebPush) SendToUser(userID uint, title, body, url string) {
    w.sendToUsers([]uint{userID}, title, body, url)
}

// SendToFarm pushes a notification to every device of the farm's active members
func (w *WebPush) SendToFarm(farmID uint, title, body, url string) {
    var userIDs []uint
    if err := w.db.Model(&models.FarmMember{}).
        Where("farm_id = ? AND status = ?", farmID, "active").
        Pluck("user_id", &userIDs).Error; err != nil {
        log.Printf("Push: failed to load members of farm %d: %v", farmID, err)
        return
    }
    w.sendToUsers(userIDs, title, body, url)
}

func (w *WebPush) sendToUsers(userIDs []uint, title, body, url string) {
    if len(userIDs) == 0 {
        return
    }

    var subs []models.PushSubscription
    if err := w.db.Where("user_id IN ?", userIDs).Find(&subs).Error; err != nil {
        log.Printf("Push: failed to load subscriptions: %v", err)
        return
    }

    // Matches the payload public/sw.js reads in its 'push' handler
    payload, _ := json.Marshal(map[string]string{"title": title, "body": body, "url": url})

    for _, sub := range subs {
        resp, err := webpush.SendNotification(payload, &webpush.Subscription{
            Endpoint: sub.Endpoint,
            Keys:     webpush.Keys{P256dh: sub.P256DH, Auth: sub.Auth},
        }, &webpush.Options{
            Subscriber:      w.subscriber,
            VAPIDPublicKey:  w.publicKey,
            VAPIDPrivateKey: w.privateKey,
            TTL:             86400,
        })
        if err != nil {
            log.Printf("Push: send to subscription %d failed: %v", sub.ID, err)
            continue
        }
        resp.Body.Close()

        // The browser dropped this subscription; stop sending to it
        if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
            w.db.Delete(&sub)
        }
    }
}
