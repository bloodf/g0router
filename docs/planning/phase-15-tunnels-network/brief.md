# Brief

**Problem:** g0router needs Cloudflare/Tailscale tunnel management with safe CLI integration, tunnel health checks, and outbound proxy connectivity testing — all without introducing supply-chain or privilege-escalation risk.

**Success criteria:**
- `cloudflared` is downloaded only with a pinned version + SHA-256 verified before exec; mismatch refuses and runs nothing.
- Tunnel + proxy CRUD/health/test endpoints follow the `{data,error}` snake_case envelope with auth + audit on mutations.
- Tunnel config persists encrypted; API responses never expose tokens.
- Background health loops (60s tunnel, 5min proxy) start from server startup and stop cleanly on context cancel.

**Non-goals:**
- No Tailscale download/install (drives a preinstalled binary only; 409 + instructions if absent).
- No UI work (verified via Go tests + curl).
- No shell interpolation or user input in `exec.Command` args.

**Constraints:** DDD-lite layering (`api/handlers` → `internal/tunnel` → `internal/store`); absolute exec paths + fixed arg slices; tunnel name validation `[a-z0-9-]{1,63}`; coverage ≥95.0%; direct push to main.

**Verification:** `go test ./... -count=1 && go vet ./... && go test -race ./... && go build ./cmd/g0router` all green with coverage ≥95.0% — non-skippable.

**QA criteria:**
```yaml
qa_skip: null
scenarios:
  - id: 1
    description: Cloudflare tunnel lifecycle — POST creates+runs a verified cloudflared process, GET reports active+URL, DELETE stops it.
    method: runtime-required
    evidence: curl against running binary; tunnel status transitions inactive→starting→active→inactive.
  - id: 2
    description: Tailscale absent from PATH returns 409 with install instructions in error field.
    method: api
    evidence: curl POST /api/tunnels/tailscale on host without tailscale; response 409 + {data:null,error:...}.
  - id: 3
    description: Proxy-test returns {ok,latency_ms,error} for reachable and unreachable proxies, never 500.
    method: api
    evidence: curl POST /api/settings/proxy-test with good + bad urls.
manual_smoke: Start binary, create a Cloudflare tunnel, confirm /healthz reachable via tunnel URL, delete tunnel, confirm process killed.
```

**Linked artifacts:** architect-plan: ./architect-plan.md; orchestration: ./orchestration.jsonl
