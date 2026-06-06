# Brief

**Problem:** g0router lacks a governance layer. Operators cannot cap spend per key
or team, route requests by rule, gate features behind flags, guard prompts (blocklist
+ PII), template prompts, group MCP tools, dispatch alerts, or back up config safely.
Phase 18 delivers this across four sub-stages (18A-18D) on the DDD-lite layering.

**Success criteria:**
1. Virtual keys (`gvk-` prefix) and teams enforce hierarchical budget + RPM/TPM limits in `/v1/*` middleware; lazy budget reset rolls the period forward.
2. Routing rules + model limits apply in priority order BEFORE alias/combo resolution, TTL-cached and invalidated on write.
3. Guardrails (blocklist + PII redaction), prompt templates, MCP tool groups, alert channels, and toggle-only feature flags (seeded all 0) are CRUD-wired and flag-gated.
4. Backup excludes all secret material via `null` placeholders + a `redacted_fields` manifest; restore keeps existing secrets on null.

**Non-goals:**
1. Adaptive routing / heuristic classifier — duplicate of existing `auto` combo strategy. Deferred.
2. OTel distributed tracing — optional infra, no UI dependency. Deferred.
3. Any `ui/` work — backend-only; verified via Go tests + curl.

**Constraints:** Direct push to `main`. snake_case JSON, `{data,error}` envelope.
Per-phase gate green, coverage ≥ 95.0%. Additive migrations only. Audit every
mutating endpoint. Encrypt reversible secrets, hash key material. Security review
mandatory (§7).

**Verification:** `go test -race ./...` green, coverage ≥ 95.0%, security pass recorded, backup secret-scan asserts zero token/password values.

**QA criteria:**
```yaml
qa_skip: null
scenarios:
  - id: vk-budget-enforcement
    method: api
    desc: Virtual key with exhausted budget returns 403; team RPM exceeded returns 429 (hierarchical); usage accumulates on key + team post-request.
  - id: routing-rule-application
    method: api
    desc: Routing rule matches in priority order and rewrites target before alias/combo resolution; inactive rule skipped; cache invalidates on write.
  - id: feature-flag-gating
    method: api
    desc: guardrails/pii_redaction flags off → pass-through; on → blocklist blocks (400) and PII spans redacted; flag toggle persists, unknown flag → 404.
  - id: backup-secret-redaction
    method: api
    desc: Backup export contains no OAuth token / proxy password / alert config values (placeholders null); restore round-trip keeps existing secrets on null placeholder; bad payload → 400.
manual_smoke: Create a gvk- key + team via curl, drive a /v1/* request, confirm budget_used_usd accrues on both; run POST /api/settings/backup and grep output for secret leakage.
```

**Linked artifacts:** architect-plan: ./architect-plan.md; orchestration: ./orchestration.jsonl
