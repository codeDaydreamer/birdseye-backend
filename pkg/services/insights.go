package services

import (
	"sort"
	"time"

	"birdseye-backend/pkg/db"
	"birdseye-backend/pkg/models"
)

// Platform insights for the admin console: growth, engagement, revenue, billing health and what
// farmers are actually doing. Everything is read-only aggregate SQL over the farm-scoped tables.

// DayCount is one day of a series
type DayCount struct {
	Day   string  `json:"day"`
	Count int64   `json:"count"`
	Value float64 `json:"value,omitempty"`
}

// recordTables are the tables whose rows count as "a farm doing something"
var recordTables = []string{"egg_productions", "sales", "expenses", "bird_movements", "vaccinations", "flocks"}

func dayKey(t time.Time) string { return t.Format("2006-01-02") }

// emptySeries returns the last n days (oldest first), all zero
func emptySeries(n int, now time.Time) ([]DayCount, map[string]int) {
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(n - 1))
	series := make([]DayCount, n)
	index := map[string]int{}
	for i := range series {
		d := dayKey(start.AddDate(0, 0, i))
		series[i].Day, index[d] = d, i
	}
	return series, index
}

type dayRow struct {
	Day   time.Time
	Count int64
	Value float64
}

func fillSeries(series []DayCount, index map[string]int, rows []dayRow) []DayCount {
	for _, r := range rows {
		if i, ok := index[dayKey(r.Day)]; ok {
			series[i].Count += r.Count
			series[i].Value += r.Value
		}
	}
	return series
}

// activeFarmIDs returns the farms that created any record since `since`
func activeFarmIDs(since time.Time) map[uint]time.Time {
	out := map[uint]time.Time{}
	for _, table := range recordTables {
		rows := []struct {
			FarmID uint
			Last   time.Time
		}{}
		db.DB.Table(table).Select("farm_id, MAX(created_at) AS last").Where("created_at >= ?", since).Group("farm_id").Scan(&rows)
		for _, r := range rows {
			if r.Last.After(out[r.FarmID]) {
				out[r.FarmID] = r.Last
			}
		}
	}
	return out
}

// LastActivity returns each farm's most recent record time (any table)
func LastActivity() map[uint]time.Time {
	return activeFarmIDs(time.Time{})
}

// PlanPrice is what a farm's plan earns per month
func monthlyValue(plan models.Plan, period string) float64 {
	if period == "yearly" && plan.YearlyPrice > 0 {
		return plan.YearlyPrice / 12
	}
	return plan.MonthlyPrice
}

