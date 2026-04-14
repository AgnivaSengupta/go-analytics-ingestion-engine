package query

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetSources(ctx context.Context, db *pgxpool.Pool, siteID string, from, to time.Time, limit int) (*SourcesResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	liveStart := liveWindowStart("day", time.Now().UTC())
	aggregateTo := aggregateWindowEnd(to, liveStart)

	res := &SourcesResult{
		SiteID: siteID,
		Range: TimeRange{
			From: from,
			To:   to,
		},
		Sources: []TopSource{},
	}

	query := `
		WITH combined AS (
			SELECT source, medium, campaign, referrer_host, visitors, sessions
			FROM agg_source_daily
			WHERE site_id = $1 AND day >= $2::date AND day < $3::date
			UNION ALL
			SELECT 
				COALESCE(NULLIF(context->>'source', ''), 'direct') AS source,
				COALESCE(NULLIF(context->>'medium', ''), 'none') AS medium,
				COALESCE(NULLIF(context->>'campaign', ''), '(none)') AS campaign,
				COALESCE(NULLIF(context->>'referrer_host', ''), '(direct)') AS referrer_host,
				COUNT(DISTINCT visitor_id) AS visitors,
				COUNT(DISTINCT session_id) AS sessions
			FROM events
			WHERE site_id = $1 AND occurred_at >= $4 AND occurred_at < $5
			GROUP BY 1, 2, 3, 4
		)
		SELECT source, medium, campaign, referrer_host, SUM(visitors) as v, SUM(sessions) as s
		FROM combined
		GROUP BY 1, 2, 3, 4
		ORDER BY v DESC
		LIMIT $6
	`

	rows, err := db.Query(ctx, query, siteID, from, aggregateTo, liveStart, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var s TopSource
		if err := rows.Scan(&s.Source, &s.Medium, &s.Campaign, &s.Referrer, &s.Visitors, &s.Sessions); err == nil {
			res.Sources = append(res.Sources, s)
		}
	}

	return res, nil
}
