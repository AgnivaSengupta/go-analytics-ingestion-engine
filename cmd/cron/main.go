package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AgnivaSengupta/analytics-engine/internal/metrics"
	"github.com/AgnivaSengupta/analytics-engine/internal/rollups"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
)

func main() {
	dbURL := os.Getenv("DB_DSN")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/analytics?sslmode=disable"
	}

	dbPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer dbPool.Close()

	c := cron.New()
	_, err = c.AddFunc("*/1 * * * *", func() {
		processPendingAggregateWork(context.Background(), dbPool)
	})
	if err != nil {
		log.Fatalf("register cron for aggregate work queue: %v", err)
	}
	registerBuilders(c, dbPool, rollups.AggregateBuilders())
	registerBuilders(c, dbPool, rollups.ReconciliationBuilders())
	registerBuilders(c, dbPool, rollups.CleanupBuilders())

	c.Start()
	log.Println("Cron Service started with canonical aggregate builders")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}

func registerBuilders(c *cron.Cron, db *pgxpool.Pool, builders []rollups.Builder) {
	for _, builder := range builders {
		builder := builder
		_, err := c.AddFunc(builder.Schedule, func() {
			now := nowUTC()
			from := builder.From(now)
			to := builder.To(now)
			log.Printf("running %s from %s to %s", builder.Name, from.Format("2006-01-02T15:04:05Z07:00"), to.Format("2006-01-02T15:04:05Z07:00"))
			err := rollups.RunBuilder(context.Background(), db, builder, from, to)
			metrics.RecordRollupRun(err == nil)
			if err != nil {
				log.Printf("%s failed: %v", builder.Name, err)
			}
		})
		if err != nil {
			log.Fatalf("register cron for %s: %v", builder.Name, err)
		}
	}
}

func nowUTC() time.Time {
	return time.Now().UTC()
}

type aggregateWorkItem struct {
	ID         int64
	SiteID     string
	EventID    string
	OccurredAt time.Time
}

func processPendingAggregateWork(ctx context.Context, db *pgxpool.Pool) {
	items, err := claimAggregateWork(ctx, db, 250)
	if err != nil {
		log.Printf("claim aggregate work: %v", err)
		return
	}
	if len(items) == 0 {
		return
	}

	from, to := aggregateWorkWindow(items)
	log.Printf("processing %d aggregate work items from %s to %s", len(items), from.Format(time.RFC3339), to.Format(time.RFC3339))

	for _, builder := range rollups.AggregateBuilders() {
		builderFrom, builderTo := rollups.WindowForDirtyRange(builder, from, to)
		err := rollups.RunBuilder(ctx, db, builder, builderFrom, builderTo)
		metrics.RecordRollupRun(err == nil)
		if err != nil {
			log.Printf("aggregate work builder %s failed: %v", builder.Name, err)
			if releaseErr := releaseAggregateWork(ctx, db, items, err.Error()); releaseErr != nil {
				log.Printf("release aggregate work after failure: %v", releaseErr)
			}
			return
		}
	}

	if err := completeAggregateWork(ctx, db, items); err != nil {
		log.Printf("complete aggregate work: %v", err)
	}
}

func claimAggregateWork(ctx context.Context, db *pgxpool.Pool, limit int) ([]aggregateWorkItem, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	rows, err := tx.Query(ctx, `
		WITH candidates AS (
			SELECT id
			FROM aggregate_work_queue
			WHERE status IN ('pending', 'failed')
			ORDER BY occurred_at ASC, enqueued_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE aggregate_work_queue awq
		SET status = 'processing',
		    claimed_at = NOW(),
		    attempt_count = attempt_count + 1,
		    last_error = NULL
		FROM candidates
		WHERE awq.id = candidates.id
		RETURNING awq.id, awq.site_id, awq.event_id, awq.occurred_at
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]aggregateWorkItem, 0)
	for rows.Next() {
		var item aggregateWorkItem
		if err := rows.Scan(&item.ID, &item.SiteID, &item.EventID, &item.OccurredAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return items, nil
}

func aggregateWorkWindow(items []aggregateWorkItem) (time.Time, time.Time) {
	minOccurred := items[0].OccurredAt.UTC()
	maxOccurred := items[0].OccurredAt.UTC()
	for _, item := range items[1:] {
		if item.OccurredAt.Before(minOccurred) {
			minOccurred = item.OccurredAt.UTC()
		}
		if item.OccurredAt.After(maxOccurred) {
			maxOccurred = item.OccurredAt.UTC()
		}
	}
	return minOccurred, maxOccurred.Add(time.Second)
}

func completeAggregateWork(ctx context.Context, db *pgxpool.Pool, items []aggregateWorkItem) error {
	return updateAggregateWorkStatus(ctx, db, items, "processed", nil)
}

func releaseAggregateWork(ctx context.Context, db *pgxpool.Pool, items []aggregateWorkItem, reason string) error {
	return updateAggregateWorkStatus(ctx, db, items, "failed", &reason)
}

func updateAggregateWorkStatus(ctx context.Context, db *pgxpool.Pool, items []aggregateWorkItem, status string, reason *string) error {
	if len(items) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	_, err := db.Exec(ctx, `
		UPDATE aggregate_work_queue
		SET status = $2,
		    processed_at = CASE WHEN $2 = 'processed' THEN NOW() ELSE processed_at END,
		    claimed_at = CASE WHEN $2 = 'processed' THEN claimed_at ELSE NULL END,
		    last_error = $3
		WHERE id = ANY($1)
	`, ids, status, reason)
	return err
}
