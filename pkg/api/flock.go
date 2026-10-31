package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/services"

	"github.com/gin-gonic/gin"
)

// FlockHandler handles batches of birds and their movement log
type FlockHandler struct {
	Service *services.FlockService
}

func SetupFlockRoutes(r *gin.Engine) {
	service := services.NewFlockService(db.DB, services.NewSalesService(db.DB), services.NewExpenseService(db.DB))
	handler := &FlockHandler{Service: service}

	flockRoutes := r.Group("/flocks").Use(middlewares.FarmAuth())
	{
		flockRoutes.GET("/", handler.GetFlocks)
		flockRoutes.GET("/:id", handler.GetFlock)
		flockRoutes.POST("/", handler.AddFlock)
		flockRoutes.PUT("/:id", handler.UpdateFlock)
		flockRoutes.DELETE("/:id", handler.DeleteFlock)
		flockRoutes.GET("/:id/movements", handler.GetMovements)
		flockRoutes.POST("/:id/movements", handler.AddMovement)
		flockRoutes.DELETE("/movements/:id", handler.DeleteMovement)
	}
}

// respondFlockError maps service errors to status codes; unknown ones are logged and hidden
func respondFlockError(c *gin.Context, err error) {
	if respondPlanLimit(c, err) {
		return
	}
	var notEnough *services.NotEnoughBirdsError
	switch {
	case errors.Is(err, services.ErrFlockNotFound), errors.Is(err, services.ErrMovementNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.As(err, &notEnough), errors.Is(err, services.ErrMovementLocked), errors.Is(err, services.ErrFlockClosed):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrFlockName), errors.Is(err, services.ErrFlockBreed),
		errors.Is(err, services.ErrProductionType), errors.Is(err, services.ErrFlockSource),
		errors.Is(err, services.ErrFlockBirds), errors.Is(err, services.ErrFlockAge),
		errors.Is(err, services.ErrFlockCost), errors.Is(err, services.ErrFlockPlacedInFuture),
		errors.Is(err, services.ErrFlockStatus), errors.Is(err, services.ErrInvalidMovement),
		errors.Is(err, services.ErrInvalidQuantity), errors.Is(err, services.ErrInvalidCause),
		errors.Is(err, services.ErrSameFlock), errors.Is(err, services.ErrBadDate):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		log.Printf("Flock request failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
	}
}

func idParam(c *gin.Context) (uint, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return 0, false
	}
	return uint(id), true
}

func (h *FlockHandler) GetFlocks(c *gin.Context) {
	farmID, _ := farmContext(c)
	flocks, err := h.Service.GetFlocksByFarm(farmID)
	if err != nil {
		respondFlockError(c, err)
		return
	}
	// Keeps /flock-financial current, as it always has been on this request
	for i := range flocks {
		h.Service.RefreshFinancials(&flocks[i])
	}
	c.JSON(http.StatusOK, flocks)
}

func (h *FlockHandler) GetFlock(c *gin.Context) {
	farmID, _ := farmContext(c)
	id, ok := idParam(c)
	if !ok {
		return
	}
	flock, err := h.Service.GetFlockByID(id, farmID)
	if err != nil {
		respondFlockError(c, err)
		return
	}
	c.JSON(http.StatusOK, flock)
}

// AddFlock starts a batch: body is services.FlockInput with "birds" set
func (h *FlockHandler) AddFlock(c *gin.Context) {
	farmID, userID := farmContext(c)
	var in services.FlockInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	flock, err := h.Service.CreateFlock(farmID, userID, in)
	if err != nil {
		respondFlockError(c, err)
		return
	}
	c.JSON(http.StatusCreated, flock)
}

// UpdateFlock changes a batch's details (not its counts)
func (h *FlockHandler) UpdateFlock(c *gin.Context) {
	farmID, _ := farmContext(c)
	id, ok := idParam(c)
	if !ok {
		return
	}
	var in services.FlockInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	flock, err := h.Service.UpdateFlock(farmID, id, in)
	if err != nil {
		respondFlockError(c, err)
		return
	}
	c.JSON(http.StatusOK, flock)
}

func (h *FlockHandler) DeleteFlock(c *gin.Context) {
	farmID, _ := farmContext(c)
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.Service.DeleteFlock(id, farmID); err != nil {
		respondFlockError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Flock deleted successfully"})
}

func (h *FlockHandler) GetMovements(c *gin.Context) {
	farmID, _ := farmContext(c)
	id, ok := idParam(c)
	if !ok {
		return
	}
	moves, err := h.Service.GetMovements(id, farmID)
	if err != nil {
		respondFlockError(c, err)
		return
	}
	c.JSON(http.StatusOK, moves)
}

// AddMovement records deaths, culls, a recount or a transfer: body is services.MovementInput
func (h *FlockHandler) AddMovement(c *gin.Context) {
	farmID, userID := farmContext(c)
	id, ok := idParam(c)
	if !ok {
		return
	}
	var in services.MovementInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if err := h.Service.RecordMovement(farmID, userID, id, in); err != nil {
		respondFlockError(c, err)
		return
	}
	flock, _ := h.Service.GetFlockByID(id, farmID)
	c.JSON(http.StatusCreated, flock)
}

func (h *FlockHandler) DeleteMovement(c *gin.Context) {
	farmID, _ := farmContext(c)
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.Service.UndoMovement(farmID, id); err != nil {
		respondFlockError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Movement removed"})
}
