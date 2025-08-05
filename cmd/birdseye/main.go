package main

import (
	"fmt"
	"log"
	"os"
	"errors"
	"time"
	"path/filepath"

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
	"birdseye-backend/pkg/services/email"
)

var (
	vapidPublicKey  string
	vapidPrivateKey string
)

func loadEnv() {
	// Get absolute path for the .env file
	envPath, _ := filepath.Abs("cmd/birdseye/.env")

	

	err := godotenv.Load(envPath)
	if err != nil {
		log.Println("Warning: No .env file found, using system environment variables.")
	}
}

func generateVAPIDKeys() {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		log.Fatalf("Error generating VAPID keys: %v", err)
	}

	

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

	
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // ⚠️ Allow all origins for WebSocket
	},
}

func handleWebSocket(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		token = c.Query("token")
	}

	if token == "" {
		log.Println("Unauthorized WebSocket connection attempt: Missing token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
		return
	}

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

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}

	broadcast.HandleWebSocket(conn, user.ID)
}

// Uptime tracking
var serverStartTime = time.Now()

func getUptime() string {
	duration := time.Since(serverStartTime)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	return fmt.Sprintf("%02dh:%02dm:%02ds", hours, minutes, seconds)
}

func checkDBConnection() bool {
	sqlDB, err := db.DB.DB()
	if err != nil {
		log.Println("Failed to get DB connection:", err)
		return false
	}

	err = sqlDB.Ping()
	if err != nil {
		log.Println("Database ping failed:", err)
		return false
	}

	return true
}


func startVaccinationReminderTask(vaccinationService *services.VaccinationService) {
	ticker := time.NewTicker(48 * time.Hour) // Run every 48 hours
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		var vaccinations []models.Vaccination

		err := vaccinationService.DB.Where("date > ? AND date < ?", now, now.Add(14*24*time.Hour)).Find(&vaccinations).Error
		if err != nil {
			log.Printf("Error retrieving vaccinations: %v", err)
			continue
		}

		log.Println("Found vaccinations:", vaccinations)

		for _, vaccination := range vaccinations {
			if now.Before(vaccination.Date.Add(14 * 24 * time.Hour)) {
				err := vaccinationService.SendVaccinationReminder(&vaccination, vaccination.UserID)
				if err != nil {
					log.Printf("Error sending reminder for vaccination %d: %v", vaccination.ID, err)
					continue
				}

				// Get user info (assuming you have a method to fetch user by ID)
				user, err := services.GetUserByID(vaccination.UserID)
				if err != nil {
					log.Printf("Error fetching user for vaccination %d: %v", vaccination.ID, err)
					continue
				}

				// Send vaccination reminder email using email package function
				err = email.SendVaccinationReminderEmail(user.Email, user.Username, &vaccination)
				if err != nil {
					log.Printf("Error sending email for vaccination %d: %v", vaccination.ID, err)
				} else {
					log.Printf("Vaccination reminder email sent for vaccination %d", vaccination.ID)
				}
			} else {
				log.Printf("Vaccination %d is no longer within the reminder window", vaccination.ID)
			}
		}
	}
}


func main() {
	
	 //gin.SetMode(gin.ReleaseMode) 
	// Load environment variables
	loadEnv()

	// Load VAPID keys
	loadVAPIDKeys()

	

	// Initialize Google OAuth config
	services.InitGoogleOAuth()

	services.InitDynapayClient()

	// Initialize the database
	db.InitializeDB()

	// Auto-migrate all models
	err := db.DB.AutoMigrate(
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
		&models.Admin{},
		&models.Payment{},
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
		AllowOrigins:     []string{"http://localhost:5173", "https://app.birdseye-poultry.com"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	router.SetTrustedProxies(nil)

	// Serve static files
	router.Static("/birdseye_backend/uploads", "./uploads")
	router.Static("/pkg/reports/generated", "./pkg/reports/generated")

	// Set up API routes
	api.SetupInventoryRoutes(router)
	api.SetupRoutes(router)
	expenseService := &services.ExpenseService{DB: db.DB}
	api.SetupExpenseRoutes(router, expenseService)
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
	api.SetupStatsRoutes(router)
	api.RegisterPaymentRoutes(router)
	api.RegisterWebhookRoutes(router)


	// WebSocket routes
	router.GET("/ws", handleWebSocket)
	router.GET("/wss", handleWebSocket)

	router.POST("/api/push/subscribe", middlewares.AuthMiddleware(), api.HandlePushSubscription)


	// Start WebSocket broadcasting
	go broadcast.BroadcastMessages()

	// Start vaccination reminder background task
	vaccinationService := services.NewVaccinationService(db.DB)
	go startVaccinationReminderTask(vaccinationService)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

		fmt.Printf("Starting Birdseye Backend on port %s...\n", port)

	// Health check endpoint
	router.GET("/ping", func(c *gin.Context) {
	dbHealthy := checkDBConnection()

	status := "ok"
	if !dbHealthy {
		status = "degraded"
	}
	c.JSON(http.StatusOK, gin.H{
		"status": status,
		"uptime": getUptime(),
		"database": gin.H{
			"connected": dbHealthy,
		},
		"google_oauth": gin.H{
			"client_id_loaded": os.Getenv("GOOGLE_CLIENT_ID") != "",
		},
	})

})


	// 🚀 Start the server
	log.Fatal(router.Run(":" + port))
}


