package queue

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)



var Client *redis.Client

func InitReddis() error {
	dsn := os.Getenv("REDIS_DSN")
	if dsn == "" {
		dsn = "redis://localhost:6379"
	}
	
	opt, err := redis.ParseURL(dsn)
	if err != nil {
		return err
	}
	
	Client = redis.NewClient(opt)
	
	for i:=0; i<5; i++ {
		_, err = Client.Ping(context.Background()).Result()
		if err == nil {
            fmt.Println("✅ Connected to Redis")
            return nil // Success!
        }
        
        fmt.Printf("⏳ Redis not ready yet... retrying (%d/5)\n", i+1)
        time.Sleep(2 * time.Second)
	}
	
	// Test connection
	
	return fmt.Errorf("Failed to connect to redis: %v", err)
	
	return nil
}