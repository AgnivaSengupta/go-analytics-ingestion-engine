package rollups

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Builder struct {
	Name     string
	From     func(time.Time) time.Time
	To       func(time.Time) time.Time
	Exec     func(context.Context, *pgxpool.Pool, time.Time, time.Time) error
	Kind     string
	Schedule string
}

func AggregateBuilders() []Builder {
	return []Builder{
		{
			Name:     "agg_site_hourly",
			Kind:     "aggregate",
			Schedule: "*/5 * * * *",
			From: func(now time.Time) time.Time {
				return now.UTC().Add(-48 * time.Hour).Truncate(time.Hour)
			},
			To: func(now time.Time) time.Time {
				return now.UTC().Add(time.Hour).Truncate(time.Hour)
			},
			Exec: execHourlySiteAggregate,
		},
		{
			Name:     "agg_site_daily",
			Kind:     "aggregate",
			Schedule: "*/15 * * * *",
			From: func(now time.Time) time.Time {
				return truncateToDay(now.UTC().AddDate(0, 0, -14))
			},
			To: func(now time.Time) time.Time {
				return truncateToDay(now.UTC().AddDate(0, 0, 1))
			},
			Exec: execDailySiteAggregate,
		},
		{
			Name:     "agg_page_daily",
			Kind:     "aggregate",
			Schedule: "*/15 * * * *",
			From: func(now time.Time) time.Time {
				return truncateToDay(now.UTC().AddDate(0, 0, -14))
			},
			To: func(now time.Time) time.Time {
				return truncateToDay(now.UTC().AddDate(0, 0, 1))
			},
			Exec: execDailyPageAggregate,
		},
		{
			Name:     "agg_source_daily",
			Kind:     "aggregate",
			Schedule: "*/15 * * * *",
			From: func(now time.Time) time.Time {
				return truncateToDay(now.UTC().AddDate(0, 0, -14))
			},
			To: func(now time.Time) time.Time {
				return truncateToDay(now.UTC().AddDate(0, 0, 1))
			},
			Exec: execDailySourceAggregate,
		},
		{
			Name:     "agg_device_daily",
			Kind:     "aggregate",
			Schedule: "*/15 * * * *",
			From: func(now time.Time) time.Time {
				return truncateToDay(now.UTC().AddDate(0, 0, -14))
			},
			To: func(now time.Time) time.Time {
				return truncateToDay(now.UTC().AddDate(0, 0, 1))
			},
			Exec: execDailyDeviceAggregate,
		},
		{
			Name:     "agg_geo_daily",
			Kind:     "aggregate",
			Schedule: "*/15 * * * *",
			From: func(now time.Time) time.Time {
				return truncateToDay(now.UTC().AddDate(0, 0, -14))
			},
			To: func(now time.Time) time.Time {
				return truncateToDay(now.UTC().AddDate(0, 0, 1))
			},
			Exec: execDailyGeoAggregate,
		},
	}
}

func ReconciliationBuilders() []Builder {
	return []Builder{
		{
			Name:     "reconcile_site_event_totals",
			Kind:     "reconciliation",
			Schedule: "20,50 * * * *",
			From: func(now time.Time) time.Time {
				return truncateToDay(now.UTC().AddDate(0, 0, -14))
			},
			To: func(now time.Time) time.Time {
				return truncateToDay(now.UTC().AddDate(0, 0, 1))
			},
			Exec: execReconcileEventTotals,
		},
		{
			Name:     "reconcile_session_totals",
			Kind:     "reconciliation",
			Schedule: "20,50 * * * *",
			From: func(now time.Time) time.Time {
				return truncateToDay(now.UTC().AddDate(0, 0, -14))
			},
			To: func(now time.Time) time.Time {
				return truncateToDay(now.UTC().AddDate(0, 0, 1))
			},
			Exec: execReconcileSessionTotals,
		},
		{
			Name:     "reconcile_visitor_totals",
			Kind:     "reconciliation",
			Schedule: "20,50 * * * *",
			From: func(now time.Time) time.Time {
				return truncateToDay(now.UTC().AddDate(0, 0, -14))
			},
			To: func(now time.Time) time.Time {
				return truncateToDay(now.UTC().AddDate(0, 0, 1))
			},
			Exec: execReconcileVisitorTotals,
		},
	}
}

