# Phase 17: Usage & Analytics

> Process, contracts, gates, architecture: see `docs/phases/STAGE-13-19-PROCESS.md`.

## Goal
Backend-bucketed time-series chart data and bulk connection quota actions.

Resolves known debt: current UI buckets client-side over the loaded
(paginated) log window only — backend aggregation fixes silent truncation.

## Features (4 backend)
1. Time-series chart API (day + hour granularity)
2. Period filtering (today/24h/7d/30d/60d)
3. Bulk disable depleted connections
4. Bulk enable available connections

UI items (countdown auto-refresh, sort, filters, pagination) are Lovable's job
on top of existing `/api/quota` + these endpoints.

## New API Endpoints
- `GET /api/usage/chart?period=7d&granularity=day`
  - `period`: `today|24h|7d|30d|60d` (required); `granularity`: `hour|day`
    (default: `hour` for today/24h, `day` otherwise)
  - Returns:
    ```json
    {"buckets": ["2026-06-01", ...], "requests": [..], "tokens_input": [..],
     "tokens_output": [..], "costs": [..]}
    ```
    Arrays index-aligned with `buckets`; empty buckets zero-filled in Go.
- `POST /api/connections/bulk-disable` — `{threshold_percent?: 5}` disable connections at/below remaining-quota threshold; returns `{affected: [ids]}`
- `POST /api/connections/bulk-enable` — enable inactive connections with remaining quota; returns `{affected: [ids]}`

Both bulk endpoints audited with affected IDs in details.

## Aggregation Implementation
- Source table is **`request_log`** (NOT "usage_logs" — that table does not exist).
- SQLite only — use `strftime`:
  - day: `strftime('%Y-%m-%d', created_at)`
  - hour: `strftime('%Y-%m-%dT%H:00', created_at)`
  - (`date_trunc` is Postgres; do not use.)
- Single GROUP BY query per request; zero-fill gaps in Go.
- Read existing usage summary handler first to reuse period parsing.

## Tasks
1. `phase-17/task-1`: store — chart aggregation query + zero-fill + tests
2. `phase-17/task-2`: handler — `/api/usage/chart` param validation + tests
3. `phase-17/task-3`: store + handlers — bulk disable/enable + tests
4. `phase-17/checkpoint`

## Test Requirements (minimum)
- Seeded request_log rows across days → correct bucket sums (requests, tokens in/out, cost)
- Gap days zero-filled; arrays aligned with buckets
- Hour granularity for 24h period; invalid period/granularity → 400
- Bulk-disable only touches connections at/below threshold; returns affected ids; audit row written
- Bulk-enable skips connections without remaining quota
- Empty table → all-zero series, not error

## Commit Message (final)
`phase-17/usage-analytics: backend chart buckets, bulk quota actions`
