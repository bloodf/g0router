---
phase: 02
slug: schemas-catalog
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-09
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (built-in) |
| **Config file** | none |
| **Quick run command** | `go test ./internal/schemas/... ./internal/catalog/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/schemas/... ./internal/catalog/...`
- **After every plan wave:** Run `go test ./...`
- **Before `$gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | OPENAI-01 | — | N/A | unit | `go test ./internal/schemas/...` | ✅ W0 | ⬜ pending |
| 02-01-02 | 01 | 1 | OPENAI-03 | — | N/A | unit | `go test ./internal/schemas/...` | ✅ W0 | ⬜ pending |
| 02-01-03 | 01 | 1 | OPENAI-04 | — | N/A | unit | `go test ./internal/schemas/...` | ✅ W0 | ⬜ pending |
| 02-02-01 | 02 | 1 | OPENAI-07 | — | N/A | unit | `go test ./internal/schemas/...` | ✅ W0 | ⬜ pending |
| 02-02-02 | 02 | 1 | OPENAI-08 | — | N/A | unit | `go test ./internal/schemas/...` | ✅ W0 | ⬜ pending |
| 02-02-03 | 02 | 1 | OPENAI-11 | — | N/A | unit | `go test ./internal/schemas/...` | ✅ W0 | ⬜ pending |
| 02-03-01 | 03 | 1 | PROV-01 | — | N/A | unit | `go test ./internal/schemas/...` | ✅ W0 | ⬜ pending |
| 02-03-02 | 03 | 1 | GOV-01 | — | N/A | unit | `go test ./internal/schemas/...` | ✅ W0 | ⬜ pending |
| 02-04-01 | 04 | 2 | CATALOG-01 | — | Seed data loads offline | unit | `go test ./internal/catalog/...` | ✅ W0 | ⬜ pending |
| 02-04-02 | 04 | 2 | CATALOG-02 | — | Sync fails gracefully | unit | `go test ./internal/catalog/...` | ✅ W0 | ⬜ pending |
| 02-05-01 | 05 | 3 | CATALOG-03 | — | Lookup fallback chain correct | unit | `go test ./internal/catalog/...` | ❌ W0 | ⬜ pending |
| 02-05-02 | 05 | 3 | CATALOG-04 | — | Cross-provider resolution correct | unit | `go test ./internal/catalog/...` | ❌ W0 | ⬜ pending |
| 02-05-03 | 05 | 3 | CATALOG-05 | — | Allowlist logic correct | unit | `go test ./internal/catalog/...` | ❌ W0 | ⬜ pending |
| 02-06-01 | 06 | 3 | CATALOG-06 | — | Tiered pricing correct | unit | `go test ./internal/catalog/...` | ❌ W0 | ⬜ pending |
| 02-06-02 | 06 | 3 | CATALOG-08 | — | Capabilities exposed | unit | `go test ./internal/catalog/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `internal/schemas/schemas_test.go` — placeholder exists
- [x] `internal/catalog/catalog_test.go` — placeholder exists
- [ ] `internal/catalog/lookup_test.go` — stub for CATALOG-03..05 (Plan 05)
- [ ] `internal/catalog/pricing_test.go` — stub for CATALOG-06..08 (Plan 06)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Background sync updates pricing | CATALOG-02 | Requires external HTTP call | Verify sync goroutine starts and logs success/failure |

---

## Validation Sign-Off

- [ ] All tasks have automated verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
