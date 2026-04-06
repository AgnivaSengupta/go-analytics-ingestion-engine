package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"

	"github.com/AgnivaSengupta/analytics-engine/internal/analytics"
	"github.com/AgnivaSengupta/analytics-engine/internal/auth"
	"github.com/AgnivaSengupta/analytics-engine/internal/metrics"
	"github.com/AgnivaSengupta/analytics-engine/internal/query"
	"github.com/AgnivaSengupta/analytics-engine/internal/queue"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func main() {

	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		log.Fatal("JWT_SECRET is not set")
	}

	if err := queue.InitRedis(); err != nil {
		log.Fatalf("Could not connect to redis: %v", err)
	}
	defer queue.Client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	queue.StartBatcher(ctx)

	dbURL := os.Getenv("DB_DSN")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/analytics?sslmode=disable"
	}
	dbPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer dbPool.Close()

	app := newApp(dbPool, jwtSecret)

	log.Println("Ingestion Service running on port 8080")
	log.Fatal(app.Listen(":8080"))
}

func newApp(db *pgxpool.Pool, jwtSecret []byte) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:   "Analytics_Ingestion_Engine",
		BodyLimit: 4 * 1024 * 1024,
	})

	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, User-Agent",
	}))

	app.Get("/health", healthHandler)
	app.Get("/metrics", metricsHandler)
	// app.Post("/api/ingest", handleIngest)

	// auth api
	app.Post("/v1/auth/login", func(c *fiber.Ctx) error { return handleLogin(c, db, jwtSecret) })
	app.Post("/v1/auth/register", func(c *fiber.Ctx) error { return handleRegister(c, db)})

	// ingest api
	ingest := app.Group("/v1", auth.IngestAuthMiddleware(db))
	ingest.Post("/ingest", handleIngest)
	ingest.Post("/events", handleSingleEvent)

	// query api
	sites := app.Group("/v1", auth.QueryAuthMiddleware(jwtSecret))
	sites.Get("/sites/:site_id/overview", func(c *fiber.Ctx) error { return handleOverview(c, db) })
	sites.Get("/sites/:site_id/realtime", func(c *fiber.Ctx) error { return handleRealtime(c, db) })
	sites.Get("/sites/:site_id/pages", func(c *fiber.Ctx) error { return handlePages(c, db) })
	sites.Get("/sites/:site_id/sources", func(c *fiber.Ctx) error { return handleSources(c, db) })

	return app
}

// health check
func healthHandler(c *fiber.Ctx) error {
	redisDepth, err := queue.RedisQueueDepth(context.Background())
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status": "degraded",
			"error":  err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":             "ok",
		"buffer_depth":       queue.BufferDepth(),
		"buffer_capacity":    queue.BufferCapacity(),
		"redis_queue_depth":  redisDepth,
		"ingestion_counters": metrics.GetSnapshot(),
	})
}

func metricsHandler(c *fiber.Ctx) error {
	redisDepth, err := queue.RedisQueueDepth(context.Background())
	if err != nil {
		redisDepth = -1
	}

	return c.JSON(fiber.Map{
		"metrics": fiber.Map{
			"counters":          metrics.GetSnapshot(),
			"buffer_depth":      queue.BufferDepth(),
			"buffer_capacity":   queue.BufferCapacity(),
			"redis_queue_depth": redisDepth,
		},
	})
}

func handleLogin(c *fiber.Ctx, db *pgxpool.Pool, jwtSecret []byte) error {
	var req AuthRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(auth.ApiErr("bad_request", "invalid JSON body"))
	}

	var userID, passwordHash string
	err := db.QueryRow(context.Background(),
		`SELECT id, password_hash FROM users WHERE email = $1`,
		req.Email,
	).Scan(&userID, &passwordHash)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(auth.ApiErr("unauthorized", "invalid email or password"))
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(auth.ApiErr("unauthorized", "invalid email or password"))
	}

	rows, err := db.Query(context.Background(),
		`SELECT id FROM sites WHERE user_id = $1`,
		userID,
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(auth.ApiErr("server_error", "failed to fetch user sites"))
	}

	defer rows.Close()

	var siteIDs []string
	for rows.Next() {
		var siteID string
		if err := rows.Scan(&siteID); err != nil {
			return err
		}

		siteIDs = append(siteIDs, siteID)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	tokenString, err := auth.IssueToken(jwtSecret, userID, siteIDs, 1*time.Hour)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(auth.ApiErr("server_error", "failed to generate token"))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token": tokenString,
		"token_type":   "Bearer",
	})
}

