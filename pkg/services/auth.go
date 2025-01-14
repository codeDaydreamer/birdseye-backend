package services

import (
    "fmt"
    "birdseye-backend/pkg/db"
    "birdseye-backend/pkg/models"
    "github.com/dgrijalva/jwt-go"
    "strings"
    "time"
    "errors"
)

var jwtSecret = []byte("your_secret_key_here") // Secret key for signing JWT

// RegisterUser registers a new user in the MySQL database and returns the user details
func RegisterUser(user *models.User) (*models.User, error) {
    // Hash the user's password
    err := user.HashPassword()
    if err != nil {
        return nil, fmt.Errorf("error hashing password: %v", err)
    }

    // Insert user into the database
    query := "INSERT INTO users (username, email, password) VALUES (?, ?, ?)"
    _, err = db.DB.Exec(query, user.Username, user.Email, user.Password)
    if err != nil {
        return nil, fmt.Errorf("error inserting user into database: %v", err)
    }
    
    // Return the user object after registration
    return user, nil
}

// LoginUser authenticates a user and returns a JWT and user details
func LoginUser(email, password string) (string, *models.User, error) {
    var user models.User

    // Query user by email
    query := "SELECT id, username, email, password FROM users WHERE email = ?"
    err := db.DB.QueryRow(query, email).Scan(&user.ID, &user.Username, &user.Email, &user.Password)
    if err != nil {
        return "", nil, fmt.Errorf("user not found: %v", err)
    }

    // Check if the password is correct
    if !user.CheckPassword(password) {
        return "", nil, fmt.Errorf("incorrect password")
    }

    // Generate JWT token
    token, err := generateJWT(user)
    if err != nil {
        return "", nil, fmt.Errorf("error generating token: %v", err)
    }

    return token, &user, nil
}

// generateJWT generates a JWT token for the user
func generateJWT(user models.User) (string, error) {
    // Create a new JWT token
    token := jwt.New(jwt.SigningMethodHS256)

    // Create claims (payload)
    claims := token.Claims.(jwt.MapClaims)
    claims["id"] = user.ID
    claims["username"] = user.Username
    claims["email"] = user.Email
    claims["exp"] = time.Now().Add(time.Hour * 24).Unix() // Expiration time (1 day)

    // Sign the token with the secret key
    tokenString, err := token.SignedString(jwtSecret)
    if err != nil {
        return "", fmt.Errorf("error signing the token: %v", err)
    }

    return tokenString, nil
}

// GetUserFromToken decodes the JWT token and retrieves the user
func GetUserFromToken(tokenString string) (*models.User, error) {
    // Remove "Bearer " prefix if it exists
    tokenString = strings.TrimPrefix(tokenString, "Bearer ")

    // Parse and validate the token
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        // Ensure the token is signed with the correct algorithm
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, errors.New("unexpected signing method")
        }
        return jwtSecret, nil
    })

    if err != nil || !token.Valid {
        return nil, errors.New("invalid token")
    }

    // Extract the user from the token
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return nil, errors.New("could not parse token claims")
    }

    // Create a user model from the claims
    user := &models.User{
        ID:       int(claims["id"].(float64)), // Type assertion to get the user ID
        Username: claims["username"].(string),
        Email:    claims["email"].(string),
    }

    return user, nil
}
// ChangePassword allows the user to change their password after verifying the current password
func ChangePassword(userID int, currentPassword, newPassword string) error {
    // Fetch the user from the database by ID
    var user models.User
    query := "SELECT id, username, email, password FROM users WHERE id = ?"
    err := db.DB.QueryRow(query, userID).Scan(&user.ID, &user.Username, &user.Email, &user.Password)
    if err != nil {
        return fmt.Errorf("user not found: %v", err)
    }

    // Check if the current password matches the stored password
    if !user.CheckPassword(currentPassword) {
        return fmt.Errorf("current password is incorrect")
    }

    // Hash the new password
    err = user.HashPasswordWithNewPassword(newPassword)
    if err != nil {
        return fmt.Errorf("error hashing new password: %v", err)
    }

    // Update the user's password in the database
    updateQuery := "UPDATE users SET password = ? WHERE id = ?"
    _, err = db.DB.Exec(updateQuery, user.Password, user.ID)
    if err != nil {
        return fmt.Errorf("error updating password in database: %v", err)
    }

    return nil
}
// UpdateUserProfile updates the user's profile information
func UpdateUserProfile(userID int, username, email, contact string) (*models.User, error) {
    // Update the user's profile in the database
    query := "UPDATE users SET username = ?, email = ?, contact = ? WHERE id = ?"
    _, err := db.DB.Exec(query, username, email, contact, userID)
    if err != nil {
        return nil, fmt.Errorf("error updating user profile: %v", err)
    }

    // Fetch the updated user
    var updatedUser models.User
    selectQuery := "SELECT id, username, email, contact FROM users WHERE id = ?"
    err = db.DB.QueryRow(selectQuery, userID).Scan(&updatedUser.ID, &updatedUser.Username, &updatedUser.Email, &updatedUser.Contact)
    if err != nil {
        return nil, fmt.Errorf("error retrieving updated user: %v", err)
    }

    return &updatedUser, nil
}
// UpdateUserProfilePicture updates the user's profile picture path
func UpdateUserProfilePicture(userID int, profilePicturePath string) (*models.User, error) {
    // Log the profile picture path for debugging
    fmt.Printf("Updating profile picture for user %d with path: %s\n", userID, profilePicturePath)

    // Ensure the file path is not empty or invalid
    if profilePicturePath == "" {
        return nil, fmt.Errorf("invalid profile picture path")
    }

    // Update the profile picture in the database
    query := "UPDATE users SET profile_picture = ? WHERE id = ?"
    result, err := db.DB.Exec(query, profilePicturePath, userID)
    if err != nil {
        // Log error details for debugging
        fmt.Printf("Error executing query: %v\n", err)
        return nil, fmt.Errorf("error updating profile picture: %v", err)
    }

    // Check if the update affected any rows
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        return nil, fmt.Errorf("no user found with ID %d", userID)
    }

    // Fetch the updated user
    var updatedUser models.User
    selectQuery := "SELECT id, username, email, contact, profile_picture FROM users WHERE id = ?"
    err = db.DB.QueryRow(selectQuery, userID).Scan(&updatedUser.ID, &updatedUser.Username, &updatedUser.Email, &updatedUser.Contact, &updatedUser.ProfilePicture)
    if err != nil {
        // Log error details for debugging
        fmt.Printf("Error retrieving updated user: %v\n", err)
        return nil, fmt.Errorf("error retrieving updated user: %v", err)
    }

    return &updatedUser, nil
}
