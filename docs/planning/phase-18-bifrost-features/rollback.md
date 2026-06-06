# Rollback

- All `feature_flags` seeded `enabled=0`; governance/guardrails/pii features default-off → revert is safe (no behavior change for existing traffic until explicitly toggled).
- All new tables (`teams`, `virtual_keys`, `routing_rules`, `model_limits`, `prompt_templates`, `mcp_tool_groups`, `alert_channels`, `feature_flags`) are additive — no column drops, no destructive migration; existing `/v1/*` and `/api/*` paths unaffected.
- `request_log.virtual_key_id` added via ensureColumn (additive, nullable).
- Per sub-stage: revert the sub-stage's commits; lower sub-stages stay green independently (18A→18D dependency chain, no back-references).
- No data migration to undo; restore endpoint is opt-in and validates schema version before applying.
