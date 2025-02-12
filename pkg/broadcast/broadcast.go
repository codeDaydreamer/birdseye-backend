package broadcast

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	clients   = make(map[*websocket.Conn]bool) // Connected clients
	broadcast = make(chan []byte)              // Channel for broadcasting messages
	mutex     = sync.Mutex{}                    // Mutex to protect concurrent access
)

// UpdateMessage represents a generic WebSocket update message
type UpdateMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// HandleWebSocket manages WebSocket connections
func HandleWebSocket(conn *websocket.Conn) {
	mutex.Lock()
	clients[conn] = true
	mutex.Unlock()

	log.Println("New WebSocket client connected")

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

// SendExpenseUpdate broadcasts an expense update message
func SendExpenseUpdate(eventType string, expense interface{}) {
	sendUpdate(eventType, "expense", expense)
}

// SendSaleUpdate broadcasts a sales update message
func SendSaleUpdate(eventType string, sale interface{}) {
	sendUpdate(eventType, "sale", sale)
}

// Generic function to send an update message
func sendUpdate(eventType, category string, data interface{}) {
	msg, err := json.Marshal(UpdateMessage{
		Type: eventType,
		Data: map[string]interface{}{
			"category": category,
			"payload":  data,
		},
	})
	if err != nil {
		log.Println("Error marshalling WebSocket message:", err)
		return
	}
	broadcast <- msg
}
