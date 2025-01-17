package api

import (
    "birdseye-backend/pkg/broadcast"
    "birdseye-backend/pkg/models"
    "birdseye-backend/pkg/services"
    "github.com/gin-gonic/gin"
    "net/http"
    "strconv"
    "strings"
    "github.com/dgrijalva/jwt-go"
    "fmt"
)

// ExtractUserID extracts the user ID from the authorization token
func ExtractUserID(c *gin.Context) (uint, error) {
    // Get the token from the Authorization header
    authHeader := c.GetHeader("Authorization")
    if authHeader == "" {
        return 0, fmt.Errorf("no authorization header provided")
    }

    // Extract the token from the header
    tokenString := strings.TrimPrefix(authHeader, "Bearer ")
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        // Ensure that the token is signed with the correct method (HMAC)
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method")
        }
        // Return the secret key
        return []byte("your-secret-key"), nil
    })

    if err != nil {
        return 0, err
    }

    // Extract user ID from token claims
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok || !token.Valid {
        return 0, fmt.Errorf("invalid token")
    }

    // Return user ID as uint
    userID := uint(claims["id"].(float64)) // Assuming the user ID is stored as a float64
    return userID, nil
}

func SetupInventoryRoutes(router *gin.Engine) {
    inventory := router.Group("/inventory")
    {
        // Fetch all items for the current user
        inventory.GET("/", func(c *gin.Context) {
            userID, err := ExtractUserID(c)
            if err != nil {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
                return
            }

            items, err := services.FetchAllItems(userID)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }
            c.JSON(http.StatusOK, gin.H{"items": items})
        })

        // Add new item for the current user
        inventory.POST("/", func(c *gin.Context) {
            var item models.InventoryItem
            if err := c.ShouldBindJSON(&item); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
                return
            }

            userID, err := ExtractUserID(c)
            if err != nil {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
                return
            }

            // Add the user ID to the item before saving
            item.UserID = userID

            addedItem, err := services.AddItem(userID, &item)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }

            // Broadcast the item addition
            broadcast.Broadcast("A new item has been added: " + addedItem.ItemName)

            c.JSON(http.StatusOK, gin.H{"item": addedItem})
        })

        // Edit item for the current user
        inventory.PUT("/:id", func(c *gin.Context) {
            var item models.InventoryItem
            if err := c.ShouldBindJSON(&item); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
                return
            }

            itemIDStr := c.Param("id")
            itemID, err := strconv.ParseUint(itemIDStr, 10, 32)
            if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
                return
            }

            userID, err := ExtractUserID(c)
            if err != nil {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
                return
            }

            item.ID = uint(itemID)
            item.UserID = userID // Ensure the user ID is associated with the item

            updatedItem, err := services.EditItem(userID, &item)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }

            // Broadcast the item update
            broadcast.Broadcast("Item updated: " + updatedItem.ItemName)

            c.JSON(http.StatusOK, gin.H{"item": updatedItem})
        })

        // Delete item for the current user
        inventory.DELETE("/:id", func(c *gin.Context) {
            itemIDStr := c.Param("id")
            itemID, err := strconv.ParseUint(itemIDStr, 10, 32)
            if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
                return
            }

            userID, err := ExtractUserID(c)
            if err != nil {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
                return
            }

            err = services.DeleteItem(userID, uint(itemID))
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }

            // Broadcast the item deletion
            broadcast.Broadcast("An item has been deleted.")

            c.JSON(http.StatusOK, gin.H{"message": "Item deleted"})
        })

        // Fetch a specific item for the current user
        inventory.GET("/:id", func(c *gin.Context) {
            itemIDStr := c.Param("id")
            itemID, err := strconv.ParseUint(itemIDStr, 10, 32)
            if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
                return
            }

            userID, err := ExtractUserID(c)
            if err != nil {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
                return
            }

            item, err := services.FetchItemByID(userID, uint(itemID))
            if err != nil {
                c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
                return
            }

            c.JSON(http.StatusOK, gin.H{"item": item})
        })

        // Update item quantity for the current user
        inventory.PUT("/:id/quantity", func(c *gin.Context) {
            var request struct {
                Quantity int `json:"quantity"`
            }

            if err := c.ShouldBindJSON(&request); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
                return
            }

            itemIDStr := c.Param("id")
            itemID, err := strconv.ParseUint(itemIDStr, 10, 32)
            if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
                return
            }

            userID, err := ExtractUserID(c)
            if err != nil {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
                return
            }

            updatedItem, err := services.UpdateItemQuantity(userID, uint(itemID), request.Quantity)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }

            // Broadcast the quantity update
            broadcast.Broadcast("Item quantity updated: " + updatedItem.ItemName)

            c.JSON(http.StatusOK, gin.H{"item": updatedItem})
        })
    }
}
