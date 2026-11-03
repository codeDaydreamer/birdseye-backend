package api

import (
	"birdseye-backend/pkg/appconfig"
	"context"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/middlewares"
	"birdseye-backend/pkg/models"
	"birdseye-backend/pkg/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var startedAt = time.Now()

// adminAudit records every change made through the admin API (after it runs, with its status)
func adminAudit() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodOptions {
			return
		}
		adminID, _ := c.Get("user_id")
		id, _ := adminID.(uint)
		email := ""
		if u, err := middlewares.GetUserFromToken(c.GetHeader("Authorization")); err == nil {
			email = u.Email
		}
		if email == "" && id != 0 {
			var a models.Admin
			if db.DB.Select("email").First(&a, id).Error == nil {
				email = a.Email
			}
		}
		target := c.Param("id")
		ua := c.Request.UserAgent()
		if len(ua) > 300 {
			ua = ua[:300]
		}
		event := models.AdminAuditEvent{
			AdminID: id, AdminEmail: email, Method: c.Request.Method, Path: c.Request.URL.Path,
			Action: c.Request.Method + " " + c.FullPath(), TargetID: target, Status: c.Writer.Status(),
			IP: c.ClientIP(), UserAgent: ua, CreatedAt: time.Now(),
		}
		if err := db.DB.Create(&event).Error; err != nil {
			logrus.Errorf("admin audit: %v", err)
		}
	}
}

func setupAdminInsightRoutes(admin *gin.RouterGroup) {
	admin.GET("/insights", func(c *gin.Context) {
		days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
		out, err := services.Insights(days)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load insights"})
			return
		}
		c.JSON(http.StatusOK, out)
	})
	admin.GET("/activity", func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		c.JSON(http.StatusOK, gin.H{"activity": services.RecentActivity(limit)})
	})
	admin.GET("/farms/:id/overview", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid farm ID"})
			return
		}
		out, err := services.FarmOverview(uint(id))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Farm not found"})
			return
		}
		c.JSON(http.StatusOK, out)
	})
	admin.GET("/farms-activity", handleAdminFarmsActivity)
	admin.GET("/payments", handleAdminPayments)
	admin.GET("/audit", handleAdminAudit)
	admin.GET("/system", handleAdminSystem)
}

// handleAdminFarmsActivity: last activity per farm, for the farms table
func handleAdminFarmsActivity(c *gin.Context) {
	last := services.LastActivity()
	out := map[string]time.Time{}
	for id, t := range last {
		out[strconv.FormatUint(uint64(id), 10)] = t
	}
	c.JSON(http.StatusOK, gin.H{"last_activity": out})
}

// handleAdminPayments lists M-Pesa subscription payments across farms
func handleAdminPayments(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if limit < 1 || limit > 200 {
		limit = 50
	}
	if page < 1 {
		page = 1
	}
	q := db.DB.Table("payments").
		Select("payments.*, farms.name AS farm_name, users.username AS payer, plans.name AS plan_name").
		Joins("LEFT JOIN farms ON farms.id = payments.farm_id").
		Joins("LEFT JOIN users ON users.id = payments.user_id").
		Joins("LEFT JOIN plans ON plans.id = payments.plan_id")
	if v := c.Query("status"); v != "" {
		q = q.Where("payments.status = ?", v)
	}
	if v := c.Query("farm_id"); v != "" {
		q = q.Where("payments.farm_id = ?", v)
	}
	if v := strings.TrimSpace(c.Query("q")); v != "" {
		like := "%" + v + "%"
		q = q.Where("payments.phone_number LIKE ? OR payments.mpesa_reference LIKE ? OR payments.reference LIKE ? OR farms.name LIKE ?", like, like, like, like)
	}
	var total int64
	q.Count(&total)
	rows := []map[string]interface{}{}
	q.Order("payments.created_at DESC").Limit(limit).Offset((page - 1) * limit).Find(&rows)
	c.JSON(http.StatusOK, gin.H{"payments": rows, "total": total, "page": page})
}

