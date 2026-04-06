package query

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetRealtime(ctx context.Context, db *pgxpool.Pool, siteID string) (*RealtimeResult, error) {
	res := &RealtimeResult{
		SiteID:        siteID,
		WindowMinutes: 30,
		TopPages:      []RealtimePage{},
		TopReferrers:  []RealtimeReferrer{},
		GeneratedAt:   time.Now().UTC(),
	}

	cutoff := time.Now().UTC().Add(-30 * time.Minute)

	// Quick stats
	statsQuery := `
		SELECT 
			COUNT(DISTINCT visitor_id),
			COUNT(*) FILTER (WHERE event_type = 'page' OR event_name = 'page_view'),
			COUNT(*)
		FROM events
		WHERE site_id = $1 AND occurred_at >= $2
	`
	_ = db.QueryRow(ctx, statsQuery, siteID, cutoff).Scan(
		&res.ActiveVisitors,
		&res.RecentPageviews,
		&res.RecentEvents,
	)

	// Top pages
	pagesQuery := `
		SELECT page_path, COUNT(*)
		FROM events
		WHERE site_id = $1 AND occurred_at >= $2 AND (event_type = 'page' OR event_name = 'page_view')
		GROUP BY 1
		ORDER BY 2 DESC
		LIMIT 10
	`
	pRows, _ := db.Query(ctx, pagesQuery, siteID, cutoff)
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
	rRows, _ := db.Query(ctx, refQuery, siteID, cutoff)
	defer rRows.Close()
	for rRows.Next() {
		var r RealtimeReferrer
		if err := rRows.Scan(&r.Referrer, &r.Events); err == nil {
			res.TopReferrers = append(res.TopReferrers, r)
		}
	}

	return res, nil
}
