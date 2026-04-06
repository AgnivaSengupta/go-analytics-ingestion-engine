package queue

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/AgnivaSengupta/analytics-engine/internal/metrics"
)

const (
	BatchSize    = 500
	PushInterval = 100 * time.Millisecond
	BufferLimit  = 10000
)

var EventChan chan []byte
var enqueueMu sync.Mutex

func StartBatcher(ctx context.Context) {
	EventChan = make(chan []byte, BufferLimit)

	go func() {
		log.Println("Batcher started..")

		var batch []any
		ticker := time.NewTicker(PushInterval)
		defer ticker.Stop()

		for {
			select {
			case event := <-EventChan:
				batch = append(batch, event)

				if len(batch) >= BatchSize {
					flushToRedis(batch)
					batch = nil
				}

			case <-ticker.C:
				// 3. If time is up, push whatever we have
				if len(batch) > 0 {
					flushToRedis(batch)
					batch = nil // Clear batch
				}

			case <-ctx.Done():
				// Graceful shutdown: Flush remaining items
				if len(batch) > 0 {
					flushToRedis(batch)
				}
				return
			}
		}
	}()
}

func EnqueueBatch(events [][]byte) (int, bool) {
	if EventChan == nil {
		return 0, false
	}

	enqueueMu.Lock()
	defer enqueueMu.Unlock()

	if cap(EventChan)-len(EventChan) < len(events) {
		return 0, false
	}

	for _, event := range events {
		EventChan <- event
	}

	return len(events), true
}

func BufferDepth() int {
	if EventChan == nil {
		return 0
	}

	return len(EventChan)
}

func BufferCapacity() int {
	if EventChan == nil {
		return BufferLimit
	}

	return cap(EventChan)
}

func RedisQueueDepth(ctx context.Context) (int64, error) {
	if Client == nil {
		return 0, nil
	}

	return Client.LLen(ctx, "analytics_queue").Result()
}

func flushToRedis(batch []any) {
	var lastErr error

	for attempt := 1; attempt <= 3; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		lastErr = Client.RPush(ctx, "analytics_queue", batch...).Err()
		cancel()
		if lastErr == nil {
			metrics.RecordRedisFlushSuccess(len(batch))
			return
		}

		log.Printf("redis flush attempt %d failed: %v", attempt, lastErr)
		time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
	}

	metrics.RecordRedisFlushFailure(len(batch))
	log.Printf("❌ Failed to flush batch to Redis after retries: %v", lastErr)
}
