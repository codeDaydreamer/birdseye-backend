package api

import (
	"fmt"
	"net/http"
	"time"

	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"

	"github.com/gin-gonic/gin"
)

// DailyLoginCount holds the day and total logins for that day
type DailyLoginCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// StatResponse now includes a field for DailyLoginTrend
type StatResponse struct {
	TotalUsers          int64               `json:"total_users"`
	ActiveToday        int64               `json:"active_today"`
	NewSignups         int64               `json:"new_signups"`
	ActiveSubscriptions int64               `json:"active_subscriptions"`
	DailyLoginTrend   []DailyLoginCount `json:"daily_login_trend"`
}

func SetupStatsRoutes(r *gin.Engine) {
	r.GET("/admin/stats", GetStats)
}

func GetStats(c *gin.Context) {
	var totalUsers int64
	var activeToday int64
	var newSignups int64
	var activeSubscriptions int64

	// total users
	db.DB.Model(&models.User{}).Count(&totalUsers)

	// active today
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())	
	db.DB.Model(&models.User{}).
		Where("last_login >= ?", todayStart).
		Count(&activeToday)

	// new signups this month
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()) 
	db.DB.Model(&models.User{}).
		Where("created_at >= ?", monthStart).
		Count(&newSignups)

	// active subscriptions
	db.DB.Model(&models.Subscription{}).
		Where("status = ?", "active").
		Count(&activeSubscriptions)

	// Daily login trend for past 7 days
	start := now.AddDate(0, 0, -6)

	rows, err := db.DB.Table("users").
		Select("DATE(last_login) as date, COUNT(*) as count").
		Where("last_login IS NOT NULL").
		Where("last_login >= ?", start).
		Group("DATE(last_login)").Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	// put into a map first
	loginMap := make(map[string]int64)
	fmt.Println("Raw rows from DB:")
	for rows.Next() {
		var item DailyLoginCount
		if err := rows.Scan(&item.Date, &item.Count); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		fmt.Printf("-> %s: %d\n", item.Date, item.Count)

		// trim to just "YYYY-MM-DD" in case it's a datetime
		dateStr := item.Date
		if len(dateStr) > 10 {
			dateStr = dateStr[:10]
		}
		loginMap[dateStr] = item.Count
	}

	fmt.Println("Login map after grouping:")
	for k, v := range loginMap {
		fmt.Printf("%s => %d\n", k, v)
	}

	// now fill in all days in range in order
	var dailyTrend []DailyLoginCount
	for i := 0; i < 7; i++ {
		day := start.AddDate(0, 0, i)
		dateStr := day.Format("2006-01-02")
		count := int64(0)
		if val, ok := loginMap[dateStr]; ok {
			count = val
		}
		dailyTrend = append(dailyTrend, DailyLoginCount{Date: dateStr, Count: count})
	}

	fmt.Println("Final Daily Trend:")
	for _, d := range dailyTrend {
		fmt.Printf("%s: %d\n", d.Date, d.Count)
	}

	// Return all in a single JSON
	c.JSON(http.StatusOK, StatResponse{
		TotalUsers:          totalUsers,
		ActiveToday:        activeToday,
		NewSignups:         newSignups,
		ActiveSubscriptions: activeSubscriptions,
		DailyLoginTrend:   dailyTrend,
	})
}