// Insights is the admin dashboard
func Insights(days int) (map[string]interface{}, error) {
	if days < 7 || days > 365 {
		days = 30
	}
	now := time.Now()
	since := now.AddDate(0, 0, -days)
	settings, err := models.GetPlatformSettings()
	if err != nil {
		return nil, err
	}
	grace := time.Duration(settings.GracePeriodDays) * 24 * time.Hour

	// ---- Totals ----
	var users, newUsers, verifiedUsers, farms, newFarms, members, activeBatches int64
	db.DB.Model(&models.User{}).Count(&users)
	db.DB.Model(&models.User{}).Where("created_at >= ?", since).Count(&newUsers)
	db.DB.Model(&models.User{}).Where("email_verified = ?", true).Count(&verifiedUsers)
	db.DB.Model(&models.Farm{}).Count(&farms)
	db.DB.Model(&models.Farm{}).Where("created_at >= ?", since).Count(&newFarms)
	db.DB.Model(&models.FarmMember{}).Where("status = ? AND role <> ?", "active", "owner").Count(&members)
	db.DB.Model(&models.Flock{}).Where("status = ?", "active").Count(&activeBatches)
	var birds struct{ N int64 }
	db.DB.Model(&models.Flock{}).Select("COALESCE(SUM(bird_count), 0) AS n").Where("status = ?", "active").Scan(&birds)
	var eggs struct{ N int64 }
	db.DB.Model(&models.EggProduction{}).Select("COALESCE(SUM(eggs_collected), 0) AS n").Where("date_produced >= ?", since).Scan(&eggs)
	var salesValue, expenseValue struct{ N float64 }
	db.DB.Model(&models.Sale{}).Select("COALESCE(SUM(amount), 0) AS n").Where("date >= ?", since).Scan(&salesValue)
	db.DB.Model(&models.Expense{}).Select("COALESCE(SUM(amount), 0) AS n").Where("date >= ?", since).Scan(&expenseValue)
	var loggedIn7d int64
	db.DB.Model(&models.User{}).Where("last_login >= ?", now.AddDate(0, 0, -7)).Count(&loggedIn7d)

	active7 := activeFarmIDs(now.AddDate(0, 0, -7))
	active30 := activeFarmIDs(now.AddDate(0, 0, -30))

	// ---- Billing ----
	var allFarms []models.Farm
	db.DB.Omit("Subscription", "BillingInfo").Find(&allFarms)
	plans := map[uint]models.Plan{}
	var planRows []models.Plan
	db.DB.Find(&planRows)
	for _, p := range planRows {
		plans[p.ID] = p
	}
	status := map[string]int64{"trial": 0, "paid": 0, "unpaid": 0}
	planMix := map[string]int64{}
	mrr := 0.0
	trialsEnding := []map[string]interface{}{}
	for _, f := range allFarms {
		s := "unpaid"
		switch {
		case f.PaidUntil != nil && now.Before(f.PaidUntil.Add(grace)):
			s = "paid"
		case now.Before(f.TrialEndsAt):
			s = "trial"
		}
		status[s]++
		name := "No plan"
		if f.PlanID != nil {
			if p, ok := plans[*f.PlanID]; ok {
				name = p.Name
				if s == "paid" {
					mrr += monthlyValue(p, f.BillingPeriod)
				}
			}
		}
		if s == "paid" {
			planMix[name]++
		}
		if s == "trial" && f.TrialEndsAt.Before(now.AddDate(0, 0, 7)) {
			_, isActive := active7[f.ID]
			trialsEnding = append(trialsEnding, map[string]interface{}{"farm_id": f.ID, "name": f.Name, "trial_ends_at": f.TrialEndsAt, "active_7d": isActive})
		}
	}
	sort.Slice(trialsEnding, func(i, j int) bool {
		return trialsEnding[i]["trial_ends_at"].(time.Time).Before(trialsEnding[j]["trial_ends_at"].(time.Time))
	})

	// Trial -> paid: of farms whose trial has ended, how many have ever paid
	var trialEnded, converted int64
	db.DB.Model(&models.Farm{}).Where("trial_ends_at < ?", now).Count(&trialEnded)
	db.DB.Model(&models.Farm{}).Where("trial_ends_at < ? AND id IN (SELECT DISTINCT farm_id FROM payments WHERE status = 'success')", now).Count(&converted)
	var conversion *float64
	if trialEnded > 0 {
		v := float64(converted) / float64(trialEnded) * 100
		conversion = &v
	}

	// ---- Revenue (M-Pesa payments) ----
	var revenuePeriod, revenueAll struct{ N float64 }
	db.DB.Model(&models.Payment{}).Select("COALESCE(SUM(amount), 0) AS n").Where("status = 'success' AND paid_at >= ?", since).Scan(&revenuePeriod)
	db.DB.Model(&models.Payment{}).Select("COALESCE(SUM(amount), 0) AS n").Where("status = 'success'").Scan(&revenueAll)
	var paymentsFailed, paymentsStuck, paymentsOK int64
	db.DB.Model(&models.Payment{}).Where("status = 'failed' AND created_at >= ?", since).Count(&paymentsFailed)
	db.DB.Model(&models.Payment{}).Where("status = 'success' AND created_at >= ?", since).Count(&paymentsOK)
	db.DB.Model(&models.Payment{}).Where("status = 'initiated' AND created_at < ?", now.Add(-20*time.Minute)).Count(&paymentsStuck)

	months := []map[string]interface{}{}
	for i := 11; i >= 0; i-- {
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -i, 0)
		var m struct {
			N float64
			C int64
		}
		db.DB.Model(&models.Payment{}).Select("COALESCE(SUM(amount), 0) AS n, COUNT(*) AS c").
			Where("status = 'success' AND paid_at >= ? AND paid_at < ?", start, start.AddDate(0, 1, 0)).Scan(&m)
		months = append(months, map[string]interface{}{"month": start.Format("2006-01"), "amount": m.N, "payments": m.C})
	}

	// ---- Growth and engagement series ----
	seriesFor := func(table, dateCol, value string) []DayCount {
		series, index := emptySeries(days, now)
		rows := []dayRow{}
		sel := "DATE(" + dateCol + ") AS day, COUNT(*) AS count"
		if value != "" {
			sel += ", COALESCE(SUM(" + value + "), 0) AS value"
		}
		db.DB.Table(table).Select(sel).Where(dateCol+" >= ?", since).Group("DATE(" + dateCol + ")").Scan(&rows)
		return fillSeries(series, index, rows)
	}
	signups := seriesFor("users", "created_at", "")
	farmSeries := seriesFor("farms", "created_at", "")
	revenueSeries := seriesFor("payments", "paid_at", "amount")
	// paid_at is set only on success, but keep failures out of the revenue line
	{
		series, index := emptySeries(days, now)
		rows := []dayRow{}
		db.DB.Table("payments").Select("DATE(paid_at) AS day, COUNT(*) AS count, COALESCE(SUM(amount), 0) AS value").
			Where("status = 'success' AND paid_at >= ?", since).Group("DATE(paid_at)").Scan(&rows)
		revenueSeries = fillSeries(series, index, rows)
	}
	// Active farms per day: distinct farms with a record that day
	activeSeries, aIndex := emptySeries(days, now)
	perDay := map[string]map[uint]bool{}
	for _, table := range recordTables {
		rows := []struct {
			Day    time.Time
			FarmID uint
		}{}
		db.DB.Table(table).Select("DATE(created_at) AS day, farm_id").Where("created_at >= ?", since).Group("DATE(created_at), farm_id").Scan(&rows)
		for _, r := range rows {
			k := dayKey(r.Day)
			if perDay[k] == nil {
				perDay[k] = map[uint]bool{}
			}
			perDay[k][r.FarmID] = true
		}
	}
	for k, set := range perDay {
		if i, ok := aIndex[k]; ok {
			activeSeries[i].Count = int64(len(set))
		}
	}

	// Records logged per type (what people use)
	usage := []map[string]interface{}{}
	for _, t := range []struct{ table, label string }{
		{"egg_productions", "Egg collections"}, {"sales", "Sales"}, {"expenses", "Expenses"},
		{"bird_movements", "Bird movements"}, {"vaccinations", "Vaccinations"}, {"flocks", "Batches"},
		{"egg_settings", "Hatchery settings"}, {"chick_bookings", "Chick bookings"}, {"feed_logs", "Feed logs"},
		{"weigh_ins", "Weigh-ins"}, {"contacts", "Customers & suppliers"},
	} {
		var n, farmsUsing int64
		if !db.DB.Migrator().HasTable(t.table) {
			continue
		}
		db.DB.Table(t.table).Where("created_at >= ?", since).Count(&n)
		db.DB.Table(t.table).Where("created_at >= ?", since).Distinct("farm_id").Count(&farmsUsing)
		usage = append(usage, map[string]interface{}{"feature": t.label, "records": n, "farms": farmsUsing})
	}
	sort.Slice(usage, func(i, j int) bool { return usage[i]["records"].(int64) > usage[j]["records"].(int64) })

	// Production types and where farms are
	types := []struct {
		ProductionType string `json:"production_type"`
		Batches        int64  `json:"batches"`
		Birds          int64  `json:"birds"`
	}{}
	db.DB.Model(&models.Flock{}).Select("production_type, COUNT(*) AS batches, COALESCE(SUM(bird_count), 0) AS birds").
		Where("status = ?", "active").Group("production_type").Order("birds DESC").Scan(&types)
	locations := []struct {
		Location string `json:"location"`
		Farms    int64  `json:"farms"`
	}{}
	db.DB.Model(&models.Farm{}).Select("location, COUNT(*) AS farms").Where("location <> ''").Group("location").Order("farms DESC").Limit(10).Scan(&locations)

	// Most active farms (records in the period)
	top := map[uint]int64{}
	for _, table := range recordTables {
		rows := []struct {
			FarmID uint
			N      int64
		}{}
		db.DB.Table(table).Select("farm_id, COUNT(*) AS n").Where("created_at >= ?", since).Group("farm_id").Scan(&rows)
		for _, r := range rows {
			top[r.FarmID] += r.N
		}
	}
	topFarms := []map[string]interface{}{}
	for id, n := range top {
		topFarms = append(topFarms, map[string]interface{}{"farm_id": id, "records": n})
	}
	sort.Slice(topFarms, func(i, j int) bool { return topFarms[i]["records"].(int64) > topFarms[j]["records"].(int64) })
	if len(topFarms) > 8 {
		topFarms = topFarms[:8]
	}
	names := map[uint]string{}
	for _, f := range allFarms {
		names[f.ID] = f.Name
	}
	for _, t := range topFarms {
		t["name"] = names[t["farm_id"].(uint)]
	}

	// Farms that went quiet: active in the 30 days before the last 14, nothing since
	quiet := []map[string]interface{}{}
	recent14 := activeFarmIDs(now.AddDate(0, 0, -14))
	for id, last := range activeFarmIDs(now.AddDate(0, 0, -44)) {
		if _, ok := recent14[id]; !ok {
			quiet = append(quiet, map[string]interface{}{"farm_id": id, "name": names[id], "last_activity": last})
		}
	}
	sort.Slice(quiet, func(i, j int) bool {
		return quiet[i]["last_activity"].(time.Time).After(quiet[j]["last_activity"].(time.Time))
	})

	var setupDone int64
	db.DB.Model(&models.Farm{}).Where("setup_completed_at IS NOT NULL").Count(&setupDone)

	return map[string]interface{}{
		"days": days,
		"totals": map[string]interface{}{
			"users": users, "new_users": newUsers, "verified_users": verifiedUsers, "logged_in_7d": loggedIn7d,
			"farms": farms, "new_farms": newFarms, "staff_members": members, "farms_setup_done": setupDone,
			"active_farms_7d": len(active7), "active_farms_30d": len(active30),
			"active_batches": activeBatches, "birds": birds.N, "eggs": eggs.N,
			"sales_recorded": salesValue.N, "expenses_recorded": expenseValue.N,
		},
		"billing": map[string]interface{}{
			"status": status, "plan_mix": planMix, "mrr": mrr, "arr": mrr * 12,
			"trial_conversion": conversion, "trials_ended": trialEnded, "converted": converted,
			"trials_ending_7d": trialsEnding,
		},
		"revenue": map[string]interface{}{
			"period": revenuePeriod.N, "all_time": revenueAll.N, "months": months,
			"payments_ok": paymentsOK, "payments_failed": paymentsFailed, "payments_stuck": paymentsStuck,
		},
		"series": map[string]interface{}{
			"signups": signups, "farms": farmSeries, "active_farms": activeSeries, "revenue": revenueSeries,
		},
		"usage":            usage,
		"production_types": types,
		"locations":        locations,
		"top_farms":        topFarms,
		"quiet_farms":      quiet,
	}, nil
}

