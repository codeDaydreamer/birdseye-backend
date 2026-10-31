package api

import (
	"errors"
	"log"
	"net/http"

	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services"

	"github.com/gin-gonic/gin"
)

// SetupHatcheryRoutes registers the hatchery (incubators, egg settings, chick bookings) and the batch feed log,
// weigh-ins and performance
func SetupHatcheryRoutes(r *gin.Engine) {
	sales := services.NewSalesService(db.DB)
	h := &services.HatcheryService{DB: db.DB, Sales: sales,
		Flocks: services.NewFlockService(db.DB, sales, services.NewExpenseService(db.DB))}

	g := r.Group("/").Use(middlewares.FarmAuth())
	{
		g.GET("/incubators", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			list, err := h.ListIncubators(farmID)
			respond(c, list, err, http.StatusOK)
		})
		g.POST("/incubators", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			var inc models.Incubator
			if !bind(c, &inc) {
				return
			}
			inc.ID, inc.Active = 0, true
			respond(c, &inc, h.SaveIncubator(farmID, &inc), http.StatusCreated)
		})
		g.PUT("/incubators/:id", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			id, ok := idParam(c)
			var inc models.Incubator
			if !ok || !bind(c, &inc) {
				return
			}
			inc.ID = id
			respond(c, &inc, h.SaveIncubator(farmID, &inc), http.StatusOK)
		})
		g.DELETE("/incubators/:id", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			if id, ok := idParam(c); ok {
				respond(c, gin.H{"message": "Incubator removed"}, h.DeleteIncubator(farmID, id), http.StatusOK)
			}
		})

		g.GET("/egg-settings", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			list, err := h.ListSettings(farmID)
			respond(c, list, err, http.StatusOK)
		})
		g.POST("/egg-settings", func(c *gin.Context) {
			farmID, userID := farmContext(c)
			var in services.SettingInput
			if !bind(c, &in) {
				return
			}
			setting, err := h.CreateSetting(farmID, userID, in)
			respond(c, setting, err, http.StatusCreated)
		})
		g.POST("/egg-settings/:id/candle", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			id, ok := idParam(c)
			var in struct {
				Clear  int    `json:"clear"`
				Rotten int    `json:"rotten"`
				Date   string `json:"date"`
			}
			if !ok || !bind(c, &in) {
				return
			}
			setting, err := h.Candle(farmID, id, in.Clear, in.Rotten, in.Date)
			respond(c, setting, err, http.StatusOK)
		})
		g.POST("/egg-settings/:id/hatch", func(c *gin.Context) {
			farmID, userID := farmContext(c)
			id, ok := idParam(c)
			var in services.HatchInput
			if !ok || !bind(c, &in) {
				return
			}
			setting, err := h.Hatch(farmID, userID, id, in)
			respond(c, setting, err, http.StatusOK)
		})
		g.POST("/egg-settings/:id/close", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			if id, ok := idParam(c); ok {
				respond(c, gin.H{"message": "Setting closed"}, h.CloseSetting(farmID, id), http.StatusOK)
			}
		})
		g.DELETE("/egg-settings/:id", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			if id, ok := idParam(c); ok {
				respond(c, gin.H{"message": "Setting deleted"}, h.DeleteSetting(farmID, id), http.StatusOK)
			}
		})

		g.GET("/chick-bookings", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			list, err := h.ListBookings(farmID)
			respond(c, list, err, http.StatusOK)
		})
		g.POST("/chick-bookings", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			var b models.ChickBooking
			if !bind(c, &b) {
				return
			}
			b.ID = 0
			respond(c, &b, h.SaveBooking(farmID, &b), http.StatusCreated)
		})
		g.PUT("/chick-bookings/:id", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			id, ok := idParam(c)
			var b models.ChickBooking
			if !ok || !bind(c, &b) {
				return
			}
			b.ID = id
			respond(c, &b, h.SaveBooking(farmID, &b), http.StatusOK)
		})
		g.DELETE("/chick-bookings/:id", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			if id, ok := idParam(c); ok {
				respond(c, gin.H{"message": "Booking deleted"}, h.DeleteBooking(farmID, id), http.StatusOK)
			}
		})
		g.POST("/chick-bookings/:id/fulfil", func(c *gin.Context) {
			farmID, userID := farmContext(c)
			id, ok := idParam(c)
			var in struct {
				FlockID uint `json:"flock_id"`
			}
			if !ok || !bind(c, &in) {
				return
			}
			b, err := h.FulfilBooking(farmID, userID, id, in.FlockID)
			respond(c, b, err, http.StatusOK)
		})

		g.GET("/flocks/:id/feed", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			if id, ok := idParam(c); ok {
				list, err := services.FeedLogs(db.DB, farmID, id)
				respond(c, list, err, http.StatusOK)
			}
		})
		g.POST("/flocks/:id/feed", func(c *gin.Context) {
			farmID, userID := farmContext(c)
			id, ok := idParam(c)
			var f models.FeedLog
			if !ok || !bind(c, &f) {
				return
			}
			respond(c, &f, services.AddFeed(db.DB, farmID, userID, id, &f), http.StatusCreated)
		})
		g.DELETE("/flocks/feed/:id", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			if id, ok := idParam(c); ok {
				respond(c, gin.H{"message": "Feed record deleted"}, services.DeleteFeed(db.DB, farmID, id), http.StatusOK)
			}
		})
		g.GET("/flocks/:id/weigh-ins", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			if id, ok := idParam(c); ok {
				list, err := services.WeighIns(db.DB, farmID, id)
				respond(c, list, err, http.StatusOK)
			}
		})
		g.POST("/flocks/:id/weigh-ins", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			id, ok := idParam(c)
			var w models.WeighIn
			if !ok || !bind(c, &w) {
				return
			}
			respond(c, &w, services.AddWeighIn(db.DB, farmID, id, &w), http.StatusCreated)
		})
		g.DELETE("/flocks/weigh-ins/:id", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			if id, ok := idParam(c); ok {
				respond(c, gin.H{"message": "Weigh-in deleted"}, services.DeleteWeighIn(db.DB, farmID, id), http.StatusOK)
			}
		})
		g.GET("/flocks/:id/performance", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			if id, ok := idParam(c); ok {
				p, err := services.BatchPerformance(db.DB, farmID, id)
				respond(c, p, err, http.StatusOK)
			}
		})
	}
}

