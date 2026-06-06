# Architect Plan — Phase 18: Bifrost Features (Governance)

Canonical spec: [`docs/phases/phase-18-bifrost-features.md`](../../phases/phase-18-bifrost-features.md)

## Summary

- **18A — Virtual Keys, Teams, Governance Middleware:** `teams` + `virtual_keys` tables (additive). Virtual keys carry `gvk-` prefix and SHA-256 `key_hash` (same scheme as `api_keys`); raw key returned once on create, list shows prefix only.
- **18A budgets:** `budget_used_usd` accumulates post-request on key AND team from computed cost; lazy budget reset zeros used + rolls period forward when `now > budget_reset_at`. Hierarchical enforcement — key limit AND team limit must both pass; checked in `/v1/*` auth middleware BEFORE regular API keys. Reject `429` (limits) / `403` (budget exhausted / inactive). New `virtual_key_id` request_log column via ensureColumn.
- **18A domain:** `internal/governance/` owns virtual-key/budget/limit logic with its own repository interface; handlers stay thin CRUD; middleware calls into governance.
- **18B — Routing Rules, Model Limits:** `routing_rules` + `model_limits` tables. Rules evaluated in priority order BEFORE alias/combo resolution in `internal/proxy`; first match rewrites target. Rules TTL-cached using the existing alias cache pattern, invalidated on write.
- **18B model limits:** checked at dispatch — `max_tokens` cap clamps/rejects (pick one, document), RPM via in-memory limiter, key allowlist → 403.
- **18C — Guardrails, PII, Prompts, MCP Tool Groups:** guardrails config via settings keys (`guardrails_enabled`, `guardrails_blocklist_json`, `pii_redaction_enabled`, `pii_types_json`) — no table. `internal/guardrails/` pure functions: blocklist match → 400, PII redaction rewrites `email`/`phone`/`ssn`/`credit_card`/`ip_address` spans with `[REDACTED:<type>]`, in dispatch BEFORE RTK/Caveman. Flag-gated `[flag: guardrails]` `[flag: pii_redaction]`.
- **18C templates + tool groups:** `prompt_templates` table ({{var}} extraction) and `mcp_tool_groups` table (JSON tool-name array) filtering MCP tool injection when a group is selected on a key/combo.
- **18D — Alerts, Feature Flags, Backup/Restore:** `alert_channels` table stores `config_enc` (encrypted URLs/tokens, OAuth-token pattern); `internal/alerts/` dispatches webhook/discord/telegram on event types with retry, no store imports.
- **18D flags:** `feature_flags` table seeded `semantic_cache`, `guardrails`, `pii_redaction`, `websocket_chat`, `mitm_proxy` all 0; `PUT /api/feature-flags/:id` toggle-only (no POST/DELETE, no user-created flags), unknown flag → 404.
- **18D backup:** export EXCLUDES all secret material — api key hashes kept (needed), but OAuth tokens / proxy passwords / tunnel + alert configs exported ONLY as `null` placeholders plus a `redacted_fields` manifest; restore keeps existing secrets on null placeholder, validates schema version, rejects unknown shape with 400. Admin-session or bearer auth; both audited.
- **Task list:** 18A tasks 1-3 (store / domain / middleware); 18B tasks 4-5 (rules + limits store/domain/proxy); 18C tasks 6-8 (guardrails / prompts / tool groups); 18D tasks 9-11 (alerts / flags / backup-restore); checkpoint task incl. security pass.

## Security notes

Security review is **mandatory** for Phase 18 (per STAGE-13-19-PROCESS §7 — budget enforcement, backup/restore secret export). Checklist per the focused pass:
- Input validation at every new handler boundary; fail fast with `400` + specific message.
- authn/authz on every new route; backup/restore requires admin-session or bearer, both audited.
- Secrets at rest: `alert_channels.config_enc` encrypted via the OAuth-token mechanism (`internal/store/oauthsessions.go`); virtual key material SHA-256 hashed; never plaintext.
- Secrets in logs: assert backup output scan contains no token/password values.
- Budget enforcement cannot be bypassed by race or stale read (see risk register).
- Privilege requirements for backup/restore documented in `## Outcome`.
