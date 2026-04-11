package query

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func GetRealtime(ctx context.Context, db *pgxpool.Pool, siteID string) (*RealtimeResult, error) {
	return getRealtime(ctx, db, siteID, time.Now().UTC())
}

type realtimeQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func getRealtime(ctx context.Context, db realtimeQuerier, siteID string, now time.Time) (*RealtimeResult, error) {
	res := &RealtimeResult{
		SiteID:        siteID,
		WindowMinutes: 30,
		TopPages:      []RealtimePage{},
		TopReferrers:  []RealtimeReferrer{},
		GeneratedAt:   now.UTC(),
	}

	cutoff := now.UTC().Add(-30 * time.Minute)

	// Quick stats
	statsQuery := `
		SELECT 
			COUNT(DISTINCT visitor_id),
			COUNT(*) FILTER (WHERE event_type = 'page' OR event_name = 'page_view'),
			COUNT(*)
		FROM events
		WHERE site_id = $1 AND occurred_at >= $2
	`
	if err := db.QueryRow(ctx, statsQuery, siteID, cutoff).Scan(
		&res.ActiveVisitors,
		&res.RecentPageviews,
		&res.RecentEvents,
	); err != nil {
		return nil, err
	}

	// Top pages
	pagesQuery := `
		SELECT page_path, COUNT(*)
		FROM events
		WHERE site_id = $1 AND occurred_at >= $2 AND (event_type = 'page' OR event_name = 'page_view')
		GROUP BY 1
		ORDER BY 2 DESC
		LIMIT 10
	`
	pRows, err := db.Query(ctx, pagesQuery, siteID, cutoff)
	if err != nil {
		return nil, err
	}
	defer pRows.Close()
	for pRows.Next() {
		var p RealtimePage
		if err := pRows.Scan(&p.PagePath, &p.Pageviews); err == nil {
			res.TopPages = append(res.TopPages, p)
		}
	}

	// Top referrers
	refQuery := `
		SELECT referrer, COUNT(*)
		FROM events
		WHERE site_id = $1 AND occurred_at >= $2 AND referrer IS NOT NULL AND referrer != ''
		GROUP BY 1
		ORDER BY 2 DESC
		LIMIT 10
	`
	rRows, err := db.Query(ctx, refQuery, siteID, cutoff)
	if err != nil {
		return nil, err
	}
	defer rRows.Close()
	for rRows.Next() {
		var r RealtimeReferrer
		if err := rRows.Scan(&r.Referrer, &r.Events); err == nil {
			res.TopReferrers = append(res.TopReferrers, r)
		}
	}

	return res, nil
}
