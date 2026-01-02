package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/AgnivaSengupta/analytics-engine/internal/queue"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
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
	// initializing the redis client
	if err := queue.InitReddis(); err != nil {
		log.Fatalf("Could not connect to redis: %v", err)
	}
	defer queue.Client.Close()

	// fiber app
	app := fiber.New(fiber.Config{
		AppName:   "Analytics_Ingestion_Engine",
		BodyLimit: 4 * 1024 * 1024,
	})

	// middleware
	app.Use(logger.New())
	app.Use(recover.New())

	app.Use(cors.New(cors.Config{
		AllowOrigins: "https://your-blog.com, http://localhost:5173",
		AllowMethods: "GET,POST,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	// Routes
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	app.Post("/api/ingest", handleIngest)

	log.Println("Ingestion Service running on port 8080")
	// Listen on 8080 to match your Dockerfile
	log.Fatal(app.Listen(":8080"))
}

func handleIngest(c *fiber.Ctx) error {
	event := new(AnalyticsEvent)

	if err := c.BodyParser(event); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid JSON")
	}

	// 2. Basic Validation
	if event.BlogID == "" || event.Url == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Missing required fields")
	}

	// 3. Enrich Data
	event.Timestamp = time.Now().UTC()
	event.UserAgent = c.Get("User-Agent")
	event.IPAddress = c.IP()

	data, err := json.Marshal(event)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).SendString("Internal Error")
	}

	// else push to redis
	err = queue.Client.RPush(context.Background(), "analytics_queue", data).Err()
	if err != nil {
		log.Printf("Redis error: %v", err)
		return c.Status(fiber.StatusServiceUnavailable).SendString("Service Unavailable")
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"status": "queued",
	})

}
