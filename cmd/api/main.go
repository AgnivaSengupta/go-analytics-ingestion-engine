package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/jackc/pgx/v5/pgconn"
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

type CreateApiKeyRequest struct {
	Name   string `json:"name"`
	SiteID string `json:"site_id"`
}

type RevokeApiKeyRequest struct {
	KeyID string `json:"key_id"`
}

type ApiKeyResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SiteID    string    `json:"site_id"`
	KeyType   string    `json:"key_type"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateSiteRequest struct {
	Name string `json:"name"`
}

type SiteResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
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

	app.Static("/tracker.js", "./sdk/tracker/tracker.js")
	
	app.Get("/health", healthHandler)
	app.Get("/metrics", metricsHandler)
	// app.Post("/api/ingest", handleIngest)

	// auth api
	app.Post("/v1/auth/login", func(c *fiber.Ctx) error { return handleLogin(c, db, jwtSecret) })
	app.Post("/v1/auth/register", func(c *fiber.Ctx) error { return handleRegister(c, db) })
	app.Post("/v1/auth/refresh", func(c *fiber.Ctx) error { return handleRefresh(c, db, jwtSecret) })
	app.Post("/v1/auth/logout", func(c *fiber.Ctx) error { return handleLogout(c, db) } )

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

	// api_key management
	sites.Post("/create-api-key", func(c *fiber.Ctx) error { return handleCreateApiKey(c, db) })
	sites.Get("/api-keys", func(c *fiber.Ctx) error { return handleGetAllApiKeys(c, db) })
	sites.Put("/revoke-api-key", func(c *fiber.Ctx) error { return handleRevokeApiKey(c, db) })

	// site management
	sites.Post("/", func(c *fiber.Ctx) error { return handleCreateSite(c, db) })
	sites.Get("/", func(c *fiber.Ctx) error { return handleListSites(c, db) })
	sites.Delete("/:id", func(c *fiber.Ctx) error { return handleDeleteSite(c, db) })

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

func handleRefresh(c *fiber.Ctx, db *pgxpool.Pool, jwtSecret []byte) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json body"})
	}

	tokenHash := hashAPIKey(req.RefreshToken)
	var userID string
	var expiresAt time.Time

	err := db.QueryRow(context.Background(),
		`SELECT user_id, expires_at FROM refresh_tokens WHERE token_hash = $1 AND revoked_at IS NULL`,
		tokenHash,
	).Scan(&userID, &expiresAt)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or revoked refresh token"})
	}

	if time.Now().After(expiresAt) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "refresh token expired, please log in again"})
	}

	rows, err := db.Query(context.Background(), `SELECT id FROM sites WHERE user_id = $1`, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error" : "failed to fetch sites from the db"})
	}
	
	defer rows.Close()
	var siteIDs []string
	for rows.Next() {
		var siteID string
		rows.Scan(&siteID)
		siteIDs = append(siteIDs, siteID)
	}

	// new access token
	newAccessToken, _ := auth.IssueToken(jwtSecret, userID, siteIDs, 15*time.Minute)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token": newAccessToken,
		"token_type":   "Bearer",
	})
}

func handleLogout(c *fiber.Ctx, db *pgxpool.Pool) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json body"})
	}

	tokenHash := hashAPIKey(req.RefreshToken)
	_, err := db.Exec(context.Background(),
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL`,
		tokenHash,
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database error"})
	}

	// We return 200 OK even if the token wasn't found (idempotent logout)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "logged out successfully",
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

	// access token
	tokenString, err := auth.IssueToken(jwtSecret, userID, siteIDs, 15*time.Minute)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(auth.ApiErr("server_error", "failed to generate access token"))
	}

	// refresh token
	refreshToken, err := generateApiKey()

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(auth.ApiErr("server_error", "failed to generate refresh token"))
	}

	refreshTokenHash := hashAPIKey(refreshToken)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	_, err = db.Exec(context.Background(),
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, refreshTokenHash, expiresAt,
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to insert refresh token"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token":  tokenString,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
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

func generateApiKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func hashAPIKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func handleCreateApiKey(c *fiber.Ctx, db *pgxpool.Pool) error {
	claims, ok := c.Locals("jwt_claims").(auth.Claims)

	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized-invalid claims"})
	}

	userId := claims.UserID
	if userId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized- Missing user ID"})
	}

	var req CreateApiKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON body"})
	}

	if req.SiteID == "" || req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name and site_id are required"})
	}

	var hasAccess bool
	if err := db.QueryRow(
		context.Background(),
		`SELECT EXISTS(SELECT 1 FROM sites WHERE id = $1 AND user_id = $2)`,
		req.SiteID,
		userId,
	).Scan(&hasAccess); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database error"})
	}
	if !hasAccess {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "site does not belong to the authenticated user"})
	}

	keyStr, err := generateApiKey()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate API key"})
	}
	keyHash := hashAPIKey(keyStr)

	var keyID string
	err = db.QueryRow(context.Background(),
		`INSERT INTO api_keys (site_id, user_id, key_hash, name, created_at, revoked_at) VALUES ($1, $2, $3, $4, Now(), NULL) RETURNING id`,
		req.SiteID, userId, keyHash, req.Name,
	).Scan(&keyID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "an active API key already exists for this site and user",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database error"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":      keyID,
		"name":    req.Name,
		"api_key": keyStr,
		"message": "Please save this API key now. You will not be able to see it again.",
	})
}

