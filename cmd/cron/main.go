package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
)

func main() {
	dbUrl := os.Getenv("DB_DSN")
	dbPool, err := pgxpool.New(context.Background(), dbUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer dbPool.Close()

	intiSchema(dbPool)

	// cron job
	c := cron.New()

	// c.AddFunc("*/ * * * *", func() {
	// 	log.Println("Running Aggregation Job...")
	// 	runAggregation(dbPool)
	// })

	// --- A. AGGREGATION JOBS (The "Rollup") ---

	// 1. Hourly: Every minute
	// Reads RAW -> Writes HOURLY
	c.AddFunc("*/1 * * * *", func() {
		log.Println("🔄 Aggregating Hourly...")
		updateHourly(dbPool)
	})

	// 2. Daily/Monthly: Every hour (at minute 5)
	// Reads HOURLY -> Writes DAILY & MONTHLY
	c.AddFunc("5 * * * *", func() {
		log.Println("📅 Aggregating Daily & Monthly...")
		updateDaily(dbPool)
		updateMonthly(dbPool)
	})

	// 3. Yearly: Once a day (at 01:00)
	// Reads MONTHLY -> Writes YEARLY
	c.AddFunc("0 1 * * *", func() {
		log.Println("🗓️ Aggregating Yearly...")
		updateYearly(dbPool)
	})

	// c.Start()
	// log.Println("Cron service started....")
	// select {}

	// --- B. CLEANUP JOB (The "Diamond" at bottom right) ---

	// Runs every hour.
	// Deletes raw events older than 24 hours.
	c.AddFunc("0 * * * *", func() {
		log.Println("🧹 Cleaning Raw DB (Retention Policy)...")
		cleanupRawData(dbPool)
	})

	c.Start()
	log.Println("⏰ Cron Service with Retention Policy Started")

	// Block forever
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}

func runAggregation(db *pgxpool.Pool) {

}

func intiSchema(db *pgxpool.Pool) {
	queries := []string{
		// 1. Hourly Stats Table
		`CREATE TABLE IF NOT EXISTS hourly_stats (
				time_bucket TIMESTAMPTZ NOT NULL,
				blog_id     TEXT NOT NULL,
				views       BIGINT DEFAULT 0,
				visitors    BIGINT DEFAULT 0,
				PRIMARY KEY (blog_id, time_bucket)
			);`,
		// Index for "Last 24 Hours" dashboard charts
		`CREATE INDEX IF NOT EXISTS idx_hourly_time ON hourly_stats (blog_id, time_bucket DESC);`,

		// 2. Daily Stats Table
		`CREATE TABLE IF NOT EXISTS daily_stats (
				day         DATE NOT NULL,
				blog_id     TEXT NOT NULL,
				views       BIGINT DEFAULT 0,
				visitors    BIGINT DEFAULT 0,
				PRIMARY KEY (blog_id, day)
			);`,
		// Index for "Last 30 Days" dashboard charts
		`CREATE INDEX IF NOT EXISTS idx_daily_time ON daily_stats (blog_id, day DESC);`,

		// 3. Monthly Stats Table
		`CREATE TABLE IF NOT EXISTS monthly_stats (
				month       DATE NOT NULL,
				blog_id     TEXT NOT NULL,
				views       BIGINT DEFAULT 0,
				visitors    BIGINT DEFAULT 0,
				PRIMARY KEY (blog_id, month)
			);`,
		// Index for "All Time" charts
		`CREATE INDEX IF NOT EXISTS idx_monthly_time ON monthly_stats (blog_id, month DESC);`,

		// 4. Yearly Stats Table
		`CREATE TABLE IF NOT EXISTS yearly_stats (
				year        DATE NOT NULL,
				blog_id     TEXT NOT NULL,
				views       BIGINT DEFAULT 0,
				visitors    BIGINT DEFAULT 0,
				PRIMARY KEY (blog_id, year)
			);`,
		`CREATE INDEX IF NOT EXISTS idx_yearly_time ON yearly_stats (blog_id, year DESC);`,
	}

	ctx := context.Background()
	for _, q := range queries {
		timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err := db.Exec(timeoutCtx, q)
		cancel()

		if err != nil {
			log.Printf("Schema Init Warning: %v", err)
		}
	}

	log.Println("✅ Aggregation Schema initialized")
}

// --- LOGIC FUNCTIONS ---

// 1. Hourly Aggregation
func updateHourly(db *pgxpool.Pool) {
	// Source: Raw Events
	// We scan only the current hour's data from the raw table
	sql := `
		INSERT INTO hourly_stats (time_bucket, blog_id, views, visitors)
		SELECT
			DATE_TRUNC('hour', timestamp) as bucket,
			blog_id,
			COUNT(*) as views,
			COUNT(DISTINCT user_id) as visitors
		FROM analytics_events
		WHERE timestamp >= DATE_TRUNC('hour', NOW())
		GROUP BY 1, 2
		ON CONFLICT (blog_id, time_bucket)
		DO UPDATE SET views = EXCLUDED.views, visitors = EXCLUDED.visitors;
	`
	exec(db, sql)
}

// 2. Daily Aggregation
func updateDaily(db *pgxpool.Pool) {
	// Source: Hourly Stats (Fast!)
	// We sum up the 24 hourly buckets to get the day total
	sql := `
		INSERT INTO daily_stats (day, blog_id, views, visitors)
		SELECT
			DATE(time_bucket) as day,
			blog_id,
			SUM(views) as views,
			MAX(visitors) as visitors -- Approximation for speed
		FROM hourly_stats
		WHERE time_bucket >= DATE_TRUNC('day', NOW())
		GROUP BY 1, 2
		ON CONFLICT (blog_id, day)
		DO UPDATE SET views = EXCLUDED.views, visitors = EXCLUDED.visitors;
	`
	exec(db, sql)
}

// 3. Monthly Aggregation
func updateMonthly(db *pgxpool.Pool) {
	// Source: Daily Stats
	sql := `
		INSERT INTO monthly_stats (month, blog_id, views, visitors)
		SELECT
			DATE_TRUNC('month', day)::DATE as month,
			blog_id,
			SUM(views) as views,
			SUM(visitors) as visitors
		FROM daily_stats
		WHERE day >= DATE_TRUNC('month', NOW())
		GROUP BY 1, 2
		ON CONFLICT (blog_id, month)
		DO UPDATE SET views = EXCLUDED.views, visitors = EXCLUDED.visitors;
	`
	exec(db, sql)
}

// 4. Yearly Aggregation (Matches your diagram)
func updateYearly(db *pgxpool.Pool) {
	// Source: Monthly Stats
	sql := `
		INSERT INTO yearly_stats (year, blog_id, views, visitors)
		SELECT
			DATE_TRUNC('year', month)::DATE as year,
			blog_id,
			SUM(views) as views,
			SUM(visitors) as visitors
		FROM monthly_stats
		WHERE month >= DATE_TRUNC('year', NOW())
		GROUP BY 1, 2
		ON CONFLICT (blog_id, year)
		DO UPDATE SET views = EXCLUDED.views, visitors = EXCLUDED.visitors;
	`
	exec(db, sql)
}

// 5. THE CLEANER (Retention Policy)
func cleanupRawData(db *pgxpool.Pool) {
	// Deletes any raw event older than 24 hours.
	// This keeps your Raw DB table small and fast.
	sql := `
		DELETE FROM analytics_events
		WHERE timestamp < NOW() - INTERVAL '24 hours';
	`
	exec(db, sql)
}

func exec(db *pgxpool.Pool, sql string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	_, err := db.Exec(ctx, sql)
	if err != nil {
		log.Printf("❌ Error: %v", err)
	}
}