func bind(c *gin.Context, dest interface{}) bool {
	if err := c.ShouldBindJSON(dest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return false
	}
	return true
}

// respond sends the result, or maps the error: not found 404, not enough stock/room or already done 409,
// bad input 400, anything else 500 (logged)
func respond(c *gin.Context, body interface{}, err error, status int) {
	if err == nil {
		c.JSON(status, body)
		return
	}
	if respondPlanLimit(c, err) {
		return
	}
	var noEggs *services.NotEnoughEggsError
	var noBirds *services.NotEnoughBirdsError
	is := func(targets ...error) bool {
		for _, t := range targets {
			if errors.Is(err, t) {
				return true
			}
		}
		return false
	}
	switch {
	case is(services.ErrIncubatorNotFound, services.ErrSettingNotFound, services.ErrBookingNotFound,
		services.ErrFlockNotFound, services.ErrFeedNotFound, services.ErrWeighFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.As(err, &noEggs), errors.As(err, &noBirds),
		is(services.ErrIncubatorFull, services.ErrSettingDone, services.ErrSettingKept, services.ErrBookingDone):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case is(services.ErrIncubatorName, services.ErrSettingName, services.ErrSettingEggs, services.ErrSettingSource,
		services.ErrSettingFlock, services.ErrCandleCounts, services.ErrHatchCounts, services.ErrCullCounts,
		services.ErrHatchTarget, services.ErrBookingCustomer, services.ErrBookingQuantity, services.ErrSalePrice,
		services.ErrFeedType, services.ErrFeedKg, services.ErrWeighSample, services.ErrWeighWeight,
		services.ErrBadDate, services.ErrFlockName, services.ErrFlockBreed, services.ErrProductionType,
		services.ErrSaleNeedsFlock, services.ErrSaleFlockMissing):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		log.Printf("Hatchery/batch request failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
	}
}
