# Planning Index — Stage 12B-19

Plan-tier artifacts per phase. Each dir: `brief.md`, `architect-plan.md` (links canonical spec in `docs/phases/`), `orchestration.jsonl` (task decomposition), `risk-register.md`, `rollback.md`, `verification-gate.md`.

## Execution order

1. [phase-12b-ddd-architecture-refactor](./phase-12b-ddd-architecture-refactor/) — whole-project DDD/layering refactor (blocking; zero behavior change)
2. [phase-13-auth-core-infrastructure](./phase-13-auth-core-infrastructure/) — dashboard auth: setup, sessions, CSRF, user CRUD
3. [phase-14-providers-testing](./phase-14-providers-testing/) — provider expansion, proxy pools, connection testing
4. [phase-15-tunnels-network](./phase-15-tunnels-network/) — cloudflared + tailscale tunnels
5. [phase-16-chat-console](./phase-16-chat-console/) — chat console streaming, attachments, server console
6. [phase-17-usage-analytics](./phase-17-usage-analytics/) — usage time-series, cost attribution, bulk disable
7. [phase-18-bifrost-features](./phase-18-bifrost-features/) — governance 18A→18B→18C→18D (teams, virtual keys, routing rules, limits, guardrails, alerts, templates, backup)
8. [phase-19-advanced-features](./phase-19-advanced-features/) — semantic cache, self-update, WebSocket chat, MITM CA (MITM last)

Phases 20-21 (UI integration with Lovable-generated dashboard) are user-driven; no Plan dir yet.

## Gates (every phase)

`go test ./... -count=1 && go vet ./... && go build ./cmd/g0router`, plus per-phase `go test -race` and coverage ≥95.0%. Commit format `phase-N/task-M: <description>`, direct push to `main` — no PRs.

## Deferred (out of scope)

WebRTC transport, 33-locale i18n, adaptive routing beyond `auto` classifier, OpenTelemetry export.