func handleAdminAudit(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if limit < 1 || limit > 200 {
		limit = 50
	}
	if page < 1 {
		page = 1
	}
	q := db.DB.Model(&models.AdminAuditEvent{})
	if v := strings.TrimSpace(c.Query("admin")); v != "" {
		q = q.Where("admin_email LIKE ?", "%"+v+"%")
	}
	if v := strings.TrimSpace(c.Query("path")); v != "" {
		q = q.Where("path LIKE ?", "%"+v+"%")
	}
	var total int64
	q.Count(&total)
	events := []models.AdminAuditEvent{}
	q.Order("created_at DESC").Limit(limit).Offset((page - 1) * limit).Find(&events)
	c.JSON(http.StatusOK, gin.H{"events": events, "total": total, "page": page})
}

// handleAdminSystem checks the database, Dynapay and how the server is configured
func handleAdminSystem(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	start := time.Now()
	sqlDB, err := db.DB.DB()
	dbOK := err == nil && sqlDB.PingContext(ctx) == nil
	dbMs := time.Since(start).Milliseconds()
	dbStats := gin.H{}
	if err == nil {
		s := sqlDB.Stats()
		dbStats = gin.H{"open": s.OpenConnections, "in_use": s.InUse, "idle": s.Idle, "wait_count": s.WaitCount}
	}

	dynapay := gin.H{"configured": appconfig.Get("DYNAPAY_API_URL") != "" && appconfig.Get("DYNAPAY_CLIENT_ID") != ""}
	if base := strings.TrimRight(appconfig.Get("DYNAPAY_API_URL"), "/"); base != "" {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/health", nil)
		t := time.Now()
		resp, err := http.DefaultClient.Do(req)
		dynapay["latency_ms"] = time.Since(t).Milliseconds()
		if err != nil {
			dynapay["ok"], dynapay["error"] = false, err.Error()
		} else {
			resp.Body.Close()
			dynapay["ok"] = resp.StatusCode == http.StatusOK
			dynapay["status"] = resp.StatusCode
		}
	}

	set := func(names ...string) bool {
		for _, n := range names {
			if appconfig.Get(n) == "" {
				return false
			}
		}
		return true
	}
	var stuck, pendingCount int64
	db.DB.Model(&models.Payment{}).Where("status = 'initiated' AND created_at < ?", time.Now().Add(-20*time.Minute)).Count(&stuck)
	db.DB.Model(&models.Payment{}).Where("status = 'initiated'").Count(&pendingCount)

	tables := gin.H{}
	for _, t := range []string{"users", "farms", "farm_members", "flocks", "egg_productions", "sales", "expenses", "bird_movements", "vaccinations", "payments", "notifications"} {
		var n int64
		db.DB.Table(t).Count(&n)
		tables[t] = n
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	c.JSON(http.StatusOK, gin.H{
		"database": gin.H{"ok": dbOK, "latency_ms": dbMs, "pool": dbStats, "rows": tables},
		"dynapay":  dynapay,
		"config": gin.H{
			"email":            set("RESEND_API_KEY", "EMAIL_FROM"),
			"web_push":         set("VAPID_PUBLIC_KEY", "VAPID_PRIVATE_KEY"),
			"google_login":     set("GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET"),
			"dynapay_webhooks": set("DYNAPAY_WEBHOOK_SECRET"),
			"firebase_push":    set("FIREBASE_SERVICE_ACCOUNT_FILE") || set("FIREBASE_SERVICE_ACCOUNT_JSON"),
		},
		"payments": gin.H{"initiated": pendingCount, "stuck_over_20m": stuck},
		"server": gin.H{"uptime_seconds": int(time.Since(startedAt).Seconds()), "go": runtime.Version(),
			"goroutines": runtime.NumGoroutine(), "memory_mb": mem.Alloc / 1024 / 1024},
		"checked_at": time.Now(),
	})
}
