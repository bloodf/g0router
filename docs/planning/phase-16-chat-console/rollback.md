# Rollback

- `chat_sessions` migration is additive; leaving the table in place is harmless. To revert, `git revert` the phase commits.
- New endpoints and `internal/console/` are isolated; removing the route registrations and the console package disables them without affecting `/v1/*` or existing `/api/*`.
- Console slog tee is constructor-injected; reverting the startup wiring restores the original logger with no global-state cleanup needed.
- No data migration is destructive; existing tables and behavior are untouched.
