package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/AgnivaSengupta/analytics-engine/internal/analytics"
	"github.com/AgnivaSengupta/analytics-engine/internal/metrics"
	"github.com/AgnivaSengupta/analytics-engine/internal/queue"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mssola/user_agent"
)

const (
	queueName        = "analytics_queue"
	batchSize        = 500
	batchTimeout     = 5 * time.Second
	processAttempts  = 3
	retryBaseBackoff = 200 * time.Millisecond
)

type queuedMessage struct {
	Raw []byte
}

type eventEnvelope struct {
	Event analytics.Event
	Raw   []byte
}

type processResult struct {
	Inserted  bool
	Duplicate bool
}

func main() {
	if err := queue.InitRedis(); err != nil {
		log.Fatalf("Failed to init Redis: %v", err)
	}

	dbURL := os.Getenv("DB_DSN")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/analytics?sslmode=disable"
	}

	dbPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer dbPool.Close()

	log.Println("Worker started. Listening for events...")
	processQueue(dbPool)
}

func processQueue(db *pgxpool.Pool) {
	var batch []queuedMessage
	ticker := time.NewTicker(batchTimeout)
	defer ticker.Stop()

	ctx := context.Background()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-sigChan:
			log.Println("Shutting down worker...")
			if len(batch) > 0 {
				processBatch(db, batch)
			}
			return
		case <-ticker.C:
			if len(batch) > 0 {
				log.Printf("Timeout reached. Processing %d queued events.", len(batch))
				processBatch(db, batch)
				batch = nil
			}
		default:
			message, err := dequeue(ctx)
			if err != nil {
				continue
			}

			metrics.RecordWorkerRead(1)
			batch = append(batch, message)

			if len(batch) >= batchSize {
				log.Printf("Batch full. Processing %d queued events.", len(batch))
				processBatch(db, batch)
				batch = nil
			}
		}
	}
}

func dequeue(ctx context.Context) (queuedMessage, error) {
	result, err := queue.Client.BLPop(ctx, time.Second, queueName).Result()
	if err != nil {
		return queuedMessage{}, err
	}

	return queuedMessage{Raw: []byte(result[1])}, nil
}

func processBatch(db *pgxpool.Pool, batch []queuedMessage) {
	insertedCount := 0

	for _, message := range batch {
		result, err := processMessageWithRetry(context.Background(), db, message)
		if err != nil {
			metrics.RecordWorkerInsertFailure(1)
			log.Printf("failed to process queued event: %v", err)
			continue
		}

		if result.Inserted {
			insertedCount++
		}
	}

	if insertedCount > 0 {
		metrics.RecordWorkerInsertSuccess(insertedCount)
		log.Printf("Synced %d canonical events to Postgres", insertedCount)
	}
}

func processMessageWithRetry(ctx context.Context, db *pgxpool.Pool, message queuedMessage) (processResult, error) {
	var lastErr error

	for attempt := 1; attempt <= processAttempts; attempt++ {
		result, err := processMessage(ctx, db, message, attempt)
		if err == nil {
			return result, nil
		}

		lastErr = err
		if !isRetryable(err) || attempt == processAttempts {
			return processResult{}, err
		}

		time.Sleep(time.Duration(attempt) * retryBaseBackoff)
	}

	return processResult{}, lastErr
}

func processMessage(ctx context.Context, db *pgxpool.Pool, message queuedMessage, attempt int) (processResult, error) {
	envelope, err := parseMessage(message)
	if err != nil {
		if dlqErr := persistDeadLetter(ctx, db, nil, message.Raw, "parse_failed: "+err.Error(), attempt); dlqErr != nil {
			log.Printf("failed to persist dead letter after parse error: %v", dlqErr)
		}
		return processResult{}, err
	}

	if err := validateEvent(envelope.Event); err != nil {
		if dlqErr := persistDeadLetter(ctx, db, &envelope.Event, envelope.Raw, "validation_failed: "+err.Error(), attempt); dlqErr != nil {
			log.Printf("failed to persist dead letter after validation error: %v", dlqErr)
		}
		return processResult{}, err
	}

	enriched, err := enrichEvent(envelope.Event)
	if err != nil {
		if dlqErr := persistDeadLetter(ctx, db, &envelope.Event, envelope.Raw, "enrichment_failed: "+err.Error(), attempt); dlqErr != nil {
			log.Printf("failed to persist dead letter after enrichment error: %v", dlqErr)
		}
		return processResult{}, err
	}

	result, err := persistEvent(ctx, db, enriched, envelope.Raw)
	if err != nil {
		return processResult{}, err
	}

	return result, nil
}

func parseMessage(message queuedMessage) (eventEnvelope, error) {
	var event analytics.Event
	if err := json.Unmarshal(message.Raw, &event); err != nil {
		return eventEnvelope{}, err
	}

	return eventEnvelope{
		Event: event,
		Raw:   message.Raw,
	}, nil
}

func validateEvent(event analytics.Event) error {
	return event.Validate()
}

