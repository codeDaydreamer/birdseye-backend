package main

import (
	"fmt"
	"log"
	"os"
	"errors"

	"birdseye-backend/pkg/api"
	"birdseye-backend/pkg/broadcast"
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services"

	"net/http"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

var (
	vapidPublicKey  string
	vapidPrivateKey string
)

func generateVAPIDKeys() {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		log.Fatalf("Error generating VAPID keys: %v", err)
	}

	fmt.Println("Generated VAPID Public Key:", publicKey)
	fmt.Println("Generated VAPID Private Key:", privateKey)

	// Save these keys as environment variables
	os.Setenv("VAPID_PUBLIC_KEY", publicKey)
	os.Setenv("VAPID_PRIVATE_KEY", privateKey)
}

func loadVAPIDKeys() {
	vapidPublicKey = os.Getenv("VAPID_PUBLIC_KEY")
	vapidPrivateKey = os.Getenv("VAPID_PRIVATE_KEY")

	if vapidPublicKey == "" || vapidPrivateKey == "" {
		log.Println("VAPID keys not found. Generating new keys...")
		generateVAPIDKeys()
		vapidPublicKey = os.Getenv("VAPID_PUBLIC_KEY")
		vapidPrivateKey = os.Getenv("VAPID_PRIVATE_KEY")
	}

	fmt.Println("Using VAPID Public Key:", vapidPublicKey)
	fmt.Println("Using VAPID Private Key:", vapidPrivateKey)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // ⚠️ Allow all origins for WebSocket
	},
}
func handleWebSocket(c *gin.Context) {
	// Extract token from query or headers
	token := c.GetHeader("Authorization")
	if token == "" {
		token = c.Query("token") // Allow token via query param as a fallback
	}

	if token == "" {
		log.Println("Unauthorized WebSocket connection attempt: Missing token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
		return
	}

	// Get the user from the token
	user, err := middlewares.GetUserFromToken(token)
	if err != nil {
		if errors.Is(err, middlewares.ErrTokenExpired) {
			log.Println("WebSocket connection attempt with expired token")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expired, please log in again"})
		} else {
			log.Println("Invalid token in WebSocket connection:", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		}
		return
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}

	// Pass user ID to WebSocket handler
	broadcast.HandleWebSocket(conn, user.ID)
}

func main() {
	// Load environment variables
	err := godotenv.Load("/home/palaski-jr/birdseye-backend/.env")
	if err != nil {
		log.Println("Warning: No .env file found, using system environment variables.")
	}

	// Load VAPID keys
	loadVAPIDKeys()

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
		&models.FlocksFinancialData{},
		&models.PushSubscription{},
		&models.Notification{},
		&models.Budget{},
	)
	if err != nil {
		log.Fatalf("Error during auto migration: %v", err)
	}
	log.Println("Database migrated successfully")

	// Initialize authentication middleware
	middlewares.InitAuthMiddleware()

	// Initialize Gin router
	router := gin.Default()

	// Enable CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "https://birdseye-client.vercel.app"},
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
	expenseService := &services.ExpenseService{DB: db.DB} // Ensure DB is assigned
	api.SetupExpenseRoutes(router, expenseService)       // Pass it to the function
	api.SetupSalesRoutes(router)
	api.SetupEggProductionRoutes(router)
	api.SetupFlockRoutes(router)
	api.SetupVaccinationRoutes(router)
	api.SetupBillingRoutes(router)
	api.SetupSubscriptionRoutes(router)
	api.SetupReportsRoutes(router)
	api.SetupFlockFinancialRoutes(router)
	api.SetupNotificationRoutes(router)
	api.SetupBudgetRoutes(router)

	// WebSocket route
	router.GET("/ws", handleWebSocket)
	// WebSocket route
	router.GET("/wss", handleWebSocket)

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
