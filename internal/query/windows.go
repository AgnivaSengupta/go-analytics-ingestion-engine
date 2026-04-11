package query

import "time"

func liveWindowStart(interval string, now time.Time) time.Time {
	now = now.UTC()
	if interval == "hour" {
		return now.Truncate(time.Hour)
	}
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func aggregateWindowEnd(to, liveStart time.Time) time.Time {
	to = to.UTC()
	liveStart = liveStart.UTC()
	if to.Before(liveStart) {
		return to
	}
	return liveStart
}
