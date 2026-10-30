package api

import (
	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"

	"github.com/gin-gonic/gin"
)

// farmContext returns the farm and acting user resolved by FarmAuth
func farmContext(c *gin.Context) (farmID, userID uint) {
	return c.GetUint("farm_id"), c.GetUint("user_id")
}

// flockInFarm reports whether the flock belongs to the farm
func flockInFarm(flockID, farmID uint) bool {
	var count int64
	db.DB.Model(&models.Flock{}).Where("id = ? AND farm_id = ?", flockID, farmID).Count(&count)
	return count > 0
}

// dbFirst loads a record by primary key
func dbFirst(dest interface{}, id uint) error {
	return db.DB.First(dest, id).Error
}
