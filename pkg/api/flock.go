package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services"
	"birdseye-backend/pkg/middlewares"
	"net/http"
	"strconv"
    "time"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"
)

// FlockHandler handles flock-related requests
type FlockHandler struct {
    Service *services.FlockService
}

func SetupFlockRoutes(r *gin.Engine) {
    eggService := services.NewEggProductionService(db.DB) // Initialize EggProductionService
    salesService := services.NewSalesService(db.DB)       // Initialize SalesService
    expenseService := services.NewExpenseService(db.DB)   // ✅ Initialize ExpenseService

    flockService := services.NewFlockService(db.DB, eggService, salesService, expenseService) // ✅ Pass ExpenseService
    handler := &FlockHandler{Service: flockService}

    flockRoutes := r.Group("/flocks").Use(middlewares.FarmAuth())
    {
        flockRoutes.GET("/", handler.GetFlocks)
        flockRoutes.GET("/:id", handler.GetFlock)
        flockRoutes.POST("/", handler.AddFlock)
        flockRoutes.PUT("/:id", handler.UpdateFlock)
        flockRoutes.DELETE("/:id", handler.DeleteFlock)
    }
}

func (h *FlockHandler) GetFlocks(c *gin.Context) {
    farmID, _ := farmContext(c)

    flocks, err := h.Service.GetFlocksByFarm(farmID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve flocks"})
        return
    }

    // Define date range for financial calculations (current month)
    start := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
    end := start.AddDate(0, 1, -1) // Last day of the month

    // Recalculate and update metrics before returning the data
    for i := range flocks {
        h.Service.CalculateFlockMetrics(&flocks[i], start, end)
    }

    c.JSON(http.StatusOK, flocks)
}

func (h *FlockHandler) GetFlock(c *gin.Context) {
    farmID, _ := farmContext(c)

    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
        return
    }

    flock, err := h.Service.GetFlockByID(uint(id), farmID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Flock not found"})
        return
    }

    // Define date range for financial calculations (current month)
    start := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
    end := start.AddDate(0, 1, -1) // Last day of the month

    // Recalculate and update metrics before returning the data
    h.Service.CalculateFlockMetrics(flock, start, end)

    c.JSON(http.StatusOK, flock)
}

// AddFlock adds a new flock to the farm
func (h *FlockHandler) AddFlock(c *gin.Context) {
    farmID, _ := farmContext(c)

    var flock models.Flock
    if err := c.ShouldBindJSON(&flock); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    flock.ID = 0
    flock.FarmID = farmID

    if err := services.CheckFlockLimit(farmID); err != nil {
        if !respondPlanLimit(c, err) {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check plan limits"})
        }
        return
    }
    if err := services.CheckBirdLimit(farmID, flock.BirdCount, 0); err != nil {
        if !respondPlanLimit(c, err) {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check plan limits"})
        }
        return
    }

    if err := db.DB.Omit(clause.Associations).Create(&flock).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create flock"})
        return
    }

    c.JSON(http.StatusCreated, flock)
}

func (h *FlockHandler) UpdateFlock(c *gin.Context) {
    farmID, _ := farmContext(c)

    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
        return
    }

    flock, err := h.Service.GetFlockByID(uint(id), farmID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Flock not found"})
        return
    }
    farm := flock.FarmID

    // Bind JSON data to the existing flock
    if err := c.ShouldBindJSON(&flock); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // The body can't move the flock to another record or farm
    flock.ID, flock.FarmID = uint(id), farm

    if err := services.CheckBirdLimit(farmID, flock.BirdCount, flock.ID); err != nil {
        if !respondPlanLimit(c, err) {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check plan limits"})
        }
        return
    }

    if err := h.Service.UpdateFlock(flock); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update flock"})
        return
    }

    c.JSON(http.StatusOK, flock)
}

// DeleteFlock deletes one of the farm's flocks
func (h *FlockHandler) DeleteFlock(c *gin.Context) {
    farmID, _ := farmContext(c)

    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
        return
    }

    // Load it first so the delete hook knows which farm to notify
    flock, err := h.Service.GetFlockByID(uint(id), farmID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Flock not found"})
        return
    }

    if err := db.DB.Delete(flock).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete flock"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Flock deleted successfully"})
}
