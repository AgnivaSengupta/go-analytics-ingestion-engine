package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AgnivaSengupta/analytics-engine/internal/queue"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnalyticsEvent struct {
	BlogID    string    `json:"blog_id"`
	Url       string    `json:"url"`
	UserID    string    `json:"user_id,omitempty"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`
}

func main() {

	fmt.Println("DEBUG: Dumping Environment Variables...")
    fmt.Printf("DEBUG: REDIS_DSN='%s'\n", os.Getenv("REDIS_DSN"))
    fmt.Printf("DEBUG: DB_DSN='%s'\n", os.Getenv("DB_DSN"))


	if err := queue.InitReddis(); err != nil {
		log.Fatalf("Failed to init Redis: %v", err)
	}

	dbUrl := os.Getenv("DB_DSN")
	if dbUrl == "" {
		dbUrl = "postgres://user:password@localhost:5432/analytics"
	}

	dbPool, err := pgxpool.New(context.Background(), dbUrl)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer dbPool.Close()

	queries := []string{

		`CREATE TABLE IF NOT EXISTS analytics_events (
			timestamp TIMESTAMPTZ NOT NULL,
			blog_id TEXT NOT NULL,
			url TEXT NOT NULL,
			user_id TEXT,
			event_type TEXT,
			user_agent TEXT,
			ip_address TEXT
		);`,

		`CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_event ON analytics_events (blog_id, user_id, url, timestamp);`,

		`CREATE INDEX IF NOT EXISTS idx_lookup ON analytics_events (blog_id, url, timestamp DESC)`,
	}

	for _, q := range queries {
		_, err = dbPool.Exec(context.Background(), q)
		if err != nil {
			log.Fatalf("Schema warning: %v", err)
		}
	}

	log.Println("🚀 Worker started. Listening for events...")

	// 4. Start Processing Loop
	processQueue(dbPool)
}


func processQueue(db *pgxpool.Pool){
	batchSize := 500
	batchTimeout := 5*time.Second

	var batch []AnalyticsEvent
		ticker := time.NewTicker(batchTimeout)
		defer ticker.Stop()

		ctx := context.Background()

		// Graceful Shutdown handling
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		for {
			select {
			case <-sigChan:
				log.Println("Shutting down worker...")
				if len(batch) > 0 {
					flushToDB(db, batch)
				}
				return

			case <-ticker.C:
				// Time trigger: Flush if we have anything pending
				if len(batch) > 0 {
					fmt.Printf("⏱️ Timeout reached. Flushing %d events.\n", len(batch))
					flushToDB(db, batch)
					batch = nil // Reset
				}

			default:
				// Fetch from Redis (Blocking Pop with 1s timeout to allow loop to check ticker)
				// We use BLPOP to wait efficiently without burning CPU
				result, err := queue.Client.BLPop(ctx, 1*time.Second, "analytics_queue").Result()

				if err != nil {
					// Redis Timeout (no data) is not a fatal error, just loop again
					continue
				}

				// Parse Data (result[1] contains the JSON payload)
				var event AnalyticsEvent
				if err := json.Unmarshal([]byte(result[1]), &event); err != nil {
					log.Printf("❌ Bad JSON: %v", err)
					continue
				}

				batch = append(batch, event)

				// Size trigger: Flush if full
				if len(batch) >= batchSize {
					fmt.Printf("📦 Batch full. Flushing %d events.\n", len(batch))
					flushToDB(db, batch)
					batch = nil // Reset
				}
			}
		}
}


func flushToDB(db *pgxpool.Pool, batch []AnalyticsEvent) {

		pgxBatch := &pgx.Batch{}


		// Prepare the data for CopyFrom
		// rows := [][]interface{}{}
		// for _, e := range batch {
		// 	rows = append(rows, []interface{}{
		// 		e.BlogID, e.Url, e.UserID, e.EventType, e.Timestamp, e.UserAgent, e.IPAddress,
		// 	})
		// }

		for _, e := range batch {
			// Normalize UserID for the Unique Constraint
			uid := e.UserID
			if uid == "" {
				uid = "anon"
			}


			query := `
						INSERT INTO analytics_events
						(timestamp, blog_id, url, user_id, event_type, user_agent, ip_address)
						VALUES ($1, $2, $3, $4, $5, $6, $7)
						ON CONFLICT (blog_id, user_id, url, timestamp) DO NOTHING;
					`

			pgxBatch.Queue(query, e.Timestamp, e.BlogID, e.Url, uid, e.EventType, e.UserAgent, e.IPAddress)

		}


		br := db.SendBatch(context.Background(), pgxBatch)
		defer br.Close()


		_, err := br.Exec()
		if err != nil {
			log.Printf("❌ Failed to insert batch: %v", err)
			// Nuance: In production, you would push these back to a "Dead Letter Queue" in Redis
		} else {
			log.Printf("✅ Synced %d events to Postgres", len(batch))
		}
}