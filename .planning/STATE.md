# GSD State

## Current Position

**Milestone:** v2.0 9router + BiFrost Clean Slate Port  
**Phase:** 4 (OpenAI API Handlers) — NEXT  
**Plan:** `.planning/ROADMAP.md`  
**Status:** Ready for autonomous execution

**Last activity:** 2026-06-09 — Phase 3 OpenAI Provider complete. OpenAI provider implements chat (non-streaming + streaming SSE), embeddings, list models, and error converter via fasthttp. Shared provider utilities (ClientPool, SSEScanner, JSON helpers) created. 20+ not-implemented stubs for future phases. All gates pass.

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

## Accumulated Context

- Clean-slate pivot from previous g0router architecture. Phase 1 wipes the v1 code (`api/`, `internal/`, `ui/src/`) and the v1-era root `e2e_*.go` files.
- Design spec approved: `docs/superpowers/specs/2026-06-08-9router-bifrost-port-design.md`
- BiFrost patterns adopted for OpenAI-compatible layer.
- 9router features targeted for management layer and dashboard.
- 19 phases grouped into 6 execution waves.
- Phase 1 leaves `go.mod` with a single direct dep (`github.com/valyala/fasthttp v1.71.0`).
- Phase 2 fills `internal/schemas/` with all shared wire-format types.
- Phase 3 implements the reference OpenAI provider in `internal/providers/openai/` with fasthttp, SSE streaming, and shared utilities.

---

## Active Blockers

_None._

---

## Next Step

Continue **Wave 1: Foundation** with **Phase 4: OpenAI API Handlers**.

Expose `/v1/chat/completions`, `/v1/embeddings`, and `/v1/models` via fasthttp handlers in `internal/server/` or `internal/api/`. Wire the OpenAI provider into the handler layer.

### What "next phase" should keep in mind
- `cmd/g0router/main.go` remains minimal. Register new routes in `internal/server/`.
- Use the schema types from Phase 2 for request/response shapes.
- Use the OpenAI provider from Phase 3 for backend calls.
- `go test ./...` and `go vet ./...` must pass green at every commit.
- Commit format: `phase-04/task-N: <description>`.
