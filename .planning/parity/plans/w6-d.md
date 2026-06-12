# w6-d — i18n: locale catalog, react-i18next runtime, locale cookie + endpoint

Status: PLANNED
Depends on: w6-pre (MERGED), w6-a (MERGED)
Runs in parallel with: w6-b (disjoint files — verified below)
Serial slot: `internal/server/routes_admin.go` — w6-d is the 2nd w6 holder (after w6-pre, 1st). Append-only.

## 1. PAR rows

| Row | Scope | Parity level |
|---|---|---|
| PAR-UI-069 | 33 locales in LOCALES array (source of truth: 9router `src/i18n/config.js:1`) | FULL |
| PAR-UI-070 | i18n runtime wiring | PARTIAL after w6-d (I18nProvider created + configured — variant of DOM approach per WAVE-6-MAP §Arch decision 3; rows DO NOT flip HAVE, only PARTIAL) |
| PAR-UI-071 | Re-process on route change | PARTIAL after w6-d (router.subscribe hook in I18nProvider — variant; HAVE deferred to w6-b which mounts I18nProvider in __root.tsx) |
| PAR-UI-072 | locale cookie + `POST /api/locale` endpoint | FULL |

**NOT in scope:**
- Porting `runtime.js` DOM-scanning approach (WAVE-6-MAP.md §Architectural decisions, decision 3: "i18n = react-i18next hook-based, not 9router's runtime DOM MutationObserver (`runtime.js`)").
- Translating page content — locale JSON files ship minimal/empty; keys are added as pages land in later waves.
- Wiring `I18nProvider` into `__root.tsx` — that file is frozen (w6-a); w6-b owns the provider slot wiring.
- Locale auto-detection from `Accept-Language` (not in the PAR matrix).

## 2. Precondition checks (run before writing any code)

