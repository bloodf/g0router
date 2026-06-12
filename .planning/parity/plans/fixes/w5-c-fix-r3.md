# w5-c fix micro-plan — close-by-decision fixes (Fable 5, 2026-06-12)

Source: `artifacts/w5-c-observability-diff-scoped-gpt.txt` (cycle 3). Applied as the
REAL→fixed half of the close-by-decision disposition (the gate record's other two
findings are rebutted there).

## Fix 1 (cycle-3 Finding 2, REAL) — Go `json.Marshal` HTML-escapes `<`,`>`,`&`;
`JSON.stringify` does not, so `_originalSize`/`_preview` and the threshold decision
diverge for payloads containing those characters.
FIX in `internal/usage/observability.go` TruncateField (and any other place the
writer serializes blob fields for size/preview — check detailwriter.go's data
marshaling too): use `json.Encoder` with `SetEscapeHTML(false)` (trim the trailing
newline the encoder appends). Test FIRST: `TestTruncateFieldNoHTMLEscape` — value
containing `<b>&` → marshaled size equals the JS length (no `<` expansion) and
preview contains literal `<b>&`; run failing → fix.

## Fix 2 (cycle-3 Finding 3, REAL) — zero-result `QueryRequestDetails` returns a nil
slice which a JSON encoder renders as `null`; the ref returns `[]`
(`requestDetailsRepo.js:169` maps over an empty rows array).
FIX in `internal/store/requestdetails.go`: initialize `rows :=
make([]json.RawMessage, 0)` (never nil). Test FIRST: extend
`TestRequestDetailsQuery` with an empty-result filter case asserting non-nil
zero-length slice; run failing → fix.

## Ownership
`internal/usage/observability.go`(+test), `internal/usage/detailwriter.go` (only if
its data marshaling also HTML-escapes — same one-line change),
`internal/store/requestdetails.go`(+test).

## Binary acceptance
- `go build ./... && go vet ./...` green; `go test ./internal/usage/... ./internal/store/...` green excluding the two known in-progress w5-d chart tests (`TestChartToday`, `TestChart24hClamp` — another job's TDD failing-first tests, NOT this plan's concern); `go test -race ./internal/usage/ ./internal/store/` same exclusion.
- TestTruncateFieldNoHTMLEscape passes; empty-result query case passes.
