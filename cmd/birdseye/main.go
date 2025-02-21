package main

import (
	"fmt"
	"log"
	"os"

	"birdseye-backend/pkg/api"
	"birdseye-backend/pkg/broadcast"
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	

	
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // ⚠️ Allow all origins for WebSocket
	},
}

// Custom middleware to log incoming requests and CORS details
func logCORS(c *gin.Context) {
	origin := c.Request.Header.Get("Origin")
	method := c.Request.Method
	fmt.Printf("Incoming request: Method=%s, Origin=%s\n", method, origin)

	// Check if the origin is allowed
	if origin == "http://localhost:5173" {
		fmt.Println("Allowed origin:", origin)
	} else {
		fmt.Println("Disallowed origin:", origin)
	}

	c.Next() // Continue to the next handler
}

func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}
	broadcast.HandleWebSocket(conn)
}

func main() {
	// Load environment variables
	err := godotenv.Load("/home/palaski-jr/birdseye-backend/.env")
	if err != nil {
		log.Println("Warning: No .env file found, using system environment variables.")
	}

	// Initialize the database
	db.InitializeDB()

	// AutoMigrate all models
	err = db.DB.AutoMigrate(
		&models.Flock{},
		&models.User{},
		&models.EggProduction{},
		&models.InventoryItem{},
		&models.Expense{},
		&models.Sale{},
		&models.Vaccination{},
		&models.Subscription{},
		&models.BillingInfo{},
		&models.Report{},
		&models.FlockFinancialData{},
	)
	if err != nil {
		log.Fatalf("Error during auto migration: %v", err)
	}
	log.Println("Database migrated successfully")

	// Initialize authentication middleware
	middlewares.InitAuthMiddleware()

	// Initialize Gin router
	router := gin.Default()

	// Enable CORS middleware for WebSockets
	router.Use(logCORS) // Log the CORS details for every request

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Remove proxy restrictions
	router.SetTrustedProxies(nil)

	// Serve uploaded profile pictures
	router.Static("/birdseye_backend/uploads", "./uploads")

	// Serve generated reports (e.g., PDFs)
	router.Static("/pkg/reports/generated", "./pkg/reports/generated")


	// Set up routes
	api.SetupInventoryRoutes(router)
	api.SetupRoutes(router)
	api.SetupExpenseRoutes(router)
	api.SetupSalesRoutes(router)
	api.SetupEggProductionRoutes(router)
	api.SetupFlockRoutes(router)
	api.SetupVaccinationRoutes(router)
	api.SetupBillingRoutes(router)
	api.SetupSubscriptionRoutes(router)
	api.SetupReportsRoutes(router)
	api.SetupFinancialRoutes(router) 
	
	

	// WebSocket route
	router.GET("/ws", handleWebSocket)

	// Start WebSocket broadcasting
	go broadcast.BroadcastMessages()

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting Birdseye Backend on port %s...\n", port)
	log.Fatal(router.Run(":" + port))
}