// ActivityItem is one line of the admin activity feed
type ActivityItem struct {
	Kind   string    `json:"kind"` // signup | farm | payment | staff | batch
	Text   string    `json:"text"`
	FarmID uint      `json:"farm_id,omitempty"`
	UserID uint      `json:"user_id,omitempty"`
	Amount float64   `json:"amount,omitempty"`
	Status string    `json:"status,omitempty"`
	At     time.Time `json:"at"`
}

// RecentActivity merges the latest signups, farms, payments, staff joins and new batches
func RecentActivity(limit int) []ActivityItem {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items := []ActivityItem{}

	var users []models.User
	db.DB.Order("created_at DESC").Limit(limit).Find(&users)
	for _, u := range users {
		items = append(items, ActivityItem{Kind: "signup", Text: u.Username + " signed up (" + u.Email + ")", UserID: u.ID, At: u.CreatedAt})
	}
	var farms []models.Farm
	db.DB.Omit("Subscription", "BillingInfo").Order("created_at DESC").Limit(limit).Find(&farms)
	names := map[uint]string{}
	for _, f := range farms {
		names[f.ID] = f.Name
		text := "Farm " + f.Name + " created"
		if f.Location != "" {
			text += " in " + f.Location
		}
		items = append(items, ActivityItem{Kind: "farm", Text: text, FarmID: f.ID, UserID: f.OwnerUserID, At: f.CreatedAt})
	}
	farmName := func(id uint) string {
		if n, ok := names[id]; ok {
			return n
		}
		var f models.Farm
		db.DB.Select("name").First(&f, id)
		names[id] = f.Name
		return f.Name
	}

	var payments []models.Payment
	db.DB.Where("status IN ?", []string{"success", "failed", "refunded"}).Order("updated_at DESC").Limit(limit).Find(&payments)
	for _, p := range payments {
		verb := map[string]string{"success": "paid", "failed": "payment failed", "refunded": "payment refunded"}[p.Status]
		items = append(items, ActivityItem{Kind: "payment", Text: farmName(p.FarmID) + " " + verb, FarmID: p.FarmID, UserID: p.UserID,
			Amount: p.Amount, Status: p.Status, At: p.UpdatedAt.Time})
	}

	members := []struct {
		FarmID    uint
		UserID    uint
		Role      string
		Username  string
		CreatedAt time.Time
	}{}
	db.DB.Table("farm_members").Select("farm_members.farm_id, farm_members.user_id, farm_members.role, users.username, farm_members.created_at").
		Joins("JOIN users ON users.id = farm_members.user_id").Where("farm_members.role <> 'owner'").
		Order("farm_members.created_at DESC").Limit(limit).Scan(&members)
	for _, m := range members {
		items = append(items, ActivityItem{Kind: "staff", Text: m.Username + " joined " + farmName(m.FarmID) + " as " + m.Role, FarmID: m.FarmID, UserID: m.UserID, At: m.CreatedAt})
	}

	var flocks []models.Flock
	db.DB.Select("id, farm_id, name, production_type, initial_bird_count, created_at").Order("created_at DESC").Limit(limit).Find(&flocks)
	for _, f := range flocks {
		items = append(items, ActivityItem{Kind: "batch", Text: farmName(f.FarmID) + " added batch " + f.Name + " (" + f.ProductionType + ")", FarmID: f.FarmID, At: f.CreatedAt})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].At.After(items[j].At) })
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

