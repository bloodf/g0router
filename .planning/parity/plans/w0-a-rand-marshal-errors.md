# w0-a — crypto rand + marshal/write error handling (rev 2)

Rows: AUD-001, AUD-002, AUD-003, AUD-007, AUD-008, AUD-009, AUD-010, AUD-011, AUD-012, AUD-045. Behavior contract per row: the binary acceptance column of PARITY.md §2 Bundle A (approved at Stage 0 checkpoint) — where the audit row offered options, Bundle A already selected one; this plan implements that selection.
Worker: M3. Reviewer: Kimi diff gate.

## File ownership (time-boxed exclusivity)
Ownership is exclusive ONLY during this plan's execution window. Wave 0 sequencing (WAVE-MAP.md): w0-a runs first and alone among Go-side plans; w0-b/w0-c start after w0-a merges; w0-e starts after w0-a and w0-c merge. No two in-flight plans ever own the same file.
- `internal/store/store.go`, `internal/store/connections.go`, `internal/store/providers.go`, `internal/store/users.go`, `internal/store/store_test.go`
- `internal/auth/session.go`, `internal/auth/oauth.go` (ONLY `randomURLSafe` + its callers), `internal/auth/auth_test.go`
- `internal/api/chat.go`, `internal/api/embeddings.go`, `internal/api/models.go`, `internal/api/errors.go`, `internal/api/chat_test.go`, `internal/api/embeddings_test.go`, `internal/api/models_test.go`, `internal/api/errors_test.go` (create the `_test.go` files that don't exist; touch no other API tests)
- `internal/providers/openai/chat.go`, `internal/providers/anthropic/chat.go`, `internal/providers/gemini/chat.go` + their existing `_test.go` files

## Error-injection seams (only where needed)
- `internal/store`, `internal/auth`: `var randRead = rand.Read` (AUD-001/002/003).
- `internal/api`: `var jsonMarshal = json.Marshal` (AUD-007/009/010/011/012).
- `internal/providers/*`: NO seam. AUD-045 tests feed a malformed SSE JSON line through the real decode path.
Tests swap a seam with a failing func and restore via `t.Cleanup`. No mock libraries, no new interfaces.

## Tasks (TDD order — each test must fail before its fix)

1. **AUD-001** `internal/store`: test `TestNewIDRandFailure` — failing `randRead` → `newID` returns `("", err)`. Change `newID() string` → `newID() (string, error)`. Exactly three call sites (verified by grep): `connections.go:34`, `providers.go:24`, `users.go:23` — each propagates wrapped `fmt.Errorf("generate id: %w", err)`.
2. **AUD-002** `internal/auth`: test `TestNewTokenRandFailure` — `newToken() (string, error)` (`session.go:103`); `CreateSession` propagates.
3. **AUD-003** `internal/auth`: test `TestRandomURLSafeFailure` — `randomURLSafe(n int) (string, error)` (`oauth.go:164`); OAuth start handler returns 500 on error.
4. **AUD-009/010/011/012** `internal/api`: table-driven test per handler — failing `jsonMarshal` → handler writes 500 (`chat.go:72`, `embeddings.go:50`, `models.go:44`); `writeError` (`errors.go:20`) falls back to `ctx.SetStatusCode(500)` + `ctx.SetBodyString("internal error")`.
5. **AUD-007/008** `internal/api/chat.go:53-58` SSE path: test with a writer whose `Write` returns an error — assert the stream loop stops consuming chunks. Fix: check `jsonMarshal` error → abort stream; check write/flush error → return.
6. **AUD-045** each provider `chat.go`: test feeding one malformed SSE JSON line — assert the stream returns an error (Bundle A acceptance: "provider stream aborts on JSON unmarshal error"). Fix: replace `continue` with `return fmt.Errorf("decode stream chunk: %w", err)`.

## Acceptance (binary)
- Tests named above exist; `go test ./...` green; `go vet ./...` clean.
- `grep -rn "rand.Read" internal/store internal/auth | grep -v "_test.go\|randRead"` → empty.
- `grep -n "json.Marshal" internal/api/*.go | grep -v "_test.go\|jsonMarshal"` → empty.
- AUD-045 per-provider malformed-line tests pass (these fail against the old `continue` behavior, making the check binary).
- No exported API changes other than error returns added to the functions named above.

## Out of scope
CORS/client_id/DDL (w0-b). Converter mappings (w0-c, w0-e). Scanner `Err()` propagation AUD-046 and hook errors AUD-047 (w0-e). Any file not listed.
