package query

import "time"

type TimeRange struct {
	From     time.Time `json:"from"`
	To       time.Time `json:"to"`
	Interval string    `json:"interval,omitempty"`
}

type OverviewTotals struct {
	Pageviews      int64   `json:"pageviews"`
	Visitors       int64   `json:"visitors"`
	Sessions       int64   `json:"sessions"`
	BounceRate     float64 `json:"bounce_rate"`
	AvgTimeSeconds float64 `json:"avg_time_seconds"`
}

type OverviewTimeseries struct {
	BucketStart time.Time `json:"bucket_start"`
	Pageviews   int64     `json:"pageviews"`
	Visitors    int64     `json:"visitors"`
	Sessions    int64     `json:"sessions"`
}

type OverviewResult struct {
	SiteID     string               `json:"site_id"`
	Range      TimeRange            `json:"range"`
	Totals     OverviewTotals       `json:"totals"`
	Timeseries []OverviewTimeseries `json:"timeseries"`
}

type TopPage struct {
	PageURL        string  `json:"page_url"`
	PagePath       string  `json:"page_path"`
	Pageviews      int64   `json:"pageviews"`
	Visitors       int64   `json:"visitors"`
	AvgTimeSeconds float64 `json:"avg_time_seconds"`
	EntryCount     int64   `json:"entry_count"`
}

type PagesResult struct {
	SiteID     string    `json:"site_id"`
	Range      TimeRange `json:"range"`
	Pages      []TopPage `json:"pages"`
	NextCursor *string   `json:"next_cursor"`
}

type TopSource struct {
	Source   string `json:"source"`
	Medium   string `json:"medium"`
	Campaign string `json:"campaign"`
	Referrer string `json:"referrer"`
	Visitors int64  `json:"visitors"`
	Sessions int64  `json:"sessions"`
}

type SourcesResult struct {
	SiteID     string      `json:"site_id"`
	Range      TimeRange   `json:"range"`
	Sources    []TopSource `json:"sources"`
	NextCursor *string     `json:"next_cursor"`
}

type RealtimePage struct {
	PagePath  string `json:"page_path"`
	Pageviews int64  `json:"pageviews"`
}

type RealtimeReferrer struct {
	Referrer string `json:"referrer"`
	Events   int64  `json:"events"`
}

type RealtimeResult struct {
	SiteID          string             `json:"site_id"`
	WindowMinutes   int                `json:"window_minutes"`
	ActiveVisitors  int64              `json:"active_visitors"`
	RecentPageviews int64              `json:"recent_pageviews"`
	RecentEvents    int64              `json:"recent_events"`
	TopPages        []RealtimePage     `json:"top_pages"`
	TopReferrers    []RealtimeReferrer `json:"top_referrers"`
	GeneratedAt     time.Time          `json:"generated_at"`
}