// FarmOverview is everything an admin needs to understand one farm
func FarmOverview(farmID uint) (map[string]interface{}, error) {
	var farm models.Farm
	if err := db.DB.Omit("Subscription", "BillingInfo").First(&farm, farmID).Error; err != nil {
		return nil, err
	}
	billing, _ := GetFarmBillingStatus(farmID)
	var owner models.User
	db.DB.First(&owner, farm.OwnerUserID)

	members := []struct {
		UserID    uint       `json:"user_id"`
		Username  string     `json:"username"`
		Email     string     `json:"email"`
		Phone     string     `json:"phone_number"`
		Role      string     `json:"role"`
		Status    string     `json:"status"`
		LastLogin *time.Time `json:"last_login"`
		Joined    time.Time  `json:"joined"`
	}{}
	db.DB.Table("farm_members").Select("farm_members.user_id, users.username, users.email, users.phone_number AS phone, farm_members.role, farm_members.status, users.last_login, farm_members.created_at AS joined").
		Joins("JOIN users ON users.id = farm_members.user_id").Where("farm_members.farm_id = ?", farmID).Order("farm_members.created_at").Scan(&members)

	var plan *models.Plan
	if farm.PlanID != nil {
		var p models.Plan
		if db.DB.First(&p, *farm.PlanID).Error == nil {
			plan = &p
		}
	}
	var payments []models.Payment
	db.DB.Where("farm_id = ?", farmID).Order("created_at DESC").Limit(25).Find(&payments)

	var flocks []models.Flock
	db.DB.Where("farm_id = ?", farmID).Order("status, placed_at DESC").Find(&flocks)

	now := time.Now()
	since := now.AddDate(0, 0, -30)
	counts := map[string]int64{}
	for _, table := range recordTables {
		var n int64
		db.DB.Table(table).Where("farm_id = ? AND created_at >= ?", farmID, since).Count(&n)
		counts[table] = n
	}
	var eggs struct{ N int64 }
	db.DB.Model(&models.EggProduction{}).Select("COALESCE(SUM(eggs_collected), 0) AS n").Where("farm_id = ? AND date_produced >= ?", farmID, since).Scan(&eggs)
	var sales, expenses struct{ N float64 }
	db.DB.Model(&models.Sale{}).Select("COALESCE(SUM(amount), 0) AS n").Where("farm_id = ? AND date >= ?", farmID, since).Scan(&sales)
	db.DB.Model(&models.Expense{}).Select("COALESCE(SUM(amount), 0) AS n").Where("farm_id = ? AND date >= ?", farmID, since).Scan(&expenses)

	series, index := emptySeries(30, now)
	for _, table := range recordTables {
		rows := []dayRow{}
		db.DB.Table(table).Select("DATE(created_at) AS day, COUNT(*) AS count").Where("farm_id = ? AND created_at >= ?", farmID, since).Group("DATE(created_at)").Scan(&rows)
		series = fillSeries(series, index, rows)
	}
	var last *time.Time
	if t, ok := LastActivity()[farmID]; ok {
		last = &t
	}

	return map[string]interface{}{
		"farm": farm, "owner": owner, "members": members, "plan": plan, "billing": billing,
		"payments": payments, "batches": flocks, "last_activity": last,
		"last_30_days": map[string]interface{}{"records": counts, "eggs": eggs.N, "sales": sales.N, "expenses": expenses.N},
		"activity":     series,
	}, nil
}