func handleRegister(c *fiber.Ctx, db *pgxpool.Pool) error {
	var req RegisterRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(auth.ApiErr("bad_request", "invalid JSON body"))
	}

	if req.Email == "" || len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(auth.ApiErr("bad_request", "invalid email or password too short"))
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(auth.ApiErr("server_error", "failed to process password"))
	}

	_, err = db.Exec(context.Background(),
		`INSERT INTO users (email, password_hash) VALUES ($1, $2)`,
		req.Email, string(hashedPassword),
	)
	if err != nil {
		// In production, check if the error is a unique constraint violation (duplicate email)
		return c.Status(fiber.StatusConflict).JSON(auth.ApiErr("conflict", "email already in use"))
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User registered successfully",
	})
}

func handleIngest(c *fiber.Ctx) error {
	var payload analytics.Payload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "invalid_json",
		})
	}

	return handleNormalizedPayload(c, payload)
}

// Normalizes and validated the incoming event
func normalizeEvent(event *analytics.Event, clientIP, fallbackUserAgent string, now time.Time) error {
	if err := event.Normalize(now, clientIP, fallbackUserAgent, false); err != nil {
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			return fiberErr
		}
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return nil
}

// func to check the json is valid or not
// It parses the context into the Event struct using BodyParser
// If err -> the json schema is invalid and not as per the Event struct
func handleSingleEvent(c *fiber.Ctx) error {
	var event analytics.Event
	if err := c.BodyParser(&event); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "invalid_json",
		})
	}

	return handleNormalizedPayload(c, analytics.Payload{Events: []analytics.Event{event}})
}

func handleNormalizedPayload(c *fiber.Ctx, payload analytics.Payload) error {
	metrics.RecordRequest(len(payload.Events))

	if len(payload.Events) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "empty_batch",
		})
	}

	now := time.Now().UTC()
	clientIP := c.IP()
	userAgent := c.Get("User-Agent")

	encoded := make([][]byte, 0, len(payload.Events))
	rejected := make([]fiber.Map, 0)
	for i := range payload.Events {
		event := &payload.Events[i]
		if err := normalizeEvent(event, clientIP, userAgent, now); err != nil {
			rejected = append(rejected, fiber.Map{
				"index": i,
				"error": err.Error(),
			})
			continue
		}

		data, err := json.Marshal(event)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status": "marshal_failed",
			})
		}
		encoded = append(encoded, data)
	}

	if len(encoded) == 0 {
		metrics.RecordDropped(len(rejected))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":          "rejected",
			"accepted_events": 0,
			"rejected_events": len(rejected),
			"errors":          rejected,
		})
	}

	queued, ok := queue.EnqueueBatch(encoded)
	if !ok {
		metrics.RecordDropped(len(encoded))
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status":          "backpressure",
			"accepted_events": 0,
			"dropped_events":  len(encoded),
			"buffer_depth":    queue.BufferDepth(),
			"buffer_capacity": queue.BufferCapacity(),
			"errors":          rejected,
		})
	}

	metrics.RecordBuffered(queued)
	status := "accepted"
	if len(rejected) > 0 {
		status = "partially_invalid"
		metrics.RecordDropped(len(rejected))
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"status":          status,
		"accepted_events": queued,
		"rejected_events": len(rejected),
		"queue_status":    "queued",
		"event_rate_unit": "events",
		"errors":          rejected,
	})
}

func parseTimeRange(c *fiber.Ctx) (time.Time, time.Time, error) {
	fromStr := c.Query("from")
	toStr := c.Query("to")
	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return from, to, nil
}

func handleOverview(c *fiber.Ctx, db *pgxpool.Pool) error {
	siteID := c.Params("site_id")
	from, to, err := parseTimeRange(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid time range"})
	}
	interval := c.Query("interval", "day")
	if interval != "day" && interval != "hour" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid interval"})
	}

	result, err := query.GetOverview(c.Context(), db, siteID, from, to, interval)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func handleRealtime(c *fiber.Ctx, db *pgxpool.Pool) error {
	siteID := c.Params("site_id")
	result, err := query.GetRealtime(c.Context(), db, siteID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func handlePages(c *fiber.Ctx, db *pgxpool.Pool) error {
	siteID := c.Params("site_id")
	from, to, err := parseTimeRange(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid time range"})
	}
	limit := c.QueryInt("limit", 100)

	result, err := query.GetPages(c.Context(), db, siteID, from, to, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func handleSources(c *fiber.Ctx, db *pgxpool.Pool) error {
	siteID := c.Params("site_id")
	from, to, err := parseTimeRange(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid time range"})
	}
	limit := c.QueryInt("limit", 100)

	result, err := query.GetSources(c.Context(), db, siteID, from, to, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}