1. **Read the ref locale list FIRST:** `/home/cortexos/Developer/github.com/bloodf/_refs/9router/src/i18n/config.js` — extract the LOCALES array. Count MUST be exactly 33. If the count differs, STOP and report; do not invent or trim locales.
2. Confirm w6-a is merged: `ui/src/lib/apiFetch` and the stores exist on `main`.
3. Confirm `internal/server/routes_admin.go` has w6-pre's routes and no uncommitted edits (`git status` clean for that file) — serial-slot handoff is clean.
4. Read 3 existing admin handlers (e.g., w6-pre's additions in `internal/admin/`) to match handler + envelope patterns before writing `locale.go`.
5. Confirm `react-i18next` / `i18next` ARE already in `ui/package.json` (PAR-UI-069 matrix note: "g0router `package.json` has `i18next` + `react-i18next`"). No new dependencies needed. If somehow absent, STOP and escalate.
6. Verify w6-b file claims (its plan's ownership list) do not intersect the list in §3.

## 3. Exact file ownership

UI — new directory `ui/src/i18n/` (no other task touches it):
- `ui/src/i18n/index.ts` — i18next + react-i18next init (resources, fallback `en`, no suspense), exports `i18n` instance.
- `ui/src/i18n/locales.ts` — `LOCALES` metadata array: `{ code, name, flag }` ×33, mirrored from ref `config.js`.
- `ui/src/i18n/locales/*.json` — ×33 minimal translation files (one per code; `{}` or shared seed keys only).
- `ui/src/providers/i18n.tsx` — `I18nProvider`: mounts react-i18next, reads `locale` cookie for initial language, subscribes to TanStack Router route changes (PAR-UI-071 variant), exposes `setLocale` that calls `POST /api/locale` via `apiFetch` then `i18n.changeLanguage`.

Go — new files plus one append-only slot:
- `internal/admin/locale.go` — handler + locale validation list (33 codes, same source).
- `internal/admin/locale_test.go` — tests first (TDD).
- `internal/server/routes_admin.go` — APPEND ONLY: register `POST /api/locale`. PAR-UI-072 only requires POST; no GET endpoint. No reordering, no edits to existing lines.

No new npm dependencies (packages are already installed per P2.5).

**Ownership boundary with w6-b (parallel plan):**
w6-d EXCLUSIVELY owns `ui/src/providers/i18n.tsx` (and its test file). w6-b DOES NOT touch
`i18n.tsx` — its provider changes are limited to `header.tsx` and `__root.tsx` slot wiring
(documented in w6-b plan §3, WAVE-6-MAP §Ownership tracks). These paths are disjoint. Verify
with `grep -r "i18n" .planning/parity/plans/w6-b.md` — w6-b must not list `i18n.tsx`.

**FORBIDDEN:**
- `ui/src/i18n/runtime.js` — must not exist (DOM approach not ported).
- `ui/src/routes/__root.tsx` — frozen; w6-b wires the I18nProvider slot.
- Any other w6-a frozen file (`__root` shell, theming, stores, lib).
- Any file outside this plan's ownership that w6-b might own.

## 4. TDD tasks (in order)

**T1 — Go endpoint (test first):**
1. Write `internal/admin/locale_test.go`:
   - `TestPostLocaleSetsCookie` — `POST /api/locale` body `{"locale":"pt-BR"}` → 200, envelope `{data:{locale:"pt-BR"}, error:null}`, `Set-Cookie: locale=pt-BR; Path=/; SameSite=Lax` (NOT HttpOnly — locale is a non-sensitive pref; JS must read it to hydrate the initial language on page load).
   - `TestPostLocaleRejectsUnknown` — body `{"locale":"xx-XX"}` → 400, `data:null`, error message names the invalid locale. Also: empty body / malformed JSON → 400.
2. Run `go test ./internal/admin/` — confirm failures are compile/404-shaped (right reason).
3. Implement `internal/admin/locale.go` minimally: validate against the 33-code set, set cookie, return envelope. No auth middleware — `POST /api/locale` is a UI preference endpoint. Ref evidence: `src/shared/components/LanguageSwitcher.js:96` (9router frozen ref @ 827e5c3): `await fetch("/api/locale", { method: "POST", ... body: JSON.stringify({ locale: nextLocale }) })` — no Authorization header, no session header.
4. Append route(s) in `routes_admin.go`; tests green; `go vet ./...` green.

**T2 — UI locale catalog (test first):**
1. Write `ui/src/i18n/locales.test.ts`: LOCALES has exactly 33 entries; codes are unique; every entry has non-empty `code`, `name`, `flag`; every code has a matching `locales/<code>.json`.
2. See it fail; create `locales.ts` (transcribed from ref) + 33 JSON files; green.

**T3 — i18n init + provider (test first):**

Context API shape (export from `providers/i18n.tsx`):
```ts
interface I18nContextValue {
  currentLocale: string;
  locales: Locale[];
  setLocale: (code: string) => Promise<void>;
}
const I18nContext = React.createContext<I18nContextValue>(...)
export const useI18n = () => React.useContext(I18nContext)
export function I18nProvider({ children }: { children: React.ReactNode })
```
`setLocale` calls `apiFetch('POST /api/locale', {locale:code})` then `i18n.changeLanguage(code)`.
Route-change subscription: TanStack Router `@tanstack/react-router@1.168.25` exposes
`router.subscribe('onResolved', callback)` — use this inside `useEffect` in I18nProvider to
re-apply the current language on navigation (PAR-UI-071 variant; no DOM re-scan).
Initial locale: read `document.cookie` for `locale=` segment on mount; fall back to `navigator.language.slice(0,2)` then `'en'`.

1. Write `ui/src/providers/i18n.test.tsx`: `useI18n()` renders children and exposes context; initial language from a seeded `document.cookie`; `setLocale('de')` calls `apiFetch` (via fake fetch) and updates `i18n.language`; subscribes to router route change (mock `router.subscribe`).
2. See failures; implement `i18n/index.ts` then `providers/i18n.tsx` minimally; green.

## 5. Binary acceptance criteria

- [ ] `LOCALES` in `ui/src/i18n/locales.ts` has exactly 33 entries, codes identical to ref `config.js` (PAR-UI-069).
- [ ] 33 files exist under `ui/src/i18n/locales/`, one per code, all valid JSON.
- [ ] `grep -r "MutationObserver" ui/src/i18n ui/src/providers` returns nothing; `ui/src/i18n/runtime.js` does not exist (PAR-UI-070 variant).
- [ ] `I18nProvider` subscribes to TanStack Router route changes; no DOM re-scan (PAR-UI-071 variant).
- [ ] `POST /api/locale` with a valid code → 200, snake_case `{data,error}` envelope, cookie `locale` set with `Path=/; SameSite=Lax` (NOT HttpOnly — JS-readable locale pref) (PAR-UI-072).
- [ ] `POST /api/locale` with an unknown code → 400 with error envelope.
- [ ] Both Go tests exist and pass: `TestPostLocaleSetsCookie`, `TestPostLocaleRejectsUnknown`; `go test ./... && go vet ./...` green. (PAR-UI-072 only requires POST; no GET test.)
- [ ] UI tests in §4 pass; `__root.tsx` byte-identical to merged w6-a (`git diff main -- ui/src/routes/__root.tsx` empty).
- [ ] `routes_admin.go` diff is append-only (no removed/modified existing lines).

## 6. Out of scope

- Full 9router runtime parity: PAR-UI-070/071 are PARTIAL by decision — the DOM MutationObserver / text-node re-scan approach is explicitly not ported. Do not "improve" parity by adding it.
- Translation content beyond minimal seed files.
- Provider mounting in the root shell (w6-b), language-switcher UI (lands with the settings page wave), server-side locale negotiation.

## 7. Diff-gate scope

Allowed paths in the closing diff — anything outside this list fails the gate:

```
ui/src/i18n/**                    (new directory + files)
ui/src/providers/i18n.tsx         (new)
ui/src/providers/i18n.test.tsx    (new)
internal/admin/locale.go          (new)
internal/admin/locale_test.go     (new)
internal/server/routes_admin.go   (append-only hunk)
docs/WORKFLOW.md                  (merge entry on close)
.planning/parity/matrix/9router-ui.md  (PAR-UI-069/070/071/072 row flips on close)
```

Gate checks: no edits to frozen w6-a files; no `runtime.js`; `routes_admin.go` hunk is additive only; no files intersecting w6-b's claim list; `ui/package.json` unchanged (no new deps); PAR-UI-070/071 flipped to PARTIAL (HAVE deferred to w6-b mount).

## Plan gate disposition (closed by decision after 3 cycles — 2026-06-12)

**Cycle 1 REJECT** — REAL: GET test conditional contradicting §5 (removed; only POST required by PAR-UI-072); P2.5 wrong direction on react-i18next (fixed; already installed); PAR-UI-070/071 claimed as HAVE but I18nProvider not mounted (fixed: rows flip PARTIAL only; HAVE deferred to w6-b mount); WAVE-6-MAP cited without line reference (fixed: §Architectural decisions decision 3 quoted); no-auth claim unsubstantiated (partially fixed).

**Cycle 2 REJECT** — REAL: HttpOnly cookie unreadable by JS (fixed: cookie is NOT HttpOnly — locale is non-sensitive pref; JS must read it); LanguageSwitcher.js no auth-header claim unsupported (fixed: `LanguageSwitcher.js:96` cites `fetch("/api/locale", ...)` with no Authorization); setLocale API shape ambiguous (fixed: explicit I18nContextValue interface added); TanStack Router route-change API unspecified (fixed: `router.subscribe('onResolved', ...)` cited with package version 1.168.25).

**Cycle 3 REJECT** — REAL: provider ownership ambiguity with w6-b (fixed: explicit ownership boundary section; w6-b does NOT touch i18n.tsx). FALSE: BLOCKER 1 (I18nContext "invented" without PAR row) — rows PAR-UI-070/071 flip to PARTIAL (not HAVE); react-i18next hook pattern is the standard approach for this library, not an invention; the variant is explicitly accepted in WAVE-6-MAP §Architectural decisions decision 3 ("i18n = react-i18next hook-based"). FALSE: MAJOR 2 (PARTIAL flip unsupported) — WAVE-6-MAP §Architectural decisions decision 3 AND §Stage-1 scope decisions PAR-UI-070/071 PARTIAL entries both document this; rows were MISSING before this plan and flip to PARTIAL (acknowledged variant), not HAVE.

Plan is actionable for kimi dispatch.
