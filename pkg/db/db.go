package db

import (
    "database/sql"
    "fmt"
    "log"
    "os"

    _ "github.com/go-sql-driver/mysql" // MySQL driver
)

// DB will hold the database connection
var DB *sql.DB

// InitializeDB initializes the MySQL database connection
func InitializeDB() {
    // Get environment variables for database connection
    dbHost := os.Getenv("DB_HOST")
    dbUser := os.Getenv("DB_USER")
    dbPassword := os.Getenv("DB_PASSWORD")
    dbName := os.Getenv("DB_NAME")

    // MySQL connection string: user:password@tcp(host:port)/dbname
    connectionString := fmt.Sprintf("%s:%s@tcp(%s)/%s", dbUser, dbPassword, dbHost, dbName)

    var err error
    DB, err = sql.Open("mysql", connectionString)
    if err != nil {
        log.Fatal("Failed to open database:", err)
    }

    err = DB.Ping()
    if err != nil {
        log.Fatal("Failed to connect to the database:", err)
    }
    fmt.Println("Successfully connected to MySQL!")
}

// CloseDB closes the database connection
func CloseDB() {
    DB.Close()
}
