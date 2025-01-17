package broadcast

import (
	"github.com/gorilla/websocket"
	"log"
)

var clients = make(map[*websocket.Conn]bool) // Track connected clients

// WebSocket handler for managing connections
func HandleWebSocket(conn *websocket.Conn) {
	clients[conn] = true
	log.Println("New WebSocket client connected")

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			delete(clients, conn)
			break
		}
		// Broadcast the message to all connected clients
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Println("Error sending message to client:", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

// Broadcast function for broadcasting a message to all clients
func Broadcast(message string) {
	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Println("Error broadcasting message:", err)
			client.Close()
			delete(clients, client)
		}
	}
}
