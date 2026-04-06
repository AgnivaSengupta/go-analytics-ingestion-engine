package query

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetPages(ctx context.Context, db *pgxpool.Pool, siteID string, from, to time.Time, limit int) (*PagesResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	res := &PagesResult{
		SiteID: siteID,
		Range: TimeRange{
			From: from,
			To:   to,
		},
		Pages: []TopPage{},
	}

	query := `
		WITH combined AS (
			SELECT page_path, page_url, pageviews, visitors, sessions
			FROM agg_page_daily
			WHERE site_id = $1 AND day >= $2::date AND day < $3::date
			UNION ALL
			SELECT page_path, page_url,
				COUNT(*) FILTER (WHERE event_type = 'page' OR event_name = 'page_view') AS pageviews,
				COUNT(DISTINCT visitor_id) AS visitors,
				COUNT(DISTINCT session_id) AS sessions
			FROM events
			WHERE site_id = $1 AND occurred_at >= $2 AND occurred_at < $3
			  AND occurred_at >= date_trunc('day', NOW())
			GROUP BY 1, 2
		)
		SELECT page_path, page_url, SUM(pageviews) as pv, SUM(visitors) as v, SUM(sessions) as s
		FROM combined
		GROUP BY 1, 2
		ORDER BY pv DESC
		LIMIT $4
	`

	rows, err := db.Query(ctx, query, siteID, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p TopPage
		if err := rows.Scan(&p.PagePath, &p.PageURL, &p.Pageviews, &p.Visitors, &p.EntryCount); err == nil {
			res.Pages = append(res.Pages, p)
		}
	}

	// In a real app we'd fetch accurate avg_time_seconds per page or calculate via lag window functions.
	// We'll leave it as 0 here.
	return res, nil
}
