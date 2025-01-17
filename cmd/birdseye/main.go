package main

import (
	"fmt"
	"log"
	"os"
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/api"
	"birdseye-backend/pkg/models" // Import the models package
	"github.com/joho/godotenv"
	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"
	"github.com/gorilla/websocket"
	"net/http"
	"birdseye-backend/pkg/broadcast" // Import the new broadcast package
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin (you can improve this by checking origin if needed)
		return true
	},
}

func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}
	defer conn.Close()

	broadcast.HandleWebSocket(conn) // Handle the WebSocket connection
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	// Initialize the database connection
	db.InitializeDB()

	// AutoMigrate all models
	err := db.DB.AutoMigrate(
		&models.InventoryItem{}, // Add all models here
		// Add other models as needed, for example:
		// &models.AnotherModel{},
	)
	if err != nil {
		log.Fatalf("Error during auto migration: %v", err)
	}
	log.Println("Database migrated successfully")

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
	api.SetupInventoryRoutes(router) // Use SetupInventoryRoutes function from api package
	api.SetupRoutes(router)

	// WebSocket route for handling real-time communication
	router.GET("/ws", handleWebSocket)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default to port 8080 if no port is specified in the environment
	}

	fmt.Printf("Starting Birdseye Backend on port %s...\n", port)
	log.Fatal(router.Run(":" + port)) // Start Gin server here
}
