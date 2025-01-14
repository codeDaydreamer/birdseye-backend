package main

import (
	"fmt"
	"log"
	"os"
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/api" // Import the api package
	"github.com/joho/godotenv"
	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"
	"github.com/gorilla/websocket"
	"net/http"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins (or modify based on your requirements)
		return true
	},
}

var clients = make(map[*websocket.Conn]bool) // Track connected clients

// WebSocket handler for managing connections
func handleWebSocket(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}
	defer conn.Close()

	// Register the new client
	clients[conn] = true
	log.Println("New WebSocket client connected")

	// Handle incoming messages
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

// Broadcast function (optional: use this to push real-time updates)
func broadcast(message string) {
	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Println("Error broadcasting message:", err)
			client.Close()
			delete(clients, client)
		}
	}
}

func init() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	// Initialize the database connection
	db.InitializeDB()

	// Initialize Gin router
	router := gin.Default()

	// Enable CORS middleware globally for all routes using gin-contrib/cors
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"}, // replace with your frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Set up routes
	api.SetupRoutes(router) // Use SetupRoutes function from api package

	// WebSocket route for handling real-time communication
	router.GET("/ws", handleWebSocket)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting Birdseye Backend on port %s...\n", port)
	log.Fatal(router.Run(":" + port)) // Start Gin server here
}
