package db
import (
	"fmt"
	"log"
	"os"

	
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DB will hold the GORM database connection
var DB *gorm.DB

// InitializeDB initializes the GORM database connection
func InitializeDB() {
	

	// Get environment variables for database connection
	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	// MySQL connection string: user:password@tcp(host:port)/dbname
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", dbUser, dbPassword, dbHost, dbName)

	var err2 error
	DB, err2 = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err2 != nil {
		log.Fatal("Failed to connect to database:", err2)
	}

	fmt.Println("Successfully connected to MySQL!")
}

// CloseDB closes the GORM database connection
func CloseDB() {
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatal("Failed to get SQL DB instance:", err)
	}
	sqlDB.Close()
}
