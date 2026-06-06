# Architect Plan — Phase 17: Usage & Analytics

Canonical spec: [`docs/phases/phase-17-usage-analytics.md`](../../phases/phase-17-usage-analytics.md)

## Summary

- `GET /api/usage/chart?period&granularity` returns `{buckets, requests, tokens_input, tokens_output, costs}` — arrays index-aligned with `buckets`, empty buckets zero-filled in Go, wrapped in the `{data, error}` envelope.
- `period`: `today|24h|7d|30d|60d` (required). `granularity`: `hour|day` (default `hour` for today/24h, `day` otherwise). Invalid period/granularity → `400`. Reuse the existing usage-summary period parsing.
- Aggregation source is the `request_log` table. SQLite `strftime` bucketing: day `strftime('%Y-%m-%d', created_at)`, hour `strftime('%Y-%m-%dT%H:00', created_at)`. Never `date_trunc`. One GROUP BY query per request; gaps zero-filled in Go.
- `POST /api/connections/bulk-disable` — `{threshold_percent?: 5}` disables connections at/below the remaining-quota threshold; returns `{affected: [ids]}`. Empty table / no matches → empty `affected`, not an error.
- `POST /api/connections/bulk-enable` — enables inactive connections that still have remaining quota; returns `{affected: [ids]}`; skips connections without remaining quota.
- Both bulk endpoints write an `audit_log` row with actor, `connection.bulk_disable`/`connection.bulk_enable` action, and the affected IDs in details (read `internal/store/audit.go` first).
- DDD-lite layering: store owns the aggregation query + zero-fill helper and bulk quota mutations; handlers stay thin (parse, validate, envelope, status). Tests written first, temp SQLite DBs, no mocks.
- Tasks: (1) store chart aggregation + zero-fill + tests; (2) handler `/api/usage/chart` param validation + tests; (3) store + handlers bulk disable/enable + tests; (4) checkpoint.
