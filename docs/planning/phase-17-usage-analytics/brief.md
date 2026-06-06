# Brief

**Problem:** The dashboard buckets usage time-series client-side over the loaded (paginated) log window, silently truncating data. We need backend aggregation plus bulk connection quota actions.

**Success criteria:**
- `GET /api/usage/chart?period&granularity` returns zero-filled, index-aligned `{buckets, requests, tokens_input, tokens_output, costs}` from `request_log`.
- `POST /api/connections/bulk-disable` disables connections at/below a remaining-quota threshold and returns `{affected: [ids]}` with an audit row.
- `POST /api/connections/bulk-enable` enables inactive connections that still have remaining quota and returns `{affected: [ids]}` with an audit row.
- Per-phase gate green (go test, vet, race, build) with coverage ≥ 95.0%.

**Non-goals:**
- No UI work (Lovable owns chart/countdown/sort/filter/pagination in phases 20-21).
- No new aggregation backend (SQLite/`request_log` only; no Postgres `date_trunc`).
- No changes to existing `/api/quota` shape.

**Constraints:**
- Data source is the `request_log` table (NOT `usage_logs`, which does not exist).
- SQLite only — use `strftime` for bucketing; never `date_trunc`. snake_case fields, `{data, error}` envelope, audit on mutating endpoints.

**Verification:** Seeded `request_log` rows produce correct per-bucket sums with gaps zero-filled; bulk endpoints touch only threshold-eligible connections and write audit rows; invalid period/granularity → 400.

**QA criteria:**
```yaml
qa_skip: null
scenarios:
  - id: usage-chart-shape
    method: api
    description: GET /api/usage/chart?period=7d&granularity=day returns zero-filled, index-aligned buckets/requests/tokens_input/tokens_output/costs in the {data,error} envelope; invalid period/granularity returns 400.
  - id: bulk-disable
    method: api
    description: POST /api/connections/bulk-disable {threshold_percent} disables only at/below-threshold connections, returns {affected:[ids]}, and writes an audit_log row.
manual_smoke: curl /api/usage/chart and /api/connections/bulk-disable against a seeded DB; confirm envelope, zero-fill, and audit row.
```

**Linked artifacts:**
- architect-plan: ./architect-plan.md
- orchestration: ./orchestration.jsonl
