# GSD State

## Current Position

**Milestone:** v2.0 9router + BiFrost Clean Slate Port
**Phase:** 5 (Anthropic + Gemini Providers) — COMPLETE
**Plan:** `.planning/phases/05-anthropic-gemini-providers/PLAN.md`
**Status:** Ready for next phase

**Last activity:** 2026-06-09 — Phase 5 Anthropic + Gemini Providers complete. Both converter-based providers implement chat (non-streaming + streaming SSE), Gemini adds embeddings, with full error converters and not-implemented stubs. Router updated with prefix-based model resolution. All gates pass.

---

## Phase 1 — Scaffolding — Deliverables

| Commit | Subject |
|---|---|
| `6338148` | phase-01/task-1: remove obsolete api/, internal/, and root e2e tests |
| `63124ba` | phase-01/task-2: scaffold internal/ package layout with placeholder tests |
| `c900b55` | phase-01/task-3: rewrite cmd/g0router/main.go as minimal fasthttp skeleton |
| `e36a19c` | phase-01/task-4: go mod tidy |
| `79db515` | phase-01/task-1: scaffold minimal UI placeholder (main.tsx, App.tsx, index.css) |

---

## Phase 2 — Schemas + Catalog — Deliverables

| Commit | Subject |
|---|---|
| `cde0e77` | phase-02/task-1: core OpenAI-compatible schema types (chat, completions, embeddings) + round-trip tests |
| `f7cc17f` | phase-02/task-2: extended schema types (images, audio, files, batch, responses, errors) + round-trip tests |
| `d1f2cee` | phase-02/task-3: provider interface, governance, catalog, and MCP stub types + compile check |

---

## Phase 3 — OpenAI Provider — Deliverables

| Commit | Subject |
|---|---|
| `ee8c48a` | phase-03/task-1: OpenAI provider (chat, embeddings, models, streaming) + utils + tests |

---

## Phase 4 — OpenAI API Handlers — Deliverables

| Commit | Subject |
|---|---|
| (to be added on commit) | phase-04/task-1: fasthttp server with CORS, request ID, chat/embeddings/models handlers |

---

## Phase 5 — Anthropic + Gemini Providers — Deliverables

| Commit | Subject |
|---|---|
| (to be added on commit) | phase-05/task-1: Anthropic provider with chat converter, streaming, error converter + tests |
| (to be added on commit) | phase-05/task-2: Gemini provider with chat/embedding converters, streaming, error converter + tests |
| (to be added on commit) | phase-05/task-3: update router with prefix-based Anthropic/Gemini resolution + env key helpers |

---

## Accumulated Context

- Clean-slate pivot from previous g0router architecture. Phase 1 wipes the v1 code (`api/`, `internal/`, `ui/src/`) and the v1-era root `e2e_*.go` files.
- Design spec approved: `docs/superpowers/specs/2026-06-08-9router-bifrost-port-design.md`
- BiFrost patterns adopted for OpenAI-compatible layer.
- 9router features targeted for management layer and dashboard.
- 19 phases grouped into 6 execution waves.
- Phase 1 leaves `go.mod` with a single direct dep (`github.com/valyala/fasthttp v1.71.0`).
- Phase 2 fills `internal/schemas/` with all shared wire-format types.
- Phase 3 implements the reference OpenAI provider in `internal/providers/openai/` with fasthttp, SSE streaming, and shared utilities.
- Phase 4 exposes `/v1/chat/completions`, `/v1/embeddings`, and `/v1/models` via fasthttp handlers.
- Phase 5 adds converter-based Anthropic and Gemini providers with format translation.

---

## Active Blockers

_None._

---

## Next Step

Continue **Wave 2: Core Providers + Admin** with **Phase 6: Management API Foundation**.

Build the admin API foundation: auth, settings, providers, and connections.

### What "next phase" should keep in mind
- `go test ./...` and `go vet ./...` must pass green at every commit.
- Commit format: `phase-06/task-N: <description>`.
