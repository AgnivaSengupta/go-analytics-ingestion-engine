package query

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetOverview(ctx context.Context, db *pgxpool.Pool, siteID string, from, to time.Time, interval string) (*OverviewResult, error) {
	result := &OverviewResult{
		SiteID: siteID,
		Range: TimeRange{
			From:     from,
			To:       to,
			Interval: interval,
		},
		Timeseries: []OverviewTimeseries{},
	}

	// Helper CTE to aggregate metrics.
	// For interval='day', we just use agg_site_daily and today's events.
	// For interval='hour', we use agg_site_hourly and recent events.
	var query string
	if interval == "day" {
		query = `
			WITH combined AS (
				SELECT day::timestamptz AS bucket_start, pageviews, visitors, sessions
				FROM agg_site_daily
				WHERE site_id = $1 AND day >= $2::date AND day < $3::date
				UNION ALL
				SELECT date_trunc('day', occurred_at) AS bucket_start,
					COUNT(*) FILTER (WHERE event_type = 'page' OR event_name = 'page_view') AS pageviews,
					COUNT(DISTINCT visitor_id) AS visitors,
					COUNT(DISTINCT session_id) AS sessions
				FROM events
				WHERE site_id = $1 AND occurred_at >= $2 AND occurred_at < $3
				  AND occurred_at >= date_trunc('day', NOW())
				GROUP BY 1
			)
			SELECT bucket_start, SUM(pageviews), SUM(visitors), SUM(sessions)
			FROM combined
			GROUP BY 1
			ORDER BY 1
		`
	} else {
		query = `
			WITH combined AS (
				SELECT time_bucket AS bucket_start, pageviews, visitors, sessions
				FROM agg_site_hourly
				WHERE site_id = $1 AND time_bucket >= date_trunc('hour', $2::timestamptz) AND time_bucket < $3
				UNION ALL
				SELECT date_trunc('hour', occurred_at) AS bucket_start,
					COUNT(*) FILTER (WHERE event_type = 'page' OR event_name = 'page_view') AS pageviews,
					COUNT(DISTINCT visitor_id) AS visitors,
					COUNT(DISTINCT session_id) AS sessions
				FROM events
				WHERE site_id = $1 AND occurred_at >= $2 AND occurred_at < $3
				  AND occurred_at >= date_trunc('hour', NOW())
				GROUP BY 1
			)
			SELECT bucket_start, SUM(pageviews), SUM(visitors), SUM(sessions)
			FROM combined
			GROUP BY 1
			ORDER BY 1
		`
	}

	rows, err := db.Query(ctx, query, siteID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ts OverviewTimeseries
		var pv, v, s *int64
		if err := rows.Scan(&ts.BucketStart, &pv, &v, &s); err != nil {
			return nil, err
		}
		if pv != nil { ts.Pageviews = *pv }
		if v != nil { ts.Visitors = *v }
		if s != nil { ts.Sessions = *s }
		result.Timeseries = append(result.Timeseries, ts)
		
		result.Totals.Pageviews += ts.Pageviews
		result.Totals.Visitors += ts.Visitors // note: accurate totals need count distinct, approximating here or via sessions
	}

	// For accurate overall totals & bounce rates
	totalsQuery := `
		SELECT 
			COUNT(DISTINCT visitor_id) as total_visitors,
			COUNT(*) as total_sessions,
			COALESCE(COUNT(*) FILTER (WHERE ended_at = started_at)::float / NULLIF(COUNT(*), 0), 0) as bounce_rate,
			COALESCE(EXTRACT(EPOCH FROM AVG(ended_at - started_at)), 0) as avg_time
		FROM sessions
		WHERE site_id = $1 AND started_at >= $2 AND started_at < $3
	`
	var tVis, tSess int64
	var bRate, avgTime float64
	err = db.QueryRow(ctx, totalsQuery, siteID, from, to).Scan(&tVis, &tSess, &bRate, &avgTime)
	if err == nil {
		result.Totals.Visitors = tVis
		result.Totals.Sessions = tSess
		result.Totals.BounceRate = bRate
		result.Totals.AvgTimeSeconds = avgTime
	}

	return result, nil
}
