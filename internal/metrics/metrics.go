package metrics

import "sync/atomic"

type Snapshot struct {
	RequestsTotal          int64 `json:"requests_total"`
	EventsReceivedTotal    int64 `json:"events_received_total"`
	EventsBufferedTotal    int64 `json:"events_buffered_total"`
	EventsDroppedTotal     int64 `json:"events_dropped_total"`
	RedisFlushSuccessTotal int64 `json:"redis_flush_success_total"`
	RedisFlushFailureTotal int64 `json:"redis_flush_failure_total"`
	RedisEventsPushedTotal int64 `json:"redis_events_pushed_total"`
	WorkerEventsReadTotal  int64 `json:"worker_events_read_total"`
	WorkerEventsInserted   int64 `json:"worker_events_inserted_total"`
	WorkerInsertFailures   int64 `json:"worker_insert_failure_total"`
	RollupRunsTotal        int64 `json:"rollup_runs_total"`
	RollupFailureTotal     int64 `json:"rollup_failure_total"`
}

var (
	requestsTotal             atomic.Int64
	eventsReceivedTotal       atomic.Int64
	eventsBufferedTotal       atomic.Int64
	eventsDroppedTotal        atomic.Int64
	redisFlushSuccessTotal    atomic.Int64
	redisFlushFailureTotal    atomic.Int64
	redisEventsPushedTotal    atomic.Int64
	workerEventsReadTotal     atomic.Int64
	workerEventsInsertedTotal atomic.Int64
	workerInsertFailureTotal  atomic.Int64
	rollupRunsTotal           atomic.Int64
	rollupFailureTotal        atomic.Int64
)

func RecordRequest(events int) {
	requestsTotal.Add(1)
	eventsReceivedTotal.Add(int64(events))
}

func RecordBuffered(count int) {
	eventsBufferedTotal.Add(int64(count))
}

func RecordDropped(count int) {
	eventsDroppedTotal.Add(int64(count))
}

func RecordRedisFlushSuccess(events int) {
	redisFlushSuccessTotal.Add(1)
	redisEventsPushedTotal.Add(int64(events))
}

func RecordRedisFlushFailure(events int) {
	redisFlushFailureTotal.Add(1)
	eventsDroppedTotal.Add(int64(events))
}

func RecordWorkerRead(count int) {
	workerEventsReadTotal.Add(int64(count))
}

func RecordWorkerInsertSuccess(count int) {
	workerEventsInsertedTotal.Add(int64(count))
}

func RecordWorkerInsertFailure(count int) {
	workerInsertFailureTotal.Add(int64(count))
}

func RecordRollupRun(success bool) {
	if success {
		rollupRunsTotal.Add(1)
		return
	}

	rollupFailureTotal.Add(1)
}

func GetSnapshot() Snapshot {
	return Snapshot{
		RequestsTotal:          requestsTotal.Load(),
		EventsReceivedTotal:    eventsReceivedTotal.Load(),
		EventsBufferedTotal:    eventsBufferedTotal.Load(),
		EventsDroppedTotal:     eventsDroppedTotal.Load(),
		RedisFlushSuccessTotal: redisFlushSuccessTotal.Load(),
		RedisFlushFailureTotal: redisFlushFailureTotal.Load(),
		RedisEventsPushedTotal: redisEventsPushedTotal.Load(),
		WorkerEventsReadTotal:  workerEventsReadTotal.Load(),
		WorkerEventsInserted:   workerEventsInsertedTotal.Load(),
		WorkerInsertFailures:   workerInsertFailureTotal.Load(),
		RollupRunsTotal:        rollupRunsTotal.Load(),
		RollupFailureTotal:     rollupFailureTotal.Load(),
	}
}
