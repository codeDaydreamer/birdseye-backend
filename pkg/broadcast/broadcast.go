package broadcast

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	clients   = make(map[*websocket.Conn]client) // Connected clients mapped to their user + farm
	broadcast = make(chan []byte)               // Channel for broadcasting messages
	mutex     = sync.Mutex{}                     // Mutex to protect concurrent access
)

// client identifies who is on a connection: updates fan out either to every connection of a
// farm (farm-scoped domains) or to one user (domains not yet migrated to farms)
type client struct {
	UserID uint
	FarmID uint
}

// Web Push senders, wired up in main when VAPID keys are configured. Notifications also go out as
// Web Push so users get them when the app isn't open.
var (
	PushToUser func(userID uint, title, body, url string)
	PushToFarm func(farmID uint, title, body, url string)
)

// UpdateMessage represents a generic WebSocket update message
type UpdateMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
	UserID uint      `json:"user_id,omitempty"`
	FarmID uint      `json:"farm_id,omitempty"`
}

// HandleWebSocket manages WebSocket connections
func HandleWebSocket(conn *websocket.Conn, userID, farmID uint) {
	mutex.Lock()
	clients[conn] = client{UserID: userID, FarmID: farmID}
	mutex.Unlock()

	log.Println("New WebSocket client connected for user:", userID, "farm:", farmID)

defer func() {
		mutex.Lock()
		delete(clients, conn)
		mutex.Unlock()
		conn.Close()
	}()

	// Listen for disconnects
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			log.Println("Client disconnected:", err)
			break
		}
	}
}

// BroadcastMessages continuously listens for messages to broadcast
func BroadcastMessages() {
	for {
		message := <-broadcast
		mutex.Lock()
		for client := range clients {
			if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Println("Error sending message to client:", err)
				client.Close()
				delete(clients, client)
			}
		}
		mutex.Unlock()
	}
}
// SendFlockUpdate broadcasts a flock update to every connected member of a farm
func SendFlockUpdate(farmID uint, eventType string, flock interface{}) {
	sendUpdateToFarm(farmID, eventType, "flock", flock)
}

// SendEggProductionUpdate broadcasts an egg production update to every connected member of a farm
func SendEggProductionUpdate(farmID uint, eventType string, record interface{}) {
	sendUpdateToFarm(farmID, eventType, "egg_production", record)
}

// SendEggAdjustmentUpdate broadcasts an egg adjustment update to every connected member of a farm
func SendEggAdjustmentUpdate(farmID uint, eventType string, adjustment interface{}) {
	sendUpdateToFarm(farmID, eventType, "egg_adjustment", adjustment)
}

// SendExpenseUpdate broadcasts an expense update to every connected member of a farm
func SendExpenseUpdate(farmID uint, eventType string, expense interface{}) {
	sendUpdateToFarm(farmID, eventType, "expense", expense)
}

// SendInventoryUpdate broadcasts an inventory update to every connected member of a farm
func SendInventoryUpdate(farmID uint, eventType string, inventoryItem interface{}) {
	sendUpdateToFarm(farmID, eventType, "inventory", inventoryItem)
}

// SendSaleUpdate broadcasts a sale update to every connected member of a farm
func SendSaleUpdate(farmID uint, eventType string, sale interface{}) {
	sendUpdateToFarm(farmID, eventType, "sale", sale)
}

// SendBudgetUpdate broadcasts a budget update to every connected member of a farm
func SendBudgetUpdate(farmID uint, eventType string, budget interface{}) {
	sendUpdateToFarm(farmID, eventType, "budget", budget)
}

// SendVaccinationUpdate broadcasts a vaccination update to every connected member of a farm
func SendVaccinationUpdate(farmID uint, eventType string, vaccination interface{}) {
	sendUpdateToFarm(farmID, eventType, "vaccination", vaccination)
}





// Generic function to send an update message to every connected member of a farm
func sendUpdateToFarm(farmID uint, eventType, category string, data interface{}) {
	if farmID == 0 {
		return
	}
	msg, err := json.Marshal(UpdateMessage{
		Type:   eventType,
		Data:   map[string]interface{}{"category": category, "payload": data},
		FarmID: farmID,
	})
	if err != nil {
		log.Println("Error marshalling WebSocket message:", err)
		return
	}

	sendTo(msg, func(cl client) bool { return cl.FarmID == farmID })
}

// sendTo writes msg to every connection matching the filter, dropping dead connections
func sendTo(msg []byte, match func(client) bool) {
	mutex.Lock()
	for conn, cl := range clients {
		if match(cl) {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Println("Error sending message to client:", err)
				conn.Close()
				delete(clients, conn)
			}
		}
	}
	mutex.Unlock()
}

// SendNotification sends a notification to a specific user
func SendNotification(userID uint, title, message, url string) {
	log.Println("Sending notification to user:", userID, title, message)
	msg, err := json.Marshal(UpdateMessage{
		Type: "notification",
		Data: map[string]interface{}{
			"title": title,
			"body":  message,
			"url":   url,
		},
		UserID: userID,
	})
	if err != nil {
		log.Println("Error marshalling notification message:", err)
		return
	}

	sendTo(msg, func(cl client) bool { return cl.UserID == userID })

	if PushToUser != nil {
		go PushToUser(userID, title, message, url)
	}
}

// SendFarmNotification sends a notification to every connected member of a farm
func SendFarmNotification(farmID uint, title, message, url string) {
	if farmID == 0 {
		return
	}
	log.Println("Sending notification to farm:", farmID, title, message)
	msg, err := json.Marshal(UpdateMessage{
		Type: "notification",
		Data: map[string]interface{}{
			"title": title,
			"body":  message,
			"url":   url,
		},
		FarmID: farmID,
	})
	if err != nil {
		log.Println("Error marshalling notification message:", err)
		return
	}

	sendTo(msg, func(cl client) bool { return cl.FarmID == farmID })

	if PushToFarm != nil {
		go PushToFarm(farmID, title, message, url)
	}
}

// DisconnectFarmMember closes a user's live connections to a farm, e.g. after they're removed from
// its staff, so they stop receiving its updates straight away
func DisconnectFarmMember(farmID, userID uint) {
	mutex.Lock()
	for conn, cl := range clients {
		if cl.FarmID == farmID && cl.UserID == userID {
			conn.Close()
			delete(clients, conn)
		}
	}
	mutex.Unlock()
}
