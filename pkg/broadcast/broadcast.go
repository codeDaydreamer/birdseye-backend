package broadcast

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	clients   = make(map[*websocket.Conn]uint) // Connected clients mapped to user IDs
	broadcast = make(chan []byte)               // Channel for broadcasting messages
	mutex     = sync.Mutex{}                     // Mutex to protect concurrent access
)

// UpdateMessage represents a generic WebSocket update message
type UpdateMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
	UserID uint      `json:"user_id,omitempty"`
}

// HandleWebSocket manages WebSocket connections
func HandleWebSocket(conn *websocket.Conn, userID uint) {
	mutex.Lock()
	clients[conn] = userID
	mutex.Unlock()

	log.Println("New WebSocket client connected for user:", userID)

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
// SendFlockUpdate broadcasts a flock update to a specific user
func SendFlockUpdate(userID uint, eventType string, flock interface{}) {
	sendUpdateToUser(userID, eventType, "flock", flock)
}
// SendEggProductionUpdate broadcasts an egg production update to a specific user
func SendEggProductionUpdate(userID uint, eventType string, record interface{}) {
	sendUpdateToUser(userID, eventType, "egg_production", record)
}
// SendExpenseUpdate broadcasts an expense update to a specific user
func SendExpenseUpdate(userID uint, eventType string, expense interface{}) {
	sendUpdateToUser(userID, eventType, "expense", expense)
}
// SendInventoryUpdate broadcasts an inventory update to a specific user
func SendInventoryUpdate(userID uint, eventType string, inventoryItem interface{}) {
	sendUpdateToUser(userID, eventType, "inventory", inventoryItem)
}
// SendSaleUpdate broadcasts a sale update to a specific user
func SendSaleUpdate(userID uint, eventType string, sale interface{}) {
	sendUpdateToUser(userID, eventType, "sale", sale)
}



// Generic function to send an update message to a specific user
func sendUpdateToUser(userID uint, eventType, category string, data interface{}) {
	msg, err := json.Marshal(UpdateMessage{
		Type:   eventType,
		Data:   map[string]interface{}{"category": category, "payload": data},
		UserID: userID,
	})
	if err != nil {
		log.Println("Error marshalling WebSocket message:", err)
		return
	}

	mutex.Lock()
	for client, uid := range clients {
		if uid == userID {
			if err := client.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Println("Error sending message to client:", err)
				client.Close()
				delete(clients, client)
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

	mutex.Lock()
	for client, uid := range clients {
		if uid == userID {
			if err := client.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Println("Error sending notification to client:", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
	mutex.Unlock()
}