func CleanupBuilders() []Builder {
	return []Builder{
		{
			Name:     "cleanup_raw_events",
			Kind:     "cleanup",
			Schedule: "45 * * * *",
			From: func(now time.Time) time.Time {
				return now.UTC().AddDate(0, 0, -30)
			},
			To: func(now time.Time) time.Time {
				return now.UTC()
			},
			Exec: execCleanupRawEvents,
		},
	}
}

func AllBuilders() []Builder {
	builders := make([]Builder, 0, len(AggregateBuilders())+len(ReconciliationBuilders())+len(CleanupBuilders()))
	builders = append(builders, AggregateBuilders()...)
	builders = append(builders, ReconciliationBuilders()...)
	builders = append(builders, CleanupBuilders()...)
	return builders
}

func RunBuilder(ctx context.Context, db *pgxpool.Pool, builder Builder, from, to time.Time) error {
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return builder.Exec(runCtx, db, from.UTC(), to.UTC())
}

func WindowForDirtyRange(builder Builder, dirtyFrom, dirtyTo time.Time) (time.Time, time.Time) {
	dirtyFrom = dirtyFrom.UTC()
	dirtyTo = dirtyTo.UTC()

	switch builder.Name {
	case "agg_site_hourly":
		return dirtyFrom.Truncate(time.Hour), dirtyTo.Add(time.Hour).Truncate(time.Hour)
	default:
		return truncateToDay(dirtyFrom), truncateToDay(dirtyTo.Add(24 * time.Hour))
	}
}

func FindBuilder(name string) (Builder, bool) {
	for _, builder := range AllBuilders() {
		if builder.Name == name {
			return builder, true
		}
	}
	return Builder{}, false
}

