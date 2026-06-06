# Risk Register

- **Budget race conditions** — concurrent `/v1/*` requests double-spend before `budget_used_usd` is written; under-counts spend and lets keys exceed cap. Mitigate: atomic check-and-accumulate (single SQL UPDATE / serialized per-key path); test concurrent accrual.
- **Routing-rule cache staleness** — TTL cache serves a deleted/edited rule after write; requests route to wrong target. Mitigate: explicit invalidation on every rule write (mirror alias cache); test cache-invalidates-on-write.
- **Flag-gating bypass** — guardrails/pii_redaction logic runs (or is skipped) inconsistent with flag state, leaking unredacted PII. Mitigate: single flag check at dispatch boundary, pass-through when off; test both states on streaming + non-streaming.
- **Backup leaking secrets** — export accidentally serializes OAuth tokens / proxy passwords / alert configs. Mitigate: null-placeholder + `redacted_fields` manifest; assert-scan test rejects any token/password value.
