# g0router

Single-binary Go LLM gateway with 20+ providers, 100% OpenAI-compatible API, embedded React dashboard, OAuth flows, and 9router-style management features.

---

## What This Is

g0router is a local/self-hosted AI gateway that exposes a drop-in OpenAI-compatible API (`/v1/*`) and routes requests across multiple upstream providers. It combines:

- **BiFrost-style Go architecture** for the OpenAI-compatible layer: explicit provider interface, converter-based request/response normalization, streaming SSE standardization, model catalog with pricing, and virtual-key governance.
- **9router-style features** for the management layer: RTK token compression, Caveman mode, 3-tier fallback, combos, multi-account per provider, OAuth auto-refresh, quota tracking, request logging, cloud sync, provider nodes, proxy pools, MCP gateway, and translator debug UI.

The result is a single Go binary with an embedded React dashboard that can replace direct provider calls for CLI tools (Claude Code, Codex, Cursor, etc.) and production applications.

---

## Core Value

1. **Drop-in OpenAI replacement** — change `base_url`, keep everything else.
2. **Multi-provider resilience** — automatic fallback across providers and accounts.
3. **Cost optimization** — RTK compression, quota tracking, weighted routing.
4. **Single-binary simplicity** — Go + embedded UI, runs anywhere.

---

## Key Decisions

- Go backend with fasthttp; no Node runtime required.
- SQLite WAL persistence with additive-only migrations.
- Embedded React 19 + Tailwind 4 + shadcn/ui dashboard.
- BiFrost provider interface + converter pattern for the OpenAI API surface.
- 9router feature parity for management APIs and dashboard pages.
- Full Playwright E2E coverage via a 1:1 mocked API layer.

---

## Validated Capabilities

_Note: previous milestones (phase-12b through phase-19) delivered an earlier iteration of the dashboard and provider system. This milestone is a clean-slate pivot to the 9router+BiFrost architecture documented in `docs/superpowers/specs/2026-06-08-9router-bifrost-port-design.md`._

---

## Current Milestone: v2.0 9router + BiFrost Clean Slate Port

**Goal:** Rebuild g0router from a clean slate as a Go 1:1 implementation of 9router's feature set, using BiFrost's proven Go patterns for the OpenAI-compatible API layer.

**Target features:**
- 100% OpenAI-compatible `/v1/*` API (chat, completions, embeddings, images, audio, responses, files, batch, models).
- Management `/api/*` for providers, connections, keys, virtual keys, models, aliases, combos, routing rules, usage, logs, proxy pools, nodes, MCP, sync, translator debug.
- 20+ provider implementations using BiFrost's provider interface + converter pattern.
- Model catalog with pricing, cross-provider resolution, and custom overrides.
- Virtual-key governance with weighted routing and automatic fallback chains.
- Embedded dashboard ported from 9router WebUI, adapted to g0router branding and existing Vite/React stack.
- Full Playwright E2E via mocked API layer.

---

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition:**
1. Requirements invalidated? → Move to Out of Scope with reason.
2. Requirements validated? → Move to Validated with phase reference.
3. New requirements emerged? → Add to Active.
4. Decisions to log? → Add to Key Decisions.
5. "What This Is" still accurate? → Update if drifted.

**After each milestone:**
1. Full review of all sections.
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state.

---

*Last updated: 2026-06-08*
