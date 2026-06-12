# w5-c fix micro-plan — diff-gate round 2 (Fable 5, 2026-06-12)

Source: `artifacts/w5-c-observability-diff-scoped-gpt.txt` (cycle 2, REJECT).

## Finding 1 (MAJOR) — "maxJsonSize overrides treated as KB ... parse overrides as
bytes."
FALSE POSITIVE — SECOND occurrence of a finding already rebutted in
`fixes/w5-c-fix-r1.md` Finding 2 with the exact ref citation:
`requestDetailsRepo.js:27` multiplies BOTH the settings value and the env value by
1024 (`(settings.observabilityMaxJsonSize || parseInt(process.env.
OBSERVABILITY_MAX_JSON_SIZE || "5", 10)) * 1024`). KB-in, bytes-out IS the ref
contract. NO CHANGE.

## Finding 2 (MAJOR) — "_preview uses the first 200 BYTES of marshaled JSON, not the
first 200 CHARACTERS; can split UTF-8."
REAL — ref `requestDetailsRepo.js:66` `str.substring(0, 200)` counts characters
(UTF-16 code units); the Go byte-slice can cut a multibyte rune. FIX in
`internal/usage/observability.go` TruncateField: take the first 200 RUNES
(`[]rune(string(str))[:min(200, runeCount)]` or equivalent rune-safe truncation).
Test FIRST: `TestTruncateFieldUTF8Preview` — value whose JSON encoding places a
multibyte rune (e.g. "é"/"日") straddling byte 200 → preview is valid UTF-8
(`utf8.ValidString`) and has ≤200 runes; run failing → fix.

## Ownership
`internal/usage/observability.go`(+test) ONLY.

## Binary acceptance
- `go build ./... && go vet ./... && go test ./...` green; `go test -race ./internal/usage/` green.
- TestTruncateFieldUTF8Preview passes; existing TestTruncateField/TestTruncateFieldShortOversize stay green.