func persistEvent(ctx context.Context, db *pgxpool.Pool, event analytics.Event, raw []byte) (result processResult, err error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return processResult{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(context.Background())
		}
	}()

	if err = persistRawEvent(ctx, tx, event, raw); err != nil {
		return processResult{}, err
	}

	inserted, err := persistCanonicalEvent(ctx, tx, event)
	if err != nil {
		return processResult{}, err
	}

	if inserted {
		if err = updateVisitor(ctx, tx, event); err != nil {
			return processResult{}, err
		}
		if err = updateSession(ctx, tx, event); err != nil {
			return processResult{}, err
		}
		if err = enqueueAggregateWork(ctx, tx, event); err != nil {
			return processResult{}, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return processResult{}, err
	}

	return processResult{
		Inserted:  inserted,
		Duplicate: !inserted,
	}, nil
}

func persistRawEvent(ctx context.Context, tx pgx.Tx, event analytics.Event, raw []byte) error {
	receivedAt, err := event.ReceivedAtTime()
	if err != nil {
		return err
	}

	var requestIP any
	if ip := parseIP(event.IPAddress); ip != nil {
		requestIP = ip.String()
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO raw_events (
			site_id,
			event_id,
			received_at,
			payload,
			source_type,
			request_ip,
			user_agent
		)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7)
	`,
		event.SiteID,
		event.EventID,
		receivedAt,
		string(raw),
		detectSourceType(event),
		requestIP,
		nullIfEmpty(event.UserAgent),
	)

	return err
}

func persistCanonicalEvent(ctx context.Context, tx pgx.Tx, event analytics.Event) (bool, error) {
	occurredAt, err := event.OccurredAtTime()
	if err != nil {
		return false, err
	}

	receivedAt, err := event.ReceivedAtTime()
	if err != nil {
		return false, err
	}

	propertiesJSON, err := json.Marshal(event.Properties)
	if err != nil {
		return false, err
	}

	contextJSON, err := json.Marshal(event.Context)
	if err != nil {
		return false, err
	}

	var ip any
	if parsed := parseIP(event.IPAddress); parsed != nil {
		ip = parsed.String()
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO events (
			event_id,
			site_id,
			visitor_id,
			session_id,
			event_name,
			event_type,
			occurred_at,
			received_at,
			page_url,
			page_path,
			referrer,
			user_agent,
			ip_address,
			schema_version,
			properties,
			context
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15::jsonb, $16::jsonb
		)
		ON CONFLICT (site_id, event_id) DO NOTHING
	`,
		event.EventID,
		event.SiteID,
		event.VisitorID,
		event.SessionID,
		event.EventName,
		event.EventType,
		occurredAt,
		receivedAt,
		event.PageURL,
		event.PagePath,
		nullIfEmpty(event.Referrer),
		nullIfEmpty(event.UserAgent),
		ip,
		event.SchemaVersion,
		string(propertiesJSON),
		string(contextJSON),
	)
	if err != nil {
		return false, err
	}

	return tag.RowsAffected() == 1, nil
}

func updateVisitor(ctx context.Context, tx pgx.Tx, event analytics.Event) error {
	occurredAt, err := event.OccurredAtTime()
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO visitors (
			site_id,
			visitor_id,
			first_seen_at,
			last_seen_at,
			first_referrer,
			last_referrer,
			first_page_url,
			last_page_url
		)
		VALUES ($1, $2, $3, $3, $4, $4, $5, $5)
		ON CONFLICT (site_id, visitor_id)
		DO UPDATE SET
			last_seen_at = GREATEST(visitors.last_seen_at, EXCLUDED.last_seen_at),
			last_referrer = COALESCE(EXCLUDED.last_referrer, visitors.last_referrer),
			last_page_url = COALESCE(EXCLUDED.last_page_url, visitors.last_page_url),
			updated_at = NOW()
	`,
		event.SiteID,
		event.VisitorID,
		occurredAt,
		nullIfEmpty(event.Referrer),
		event.PageURL,
	)

	return err
}

func updateSession(ctx context.Context, tx pgx.Tx, event analytics.Event) error {
	occurredAt, err := event.OccurredAtTime()
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO sessions (
			site_id,
			session_id,
			visitor_id,
			started_at,
			ended_at,
			landing_page_url,
			landing_page_path,
			landing_referrer,
			device_type,
			os_name,
			geo_country,
			source,
			medium,
			campaign
		)
		VALUES (
			$1, $2, $3, $4, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13
		)
		ON CONFLICT (site_id, session_id)
		DO UPDATE SET
			ended_at = GREATEST(sessions.ended_at, EXCLUDED.ended_at),
			visitor_id = EXCLUDED.visitor_id,
			device_type = COALESCE(sessions.device_type, EXCLUDED.device_type),
			os_name = COALESCE(sessions.os_name, EXCLUDED.os_name),
			geo_country = COALESCE(sessions.geo_country, EXCLUDED.geo_country),
			source = COALESCE(sessions.source, EXCLUDED.source),
			medium = COALESCE(sessions.medium, EXCLUDED.medium),
			campaign = COALESCE(sessions.campaign, EXCLUDED.campaign),
			updated_at = NOW()
	`,
		event.SiteID,
		event.SessionID,
		event.VisitorID,
		occurredAt,
		event.PageURL,
		event.PagePath,
		nullIfEmpty(event.Referrer),
		contextString(event.Context, "device_type"),
		contextString(event.Context, "os_name"),
		contextString(event.Context, "geo_country"),
		contextString(event.Context, "source"),
		contextString(event.Context, "medium"),
		contextString(event.Context, "campaign"),
	)

	return err
}