func handleGetAllApiKeys(c *fiber.Ctx, db *pgxpool.Pool) error {
	claims, ok := c.Locals("jwt_claims").(auth.Claims)

	if !ok || claims.UserID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	userId := claims.UserID
	rows, err := db.Query(context.Background(),
		`SELECT id, name, site_id, key_type, created_at FROM api_keys WHERE user_id = $1 AND revoked_at IS NULL`,
		userId,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch API keys"})
	}

	defer rows.Close()

	var keys []ApiKeyResponse
	for rows.Next() {
		var k ApiKeyResponse
		if err := rows.Scan(&k.ID, &k.Name, &k.SiteID, &k.KeyType, &k.CreatedAt); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database scan error"})
		}

		keys = append(keys, k)
	}

	if err := rows.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database row error"})
	}

	if keys == nil {
		keys = []ApiKeyResponse{}
	}

	return c.JSON(fiber.Map{
		"api_keys": keys,
	})
}

func handleRevokeApiKey(c *fiber.Ctx, db *pgxpool.Pool) error {
	claims, ok := c.Locals("jwt_claims").(auth.Claims)
	if !ok || claims.UserID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	userId := claims.UserID

	var req RevokeApiKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON body"})
	}

	if req.KeyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "key_id is required"})
	}

	command, err := db.Exec(context.Background(),
		`UPDATE api_keys SET revoked_at = NOW() WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`,
		req.KeyID, userId,
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to revoke API key"})
	}

	if command.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "API key not found or already revoked"})
	}

	return c.JSON(fiber.Map{
		"message": "API key successfully revoked",
	})
}

func handleCreateSite(c *fiber.Ctx, db *pgxpool.Pool) error {
	claims, ok := c.Locals("jwt_claims").(auth.Claims)
	if !ok || claims.UserID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthoried"})
	}

	var req CreateSiteRequest
	if err := c.BodyParser(&req); err != nil {
		c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON body"})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "site name is required"})
	}

	var site SiteResponse

	err := db.QueryRow(context.Background(),
		`INSERT INTO sites (name, user_id, created_at) VALUES ($1, $2, NOW()) RETURNING id, name, created_at`,
		req.Name, claims.UserID,
	).Scan(&site.ID, &site.Name, &site.CreatedAt)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create site"})
	}

	return c.Status(fiber.StatusCreated).JSON(site)
}

func handleListSites(c *fiber.Ctx, db *pgxpool.Pool) error {
	claims, ok := c.Locals("jwt_claims").(auth.Claims)
	if !ok || claims.UserID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	rows, err := db.Query(context.Background(),
		`SELECT id, name, created_at FROM sites WHERE user_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`,
		claims.UserID,
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch sites"})
	}

	defer rows.Close()

	var sites []SiteResponse
	for rows.Next() {
		var s SiteResponse

		if err := rows.Scan(&s.ID, &s.Name, &s.CreatedAt); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database scan error"})
		}

		sites = append(sites, s)
	}

	if err := rows.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database row error"})
	}

	if sites == nil {
		sites = []SiteResponse{}
	}

	return c.JSON(fiber.Map{
		"sites": sites,
	})
}

func handleDeleteSite(c *fiber.Ctx, db *pgxpool.Pool) error {
	claims, ok := c.Locals("jwt_claims").(auth.Claims)
	if !ok || claims.UserID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	siteID := c.Params("id")
	if siteID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "site ID is required"})
	}

	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "failed to start the transaction"})
	}

	defer tx.Rollback(ctx)
	// soft delete
	commandTag, err := tx.Exec(context.Background(),
		`UPDATE sites SET deleted_at = NOW() WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		siteID, claims.UserID,
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete site"})
	}

	if commandTag.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "site not found or already deleted"})
	}

	// 3. Revoke all active API keys associated with this site
	_, err = tx.Exec(ctx,
		`UPDATE api_keys SET revoked_at = NOW() WHERE site_id = $1 AND user_id = $2 AND revoked_at IS NULL`,
		siteID, claims.UserID,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to revoke associated API keys"})
	}

	// 4. Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to finalize deletion"})
	}

	return c.SendStatus(fiber.StatusNoContent)

}
