package analytics

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type Payload struct {
	Events []Event `json:"events"`
}

type Event struct {
	EventID       string         `json:"event_id"`
	SiteID        string         `json:"site_id"`
	VisitorID     string         `json:"visitor_id"`
	SessionID     string         `json:"session_id"`
	EventName     string         `json:"event_name"`
	EventType     string         `json:"event_type"`
	OccurredAt    string         `json:"occurred_at"`
	ReceivedAt    string         `json:"received_at,omitempty"`
	PageURL       string         `json:"page_url"`
	PagePath      string         `json:"page_path"`
	Referrer      string         `json:"referrer,omitempty"`
	UserAgent     string         `json:"user_agent,omitempty"`
	IPAddress     string         `json:"ip_address,omitempty"`
	SchemaVersion int            `json:"schema_version,omitempty"`
	Properties    map[string]any `json:"properties,omitempty"`
	Context       map[string]any `json:"context,omitempty"`
}

func (e *Event) Normalize(now time.Time, fallbackIP, fallbackUA string, allowGeneratedEventID bool) error {
	now = now.UTC()

	e.EventID = strings.TrimSpace(e.EventID)
	e.SiteID = strings.TrimSpace(e.SiteID)
	e.VisitorID = strings.TrimSpace(e.VisitorID)
	e.SessionID = strings.TrimSpace(e.SessionID)
	e.EventName = strings.TrimSpace(e.EventName)
	e.EventType = strings.TrimSpace(e.EventType)
	e.OccurredAt = strings.TrimSpace(e.OccurredAt)
	e.PageURL = strings.TrimSpace(e.PageURL)
	e.PagePath = strings.TrimSpace(e.PagePath)
	e.Referrer = strings.TrimSpace(e.Referrer)
	e.UserAgent = strings.TrimSpace(e.UserAgent)
	e.IPAddress = strings.TrimSpace(e.IPAddress)

	if e.SchemaVersion == 0 {
		e.SchemaVersion = 1
	}

	if e.Properties == nil {
		e.Properties = map[string]any{}
	}
	if e.Context == nil {
		e.Context = map[string]any{}
	}

	if e.ReceivedAt == "" {
		e.ReceivedAt = now.Format(time.RFC3339)
	}

	if e.IPAddress == "" {
		e.IPAddress = strings.TrimSpace(fallbackIP)
	}

	if e.UserAgent == "" {
		e.UserAgent = strings.TrimSpace(fallbackUA)
	}

	if e.EventID == "" && allowGeneratedEventID {
		id, err := generateEventID()
		if err != nil {
			return fmt.Errorf("generate event_id: %w", err)
		}
		e.EventID = id
	}

	return e.Validate()
}

func (e *Event) Validate() error {
	required := map[string]string{
		"event_id":    e.EventID,
		"site_id":     e.SiteID,
		"visitor_id":  e.VisitorID,
		"session_id":  e.SessionID,
		"event_name":  e.EventName,
		"event_type":  e.EventType,
		"occurred_at": e.OccurredAt,
		"page_url":    e.PageURL,
		"page_path":   e.PagePath,
	}

	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}

	if _, err := parseRFC3339UTC(e.OccurredAt); err != nil {
		return fmt.Errorf("occurred_at must be a valid RFC3339 timestamp: %w", err)
	}

	if _, err := parseRFC3339UTC(e.ReceivedAt); err != nil {
		return fmt.Errorf("received_at must be a valid RFC3339 timestamp: %w", err)
	}

	u, err := url.Parse(e.PageURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("page_url must be a valid absolute URL")
	}

	if !strings.HasPrefix(e.PagePath, "/") {
		return fmt.Errorf("page_path must start with '/'")
	}

	if e.Referrer != "" {
		ref, err := url.Parse(e.Referrer)
		if err != nil || ref.Scheme == "" || ref.Host == "" {
			return fmt.Errorf("referrer must be a valid absolute URL when provided")
		}
	}

	if e.SchemaVersion < 1 {
		return fmt.Errorf("schema_version must be >= 1")
	}

	return nil
}

func (e Event) OccurredAtTime() (time.Time, error) {
	return parseRFC3339UTC(e.OccurredAt)
}

func (e Event) ReceivedAtTime() (time.Time, error) {
	return parseRFC3339UTC(e.ReceivedAt)
}

func parseRFC3339UTC(value string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

func generateEventID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "evt_" + hex.EncodeToString(b[:]), nil
}

// func NormalizeTimestampMillis(ts int64, now time.Time) int64 {
// 	if ts <= 0 {
// 		return now.UnixMilli()
// 	}

// 	// Older clients may still send seconds; normalize them to milliseconds.
// 	if ts < 1_000_000_000_000 {
// 		return ts * 1000
// 	}

// 	return ts
// }

// func EventTimeFromMillis(ts int64, fallback time.Time) time.Time {
// 	if ts <= 0 {
// 		return fallback.UTC()
// 	}

// 	return time.UnixMilli(ts).UTC()
// }