func truncateToDay(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func execHourlySiteAggregate(ctx context.Context, db *pgxpool.Pool, from, to time.Time) error {
	return execSQL(ctx, db, "agg_site_hourly", `
		DELETE FROM agg_site_hourly a
		WHERE a.time_bucket >= date_trunc('hour', $1::timestamptz)
		  AND a.time_bucket < date_trunc('hour', $2::timestamptz)
		  AND NOT EXISTS (
			  SELECT 1
			  FROM events e
			  WHERE e.site_id = a.site_id
			    AND date_trunc('hour', e.occurred_at) = a.time_bucket
			    AND e.occurred_at >= $1
			    AND e.occurred_at < $2
		  );
		`, `
		INSERT INTO agg_site_hourly (time_bucket, site_id, events, pageviews, visitors, sessions)
		SELECT
			date_trunc('hour', occurred_at) AS time_bucket,
			site_id,
			COUNT(*) AS events,
			COUNT(*) FILTER (WHERE event_type = 'page' OR event_name = 'page_view') AS pageviews,
			COUNT(DISTINCT visitor_id) AS visitors,
			COUNT(DISTINCT session_id) AS sessions
		FROM events
		WHERE occurred_at >= $1
		  AND occurred_at < $2
		GROUP BY 1, 2
		ON CONFLICT (site_id, time_bucket) DO UPDATE SET
			events = EXCLUDED.events,
			pageviews = EXCLUDED.pageviews,
			visitors = EXCLUDED.visitors,
			sessions = EXCLUDED.sessions;
	`, from, to)
}

func execDailySiteAggregate(ctx context.Context, db *pgxpool.Pool, from, to time.Time) error {
	return execSQL(ctx, db, "agg_site_daily", `
		DELETE FROM agg_site_daily a
		WHERE a.day >= $1::date
		  AND a.day < $2::date
		  AND NOT EXISTS (
			  SELECT 1
			  FROM events e
			  WHERE e.site_id = a.site_id
			    AND e.occurred_at::date = a.day
			    AND e.occurred_at >= $1
			    AND e.occurred_at < $2
		  );
		`, `
		INSERT INTO agg_site_daily (day, site_id, events, pageviews, visitors, sessions)
		SELECT
			occurred_at::date AS day,
			site_id,
			COUNT(*) AS events,
			COUNT(*) FILTER (WHERE event_type = 'page' OR event_name = 'page_view') AS pageviews,
			COUNT(DISTINCT visitor_id) AS visitors,
			COUNT(DISTINCT session_id) AS sessions
		FROM events
		WHERE occurred_at >= $1
		  AND occurred_at < $2
		GROUP BY 1, 2
		ON CONFLICT (site_id, day) DO UPDATE SET
			events = EXCLUDED.events,
			pageviews = EXCLUDED.pageviews,
			visitors = EXCLUDED.visitors,
			sessions = EXCLUDED.sessions;
	`, from, to)
}

func execDailyPageAggregate(ctx context.Context, db *pgxpool.Pool, from, to time.Time) error {
	return execSQL(ctx, db, "agg_page_daily", `
		DELETE FROM agg_page_daily a
		WHERE a.day >= $1::date
		  AND a.day < $2::date
		  AND NOT EXISTS (
			  SELECT 1
			  FROM events e
			  WHERE e.site_id = a.site_id
			    AND e.occurred_at::date = a.day
			    AND e.page_path = a.page_path
			    AND e.page_url = a.page_url
			    AND e.occurred_at >= $1
			    AND e.occurred_at < $2
		  );
		`, `
		INSERT INTO agg_page_daily (day, site_id, page_path, page_url, events, pageviews, visitors, sessions)
		SELECT
			occurred_at::date AS day,
			site_id,
			page_path,
			page_url,
			COUNT(*) AS events,
			COUNT(*) FILTER (WHERE event_type = 'page' OR event_name = 'page_view') AS pageviews,
			COUNT(DISTINCT visitor_id) AS visitors,
			COUNT(DISTINCT session_id) AS sessions
		FROM events
		WHERE occurred_at >= $1
		  AND occurred_at < $2
		GROUP BY 1, 2, 3, 4
		ON CONFLICT (site_id, day, page_path, page_url) DO UPDATE SET
			events = EXCLUDED.events,
			pageviews = EXCLUDED.pageviews,
			visitors = EXCLUDED.visitors,
			sessions = EXCLUDED.sessions;
	`, from, to)
}

func execDailySourceAggregate(ctx context.Context, db *pgxpool.Pool, from, to time.Time) error {
	return execSQL(ctx, db, "agg_source_daily", `
		DELETE FROM agg_source_daily a
		WHERE a.day >= $1::date
		  AND a.day < $2::date
		  AND NOT EXISTS (
			  SELECT 1
			  FROM events e
			  WHERE e.site_id = a.site_id
			    AND e.occurred_at::date = a.day
			    AND COALESCE(NULLIF(e.context->>'source', ''), 'direct') = a.source
			    AND COALESCE(NULLIF(e.context->>'medium', ''), 'none') = a.medium
			    AND COALESCE(NULLIF(e.context->>'campaign', ''), '(none)') = a.campaign
			    AND COALESCE(NULLIF(e.context->>'referrer_host', ''), '(direct)') = a.referrer_host
			    AND e.occurred_at >= $1
			    AND e.occurred_at < $2
		  );
		`, `
		INSERT INTO agg_source_daily (day, site_id, source, medium, campaign, referrer_host, events, pageviews, visitors, sessions)
		SELECT
			occurred_at::date AS day,
			site_id,
			COALESCE(NULLIF(context->>'source', ''), 'direct') AS source,
			COALESCE(NULLIF(context->>'medium', ''), 'none') AS medium,
			COALESCE(NULLIF(context->>'campaign', ''), '(none)') AS campaign,
			COALESCE(NULLIF(context->>'referrer_host', ''), '(direct)') AS referrer_host,
			COUNT(*) AS events,
			COUNT(*) FILTER (WHERE event_type = 'page' OR event_name = 'page_view') AS pageviews,
			COUNT(DISTINCT visitor_id) AS visitors,
			COUNT(DISTINCT session_id) AS sessions
		FROM events
		WHERE occurred_at >= $1
		  AND occurred_at < $2
		GROUP BY 1, 2, 3, 4, 5, 6
		ON CONFLICT (site_id, day, source, medium, campaign, referrer_host) DO UPDATE SET
			events = EXCLUDED.events,
			pageviews = EXCLUDED.pageviews,
			visitors = EXCLUDED.visitors,
			sessions = EXCLUDED.sessions;
	`, from, to)
}

func execDailyDeviceAggregate(ctx context.Context, db *pgxpool.Pool, from, to time.Time) error {
	return execSQL(ctx, db, "agg_device_daily", `
		DELETE FROM agg_device_daily a
		WHERE a.day >= $1::date
		  AND a.day < $2::date
		  AND NOT EXISTS (
			  SELECT 1
			  FROM events e
			  WHERE e.site_id = a.site_id
			    AND e.occurred_at::date = a.day
			    AND COALESCE(NULLIF(e.context->>'device_type', ''), 'unknown') = a.device_type
			    AND COALESCE(NULLIF(e.context->>'os_name', ''), 'Unknown') = a.os_name
			    AND e.occurred_at >= $1
			    AND e.occurred_at < $2
		  );
		`, `
		INSERT INTO agg_device_daily (day, site_id, device_type, os_name, events, pageviews, visitors, sessions)
		SELECT
			occurred_at::date AS day,
			site_id,
			COALESCE(NULLIF(context->>'device_type', ''), 'unknown') AS device_type,
			COALESCE(NULLIF(context->>'os_name', ''), 'Unknown') AS os_name,
			COUNT(*) AS events,
			COUNT(*) FILTER (WHERE event_type = 'page' OR event_name = 'page_view') AS pageviews,
			COUNT(DISTINCT visitor_id) AS visitors,
			COUNT(DISTINCT session_id) AS sessions
		FROM events
		WHERE occurred_at >= $1
		  AND occurred_at < $2
		GROUP BY 1, 2, 3, 4
		ON CONFLICT (site_id, day, device_type, os_name) DO UPDATE SET
			events = EXCLUDED.events,
			pageviews = EXCLUDED.pageviews,
			visitors = EXCLUDED.visitors,
			sessions = EXCLUDED.sessions;
	`, from, to)
}

func execDailyGeoAggregate(ctx context.Context, db *pgxpool.Pool, from, to time.Time) error {
	return execSQL(ctx, db, "agg_geo_daily", `
		DELETE FROM agg_geo_daily a
		WHERE a.day >= $1::date
		  AND a.day < $2::date
		  AND NOT EXISTS (
			  SELECT 1
			  FROM events e
			  WHERE e.site_id = a.site_id
			    AND e.occurred_at::date = a.day
			    AND COALESCE(NULLIF(e.context->>'geo_country', ''), 'Unknown') = a.geo_country
			    AND e.occurred_at >= $1
			    AND e.occurred_at < $2
		  );
		`, `
		INSERT INTO agg_geo_daily (day, site_id, geo_country, events, pageviews, visitors, sessions)
		SELECT
			occurred_at::date AS day,
			site_id,
			COALESCE(NULLIF(context->>'geo_country', ''), 'Unknown') AS geo_country,
			COUNT(*) AS events,
			COUNT(*) FILTER (WHERE event_type = 'page' OR event_name = 'page_view') AS pageviews,
			COUNT(DISTINCT visitor_id) AS visitors,
			COUNT(DISTINCT session_id) AS sessions
		FROM events
		WHERE occurred_at >= $1
		  AND occurred_at < $2
		GROUP BY 1, 2, 3
		ON CONFLICT (site_id, day, geo_country) DO UPDATE SET
			events = EXCLUDED.events,
			pageviews = EXCLUDED.pageviews,
			visitors = EXCLUDED.visitors,
			sessions = EXCLUDED.sessions;
	`, from, to)
}

func execReconcileEventTotals(ctx context.Context, db *pgxpool.Pool, from, to time.Time) error {
	return writeReconciliation(ctx, db, "reconcile_site_event_totals", from, to, `
		WITH canonical AS (
			SELECT
				occurred_at::date AS day,
				site_id,
				COUNT(*) AS canonical_events
			FROM events
			WHERE occurred_at >= $1
			  AND occurred_at < $2
			GROUP BY 1, 2
		),
		rollup AS (
			SELECT
				day,
				site_id,
				SUM(events) AS aggregate_events
			FROM agg_site_daily
			WHERE day >= $1::date
			  AND day < $2::date
			GROUP BY 1, 2
		),
		mismatches AS (
			SELECT
				COALESCE(c.day, r.day) AS day,
				COALESCE(c.site_id, r.site_id) AS site_id,
				COALESCE(c.canonical_events, 0) AS canonical_events,
				COALESCE(r.aggregate_events, 0) AS aggregate_events
			FROM canonical c
			FULL OUTER JOIN rollup r
				ON c.day = r.day
			   AND c.site_id = r.site_id
			WHERE COALESCE(c.canonical_events, 0) <> COALESCE(r.aggregate_events, 0)
		)
		SELECT COUNT(*), COALESCE(jsonb_agg(to_jsonb(mismatches)), '[]'::jsonb)
		FROM mismatches;
	`)
}

func execReconcileSessionTotals(ctx context.Context, db *pgxpool.Pool, from, to time.Time) error {
	return writeReconciliation(ctx, db, "reconcile_session_totals", from, to, `
		WITH canonical AS (
			SELECT
				occurred_at::date AS day,
				site_id,
				COUNT(DISTINCT session_id) AS canonical_sessions
			FROM events
			WHERE occurred_at >= $1
			  AND occurred_at < $2
			GROUP BY 1, 2
		),
		rollup AS (
			SELECT
				day,
				site_id,
				SUM(sessions) AS aggregate_sessions
			FROM agg_site_daily
			WHERE day >= $1::date
			  AND day < $2::date
			GROUP BY 1, 2
		),
		mismatches AS (
			SELECT
				COALESCE(c.day, r.day) AS day,
				COALESCE(c.site_id, r.site_id) AS site_id,
				COALESCE(c.canonical_sessions, 0) AS canonical_sessions,
				COALESCE(r.aggregate_sessions, 0) AS aggregate_sessions
			FROM canonical c
			FULL OUTER JOIN rollup r
				ON c.day = r.day
			   AND c.site_id = r.site_id
			WHERE COALESCE(c.canonical_sessions, 0) <> COALESCE(r.aggregate_sessions, 0)
		)
		SELECT COUNT(*), COALESCE(jsonb_agg(to_jsonb(mismatches)), '[]'::jsonb)
		FROM mismatches;
	`)
}

func execReconcileVisitorTotals(ctx context.Context, db *pgxpool.Pool, from, to time.Time) error {
	return writeReconciliation(ctx, db, "reconcile_visitor_totals", from, to, `
		WITH canonical AS (
			SELECT
				occurred_at::date AS day,
				site_id,
				COUNT(DISTINCT visitor_id) AS canonical_visitors
			FROM events
			WHERE occurred_at >= $1
			  AND occurred_at < $2
			GROUP BY 1, 2
		),
		rollup AS (
			SELECT
				day,
				site_id,
				SUM(visitors) AS aggregate_visitors
			FROM agg_site_daily
			WHERE day >= $1::date
			  AND day < $2::date
			GROUP BY 1, 2
		),
		mismatches AS (
			SELECT
				COALESCE(c.day, r.day) AS day,
				COALESCE(c.site_id, r.site_id) AS site_id,
				COALESCE(c.canonical_visitors, 0) AS canonical_visitors,
				COALESCE(r.aggregate_visitors, 0) AS aggregate_visitors
			FROM canonical c
			FULL OUTER JOIN rollup r
				ON c.day = r.day
			   AND c.site_id = r.site_id
			WHERE COALESCE(c.canonical_visitors, 0) <> COALESCE(r.aggregate_visitors, 0)
		)
		SELECT COUNT(*), COALESCE(jsonb_agg(to_jsonb(mismatches)), '[]'::jsonb)
		FROM mismatches;
	`)
}

func execCleanupRawEvents(ctx context.Context, db *pgxpool.Pool, from, _ time.Time) error {
	_, err := db.Exec(ctx, `
		DELETE FROM raw_events
		WHERE received_at < $1
	`, from)
	return err
}

func writeReconciliation(ctx context.Context, db *pgxpool.Pool, jobName string, from, to time.Time, mismatchSQL string) error {
	var mismatchCount int64
	var details []byte

	if err := db.QueryRow(ctx, mismatchSQL, from, to).Scan(&mismatchCount, &details); err != nil {
		return err
	}

	status := "ok"
	if mismatchCount > 0 {
		status = "mismatch"
	}

	if !json.Valid(details) {
		details = []byte(`[]`)
	}

	_, err := db.Exec(ctx, `
		INSERT INTO reconciliation_runs (
			job_name,
			window_start,
			window_end,
			mismatch_count,
			status,
			details
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb)
	`, jobName, from, to, mismatchCount, status, string(details))

	return err
}

func execSQL(ctx context.Context, db *pgxpool.Pool, lockName, deleteSQL, insertSQL string, from, to time.Time) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, advisoryLockKey(lockName)); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, deleteSQL, from, to); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, insertSQL, from, to); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func advisoryLockKey(name string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(name))
	return int64(h.Sum64() & 0x7fffffffffffffff)
}

func BuilderNames(builders []Builder) []string {
	names := make([]string, 0, len(builders))
	for _, builder := range builders {
		names = append(names, builder.Name)
	}
	return names
}

func Describe(builders []Builder) string {
	names := BuilderNames(builders)
	return strings.Join(names, ", ")
}

func RequireBuilder(name string) (Builder, error) {
	builder, ok := FindBuilder(name)
	if !ok {
		return Builder{}, fmt.Errorf("unknown builder %q", name)
	}
	return builder, nil
}
