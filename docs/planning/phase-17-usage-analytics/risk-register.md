# Risk Register — Phase 17

- **Slow queries on large `request_log`.** A full-scan GROUP BY over a large table can be slow. Mitigation: single GROUP BY per request, filter by `created_at` range first; verify an index on `created_at` exists (add if missing).
- **Timezone bucketing bugs.** `strftime` operates on the stored `created_at` representation; bucket labels can drift if storage is not UTC. Mitigation: assert tests against known UTC-seeded rows; keep bucket math in SQL, zero-fill in Go using the same boundaries.
- **Bulk-disable disabling active models.** A wrong threshold comparison could disable connections that still have quota. Mitigation: strict at/below-threshold predicate, test the boundary (just-above stays enabled), return + audit affected ids for traceability.
