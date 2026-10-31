package api

import (
	"net/http"

	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services"

	"github.com/gin-gonic/gin"
)

// SetupContactRoutes registers customers and suppliers, their statements, and the batch comparison
func SetupContactRoutes(r *gin.Engine) {
	g := r.Group("/").Use(middlewares.FarmAuth())
	{
		g.GET("/contacts", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			list, err := services.ListContacts(db.DB, farmID, c.Query("kind"))
			respond(c, list, err, http.StatusOK)
		})
		g.POST("/contacts", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			var ct models.Contact
			if !bind(c, &ct) {
				return
			}
			ct.ID = 0
			respond(c, &ct, services.SaveContact(db.DB, farmID, &ct), http.StatusCreated)
		})
		g.PUT("/contacts/:id", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			id, ok := idParam(c)
			var ct models.Contact
			if !ok || !bind(c, &ct) {
				return
			}
			ct.ID = id
			respond(c, &ct, services.SaveContact(db.DB, farmID, &ct), http.StatusOK)
		})
		g.DELETE("/contacts/:id", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			if id, ok := idParam(c); ok {
				respond(c, gin.H{"message": "Contact deleted"}, services.DeleteContact(db.DB, farmID, id), http.StatusOK)
			}
		})
		g.GET("/contacts/:id/statement", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			if id, ok := idParam(c); ok {
				st, err := services.ContactStatement(db.DB, farmID, id)
				respond(c, st, err, http.StatusOK)
			}
		})
		g.GET("/flock-compare", func(c *gin.Context) {
			farmID, _ := farmContext(c)
			rows, err := services.CompareBatches(db.DB, farmID)
			respond(c, rows, err, http.StatusOK)
		})
	}
}
