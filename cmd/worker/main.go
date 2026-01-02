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
		dbUrl = "postgresql://neondb_owner:npg_2Dj0yanOAcze@ep-mute-butterfly-a19gn7fh-pooler.ap-southeast-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require"
	}

	dbPool, err := pgxpool.New(context.Background(), dbUrl)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer dbPool.Close()

	createTableSQL := `
		CREATE TABLE IF NOT EXISTS analytics_events (
			id SERIAL PRIMARY KEY,
			blog_id TEXT NOT NULL,
			url TEXT NOT NULL,
			user_id TEXT,
			event_type TEXT,
			timestamp TIMESTAMPTZ NOT NULL,
			user_agent TEXT,
			ip_address TEXT
		);
		`
	_, err = dbPool.Exec(context.Background(), createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
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

// flushToDB uses PostgreSQL COPY protocol for maximum speed
func flushToDB(db *pgxpool.Pool, batch []AnalyticsEvent) {
		// Prepare the data for CopyFrom
		rows := [][]interface{}{}
		for _, e := range batch {
			rows = append(rows, []interface{}{
				e.BlogID, e.Url, e.UserID, e.EventType, e.Timestamp, e.UserAgent, e.IPAddress,
			})
		}

		// Bulk Insert
		count, err := db.CopyFrom(
			context.Background(),
			pgx.Identifier{"analytics_events"},
			[]string{"blog_id", "url", "user_id", "event_type", "timestamp", "user_agent", "ip_address"},
			pgx.CopyFromRows(rows),
		)

		if err != nil {
			log.Printf("❌ Failed to insert batch: %v", err)
			// Nuance: In production, you would push these back to a "Dead Letter Queue" in Redis
		} else {
			log.Printf("✅ Saved %d events to Postgres", count)
		}
}