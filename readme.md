# Analytics Engine

High-throughput event ingestion and analytics pipeline built with Go, Redis, PostgreSQL, and Fiber.

## Overview

The engine accepts analytics events over HTTP, buffers them through Redis, and persists them to PostgreSQL via asynchronous workers. A cron service maintains rolling aggregate tables that power the query API. Authentication is split between ingest (site-scoped public API keys) and query (JWT-based owner/member access).

## Services

| Binary | Description |
|--------|-------------|
| `api` | HTTP ingestion and query service. Accepts events, queues them to Redis, and serves the query API. |
| `worker` | Consumes events from Redis, validates and enriches them, persists raw and canonical facts, and maintains visitor/session state. |
| `cron` | Rebuilds sliding canonical aggregates on a schedule and writes reconciliation results. |
| `migrate` | Applies versioned SQL migrations from `infra/migrations/` in order. |
| `backfill` | One-shot tool to manually rebuild aggregate windows from canonical events. |

## Project Structure

```
cmd/
  api/       - Ingestion and query HTTP server
  worker/    - Event processing consumer
  cron/      - Aggregate rollup scheduler
  migrate/   - Migration runner
  backfill/  - Aggregate backfill tool

internal/
  analytics/ - Event normalization and payload types
  auth/      - JWT issuance, ingest middleware, query middleware
  metrics/   - In-process ingestion counters
  query/     - Query API handlers (overview, realtime, pages, sources)
  queue/     - Redis client, batcher, and enqueue logic
  rollups/   - Aggregate build logic shared by cron and backfill

infra/
  migrations/ - Versioned SQL migration files
  haproxy/    - HAProxy config for API load balancing
```

## Database Schema

Migrations are applied in order by the `migrate` service:

| Migration | Description |
|-----------|-------------|
| `000001` | Legacy analytics schema (`analytics_events`, legacy rollup tables) |
| `000002` | Canonical event tables (`raw_events`, `events`) |
| `000003` | Identity and dead-letter tables (`visitors`, `sessions`, `dead_letter_events`) |
| `000004` | Canonical aggregate tables (`agg_site_hourly`, `agg_site_daily`, `agg_page_daily`, `agg_source_daily`) |
| `000005` | Aggregate work queue |
| `000006` | Auth tables (`users`, `sites`, `api_keys`) |

## API

The HTTP API runs on port `8080`. See [`api-spec.md`](api-spec.md) for the full specification.

### Authentication

**Ingest** endpoints require a site-scoped public API key passed as a Bearer token. The key is hashed and validated against the `api_keys` table.

**Query** endpoints require a JWT issued at login. The token encodes the user's accessible site IDs and is validated against the `JWT_SECRET` environment variable.

### Endpoints

**Auth**
```
POST /v1/auth/register
POST /v1/auth/login
```

**Ingest** — requires ingest API key
```
POST /v1/ingest          Batch ingest (array of events)
POST /v1/events          Single event ingest
```

**Query** — requires JWT
```
GET /v1/sites/:site_id/overview    Top-line metrics and time series
GET /v1/sites/:site_id/realtime    Near-realtime activity (last 30 minutes)
GET /v1/sites/:site_id/pages       Top pages for a date range
GET /v1/sites/:site_id/sources     Traffic source breakdown
```

**Operational**
```
GET /health    Service health and queue depth
GET /metrics   Ingestion counters and buffer state
```

### Event Schema

Events follow the canonical v1 schema. Required fields:

```json
{
  "event_id":   "evt_123",
  "site_id":    "site_abc",
  "visitor_id": "vis_1",
  "session_id": "sess_1",
  "event_name": "page_view",
  "event_type": "page",
  "occurred_at": "2026-04-04T12:30:00Z",
  "page_url":   "https://example.com/posts/hello",
  "page_path":  "/posts/hello"
}
```

Product-specific fields (`post_id`, `author_id`, etc.) go inside a `properties` object. `occurred_at` must be RFC3339 UTC.

## Local Setup

Copy the example environment file and adjust values as needed:

```bash
cp .env.example .env
```

**Environment variables:**

| Variable | Description |
|----------|-------------|
| `DB_DSN` | PostgreSQL connection string |
| `REDIS_DSN` | Redis connection string |
| `JWT_SECRET` | Secret used to sign and verify JWTs |

Start everything with Docker Compose:

```bash
docker compose up --build
```

Run migrations only:

```bash
docker compose run --rm migrate
```

## Backfill

Rebuild all aggregate windows for a time range:

```bash
go run ./cmd/backfill -from 2026-04-01T00:00:00Z -to 2026-04-08T00:00:00Z
```

Rebuild specific aggregate builders only:

```bash
go run ./cmd/backfill -builders agg_site_daily,agg_page_daily -from 2026-04-01T00:00:00Z -to 2026-04-08T00:00:00Z
```

## Operational Notes

- Raw events are retained for 24 hours. Range queries beyond that depend on aggregate tables.
- A `202 Accepted` response from the ingest endpoint means the event was queued; persistence depends on the worker keeping up with queue drain.
- Dead-lettered events (validation failures) are written to `dead_letter_events` and are not retried automatically.
- Duplicate events with the same `(site_id, event_id)` pair are deduplicated by the worker before affecting analytics counts.