func enqueueAggregateWork(ctx context.Context, tx pgx.Tx, event analytics.Event) error {
	occurredAt, err := event.OccurredAtTime()
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO aggregate_work_queue (
			site_id,
			event_id,
			occurred_at,
			status
		)
		VALUES ($1, $2, $3, 'pending')
		ON CONFLICT (site_id, event_id) DO NOTHING
	`,
		event.SiteID,
		event.EventID,
		occurredAt,
	)
	if err == nil {
		log.Printf("aggregate work enqueued for site=%s event=%s", event.SiteID, event.EventID)
	}
	return err
}

func persistDeadLetter(ctx context.Context, db *pgxpool.Pool, event *analytics.Event, raw []byte, reason string, attempt int) error {
	var siteID any
	var eventID any
	if event != nil {
		siteID = nullIfEmpty(event.SiteID)
		eventID = nullIfEmpty(event.EventID)
	}

	_, err := db.Exec(ctx, `
		INSERT INTO dead_letter_events (
			site_id,
			event_id,
			payload,
			error_reason,
			attempt_count
		)
		VALUES ($1, $2, $3::jsonb, $4, $5)
	`,
		siteID,
		eventID,
		serializeDeadLetterPayload(raw),
		reason,
		attempt,
	)

	return err
}

func enrichEvent(event analytics.Event) (analytics.Event, error) {
	if event.Context == nil {
		event.Context = map[string]any{}
	}

	ua := user_agent.New(event.UserAgent)
	osName := ua.OSInfo().Name
	if osName == "" {
		osName = "Unknown"
	}

	deviceType := "desktop"
	if ua.Mobile() {
		deviceType = "mobile"
	}

	source, medium := deriveSource(event.Referrer)
	campaign := extractUTM(event.PageURL, "utm_campaign")
	if utmSource := extractUTM(event.PageURL, "utm_source"); utmSource != "" {
		source = utmSource
	}
	if utmMedium := extractUTM(event.PageURL, "utm_medium"); utmMedium != "" {
		medium = utmMedium
	}

	event.Context["os_name"] = osName
	event.Context["device_type"] = deviceType
	event.Context["geo_country"] = deriveGeoCountry(event.IPAddress)
	event.Context["source"] = source
	event.Context["medium"] = medium
	event.Context["campaign"] = campaign
	event.Context["referrer_host"] = extractHost(event.Referrer)

	return event, nil
}

func deriveSource(referrer string) (string, string) {
	host := extractHost(referrer)
	if host == "" {
		return "direct", "none"
	}
	if strings.Contains(host, "google.") {
		return "google", "organic"
	}
	if strings.Contains(host, "bing.") {
		return "bing", "organic"
	}
	if strings.Contains(host, "twitter.") || strings.Contains(host, "x.com") {
		return "twitter", "social"
	}
	if strings.Contains(host, "linkedin.") {
		return "linkedin", "social"
	}
	return host, "referral"
}

func extractUTM(pageURL, key string) string {
	u, err := url.Parse(pageURL)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(u.Query().Get(key))
}

func extractHost(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}

	u, err := url.Parse(value)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(u.Hostname())
}

func deriveGeoCountry(ipAddress string) string {
	parsed := parseIP(ipAddress)
	if parsed == nil {
		return "Unknown"
	}
	if parsed.IsLoopback() {
		return "Local"
	}
	return "Unknown"
}

func parseIP(value string) net.IP {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return net.ParseIP(strings.TrimSpace(value))
}

func detectSourceType(event analytics.Event) string {
	if strings.HasPrefix(strings.ToLower(event.EventType), "server") {
		return "server"
	}
	return "browser"
}

func contextString(ctxMap map[string]any, key string) any {
	if ctxMap == nil {
		return nil
	}
	value, ok := ctxMap[key]
	if !ok {
		return nil
	}

	str := strings.TrimSpace(toString(value))
	if str == "" {
		return nil
	}
	return str
}

func toString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprint(value)
	}
}

func serializeDeadLetterPayload(raw []byte) string {
	if json.Valid(raw) {
		return string(raw)
	}

	payload, err := json.Marshal(map[string]string{
		"raw_text": string(raw),
	})
	if err != nil {
		return `{"raw_text":"unavailable"}`
	}
	return string(payload)
}

func nullIfEmpty(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func isRetryable(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if strings.HasPrefix(pgErr.Code, "08") || strings.HasPrefix(pgErr.Code, "40") || strings.HasPrefix(pgErr.Code, "53") {
			return true
		}
		return false
	}

	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}
