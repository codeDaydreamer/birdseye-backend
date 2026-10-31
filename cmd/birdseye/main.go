package main

import (
	"context"
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

// loadVAPIDKeys enables Web Push. The key pair must be fixed in the environment: the client is built
// with the matching public key (VITE_VAPID_PUBLIC_KEY), so keys generated at boot could never work.
func loadVAPIDKeys() {
	vapidPublicKey = os.Getenv("VAPID_PUBLIC_KEY")
	vapidPrivateKey = os.Getenv("VAPID_PRIVATE_KEY")

	if vapidPublicKey == "" || vapidPrivateKey == "" {
		log.Println("Warning: VAPID_PUBLIC_KEY/VAPID_PRIVATE_KEY not set, Web Push disabled")
		return
	}

	subscriber := os.Getenv("VAPID_SUBSCRIBER") // e.g. mailto:support@birdseye-poultry.com
	if subscriber == "" {
		subscriber = "mailto:support@birdseye-poultry.com"
	}

	push := services.NewWebPush(db.DB, vapidPublicKey, vapidPrivateKey, subscriber)
	broadcast.PushToUser = push.SendToUser
	broadcast.PushToFarm = push.SendToFarm
}

// loadFCM enables push to the mobile app through Firebase, alongside Web Push. The service account comes from
// FIREBASE_SERVICE_ACCOUNT_FILE (a path to the JSON key) or FIREBASE_SERVICE_ACCOUNT_JSON (its contents).
func loadFCM() {
	key := []byte(os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON"))
	if path := os.Getenv("FIREBASE_SERVICE_ACCOUNT_FILE"); len(key) == 0 && path != "" {
		var err error
		if key, err = os.ReadFile(path); err != nil {
			log.Printf("Warning: can't read FIREBASE_SERVICE_ACCOUNT_FILE: %v, mobile push disabled", err)
			return
		}
	}
	if len(key) == 0 {
		log.Println("Warning: FIREBASE_SERVICE_ACCOUNT_FILE not set, mobile push disabled")
		return
	}
	fcm, err := services.NewFCM(db.DB, key)
	if err != nil {
		log.Printf("Warning: %v, mobile push disabled", err)
		return
	}

	// Send to phones as well as to whatever loadVAPIDKeys set up for browsers
	webToUser, webToFarm := broadcast.PushToUser, broadcast.PushToFarm
	broadcast.PushToUser = func(userID uint, title, body, url string) {
		if webToUser != nil {
			webToUser(userID, title, body, url)
		}
		fcm.SendToUser(userID, title, body, url)
	}
	broadcast.PushToFarm = func(farmID uint, title, body, url string) {
		if webToFarm != nil {
			webToFarm(farmID, title, body, url)
		}
		fcm.SendToFarm(farmID, title, body, url)
	}
	log.Println("Mobile push (Firebase Cloud Messaging) enabled")
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

	userID, err := middlewares.ParseUserToken(token)
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

	// Farm comes from ?farm_id= (browsers can't set headers on a WS upgrade), else the user's only farm
	member, err := middlewares.ResolveFarmMember(userID, c.Query("farm_id"))
	if err != nil {
		switch {
		case errors.Is(err, middlewares.ErrFarmRequired):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, middlewares.ErrNotFarmMember):
			c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this farm"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve farm"})
		}
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}

	broadcast.HandleWebSocket(conn, userID, member.FarmID)
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
				err := vaccinationService.SendVaccinationReminder(&vaccination)
				if err != nil {
					log.Printf("Error sending reminder for vaccination %d: %v", vaccination.ID, err)
					continue
				}

				// Email the farm's owners and managers
				managers, err := services.GetFarmManagers(vaccination.FarmID)
				if err != nil {
					log.Printf("Error fetching farm managers for vaccination %d: %v", vaccination.ID, err)
					continue
				}

				for _, user := range managers {
					err = email.SendVaccinationReminderEmail(user.Email, user.Username, &vaccination)
					if err != nil {
						log.Printf("Error sending email for vaccination %d to user %d: %v", vaccination.ID, user.ID, err)
					} else {
						log.Printf("Vaccination reminder email sent for vaccination %d to user %d", vaccination.ID, user.ID)
					}
				}
			} else {
				log.Printf("Vaccination %d is no longer within the reminder window", vaccination.ID)
			}
		}
	}
}


func main() {
	
	gin.SetMode(gin.ReleaseMode) 
	// Load environment variables
	loadEnv()

	

	// Initialize Google OAuth config
	services.InitGoogleOAuth()

	services.InitDynapayClient()

	// Initialize the database
	db.InitializeDB()

	// Auto-migrate all models
	err := db.DB.AutoMigrate(
		// Farm first: it declares the farm_id FK on subscriptions/billing_infos
		&models.Farm{},
		&models.FarmMember{},
		&models.FarmInvite{},
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
		&models.DeviceToken{},
		&models.BirdMovement{},
		&models.Incubator{},
		&models.EggSetting{},
		&models.ChickBooking{},
		&models.FeedLog{},
		&models.WeighIn{},
		&models.Contact{},
		&models.Notification{},
		&models.Budget{},
		&models.Admin{},
		&models.Plan{},
		&models.PromoCode{},
		&models.PlatformSettings{},
		&models.Payment{},
		&models.EggAdjustment{},
	)
	if err != nil {
		log.Fatalf("Error during auto migration: %v", err)
	}
	log.Println("Database migrated successfully")
	
	
	// Load VAPID keys (needs the DB for push subscriptions)
	loadVAPIDKeys()
	loadFCM()

	// Initialize authentication middleware
	middlewares.InitAuthMiddleware()

	// Initialize Gin router
	router := gin.Default()


	// Enable CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173","http://localhost:3000", "https://app.birdseye-poultry.com", "https://www.app.birdseye-poultry.com","https://www.console.birdseye-poultry.com","https://console.birdseye-poultry.com", "https://birdseye-poultry.com", "https://www.birdseye-poultry.com"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Farm-ID"},
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
	api.SetupHatcheryRoutes(router)
	api.SetupContactRoutes(router)
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
	api.SetupEggAdjustmentRoutes(router)
	api.SetupStaffRoutes(router)
	api.SetupPlanRoutes(router)


	// WebSocket routes
	router.GET("/ws", handleWebSocket)
	router.GET("/wss", handleWebSocket)

	router.POST("/api/push/subscribe", middlewares.UserAuth(), api.HandlePushSubscription)
	router.POST("/push/device", middlewares.UserAuth(), api.HandleRegisterDevice)
	router.DELETE("/push/device", middlewares.UserAuth(), api.HandleUnregisterDevice)


	// Start WebSocket broadcasting
	go broadcast.BroadcastMessages()

	// Start vaccination reminder background task
	vaccinationService := services.NewVaccinationService(db.DB)
	go startVaccinationReminderTask(vaccinationService)
	// Settles M-Pesa payments whose Dynapay webhook never arrived
	go services.RunDynapayReconciler(context.Background(), api.NotifyPaymentOutcome)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

		fmt.Printf("🚀 Starting Birdseye Backend on port %s...\n", port)

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


