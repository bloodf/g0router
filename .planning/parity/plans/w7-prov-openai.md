# Micro-plan w7-prov-openai — Config-only openai-format providers (catalog/models/aliases)

```
wave: 7
plan: w7-prov-openai
status: READY (rev 1 — authored against live tree @ 69f4981; 9router frozen @ 827e5c3)
track: CATALOG TRACK — fully disjoint. NO routes_admin.go, NO new packages, NO UI,
  NO e2e. Runs in parallel with everything for the whole wave (WAVE-7-MAP
  §Concurrency: "Catalog track parallel to everything … start it day one").
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w7-prov-openai:
ref-source: 9router frozen @ 827e5c3 —
  open-sse/config/providers.js (PROVIDERS map, lines 115-437),
  open-sse/config/providerModels.js (PROVIDER_MODELS map, lines 195-783).
base: <base> = git rev-parse HEAD recorded at P0 (expected 69f4981 at authoring;
  if main advanced, record the actual SHA and substitute everywhere §5 says <base>).
go-serial-slot: NONE. This plan does NOT touch internal/server/routes_admin.go
  (no new HTTP routes — generic routing already serves these; §1.4). It never
  holds the serial slot and never blocks the serial chain.
factory-micro-serial: NONE. This plan does NOT touch internal/inference/factory.go
  or selection.go (§1.4 confirms generic routing requires no change).
freeze: everything outside internal/providers/catalog/{catalog,models,aliases}.go
  (+ their _test.go), the matrix, and docs/WORKFLOW.md is FROZEN.
```

---

## 1. Scope — PAR rows

### Rows this plan closes (catalog config only; `.planning/parity/matrix/9router-providers.md`)

| Row | Provider(s) | ref format | Target state after w7-prov-openai |
|---|---|---|---|
| PAR-PROV-014 | openrouter | openai | HAVE (already in catalog — VERIFY + ensure models §1.5; matrix says HAVE, no g0router dir but catalog entry present) |
| PAR-PROV-029 | perplexity | openai | HAVE (already in catalog — VERIFY §1.5) |
| PAR-PROV-035 | glm-cn | openai | HAVE (NEW catalog+models entry) |
| PAR-PROV-037 | alicode, alicode-intl | openai | HAVE (NEW ×2) |
| PAR-PROV-038 | volcengine-ark | openai | HAVE (NEW; alias `ark`/`volcengine-ark` already present) |
| PAR-PROV-039 | byteplus | openai | HAVE (NEW; alias `byteplus`/`bpm` already present) |
| PAR-PROV-041 | nvidia | openai | HAVE (NEW; alias `nvidia` already present) |
| PAR-PROV-042 | cerebras | openai | HAVE (NEW; alias `cerebras` already present) |
| PAR-PROV-043 | nebius | openai | HAVE (NEW; alias `nebius` already present) |
| PAR-PROV-044 | siliconflow | openai | HAVE (NEW; alias `siliconflow` already present) |
| PAR-PROV-045 | hyperbolic | openai | HAVE (NEW; alias `hyp`/`hyperbolic` already present) |
| PAR-PROV-046 | xiaomi-mimo | openai | HAVE (NEW; alias `mimo`/`xiaomi-mimo` already present) |
| PAR-PROV-048 | opencode-go | openai | HAVE (catalog+models; alias `ocg` already present) — see §8 ESC-2 (specialized executor caveat) |
| PAR-PROV-049 | opencode | openai (noAuth) | HAVE (catalog only, empty model block per ref) — see §8 ESC-2 |
| PAR-PROV-050 | gitlab | openai | HAVE (catalog only — NO static model block in ref) |
| PAR-PROV-051 | codebuddy | openai | HAVE (catalog only — NO static model block; device-code auth caveat §8 ESC-3) |
| PAR-PROV-052 | vercel-ai-gateway | openai | HAVE (catalog only — NO static model block; alias present) |
| PAR-PROV-056 | chutes | openai | HAVE (catalog only — NO static model block; alias `ch`/`chutes` present) |
| PAR-PROV-057 | blackbox | openai | HAVE (NEW catalog+models; alias `bb`/`blackbox` present) |
| PAR-PROV-067 | free-tier bundle (29 providers) | openai (2 caveats) | HAVE (NEW catalog+models ×29; aliases ALL present §1.6) — see §8 ESC-4 (`agentrouter` is `format:"claude"`) |

### Rows in the task brief that ESCALATE (format divergence — NOT closed as openai here)

The task brief lists these as `format:"openai"`, but the 9router reference defines
them as **`format:"claude"`** (Anthropic Messages wire format). The g0router
generic adapter (`internal/providers/generic/`) is an **OpenAI-format** adapter
(`internal/inference/factory.go:108` → `generic.New(providerID)`), so a
`format:"claude"` catalog entry would route through the wrong converter. These are
recorded as **§8 ESC-1 (format-claude escalation)** and are NOT included in this
plan's committed openai set — they belong with the claude-format / specialized
adapter work, not the generic-openai catalog track:

| Row | Provider(s) | ref evidence | Why escalated |
|---|---|---|---|
| PAR-PROV-013 | minimax, minimax-cn | `providers.js:146-154` → `format:"claude"`, baseUrl `…/anthropic/v1/messages` | claude wire format; generic adapter is openai-only |
| PAR-PROV-034 | glm | `providers.js:131-134` → `format:"claude"`, baseUrl `https://api.z.ai/api/anthropic/v1/messages` | claude wire format |
| PAR-PROV-036 | kimi | `providers.js:141-144` → `format:"claude"`, baseUrl `https://api.kimi.com/coding/v1/messages` | claude wire format |

Note: **glm-cn (PAR-PROV-035) IS `format:"openai"`** (`providers.js:136-139`,
baseUrl `https://open.bigmodel.cn/api/coding/paas/v4/chat/completions`) and IS in
scope — do not confuse it with `glm` (claude). Same for `minimax` (claude, ESC-1)
vs the openai western providers. See §8 ESC-1 for the full disposition and the
matrix-flip implication (013/034/036 stay MISSING until the claude-format track).

### NOT in scope (explicit)

- **No `format:"claude"` providers** — PAR-PROV-013/034/036 (minimax/-cn, glm,
  kimi) and the free-tier `agentrouter` (§8 ESC-4) escalate; only `format:"openai"`
  (and the openai-shaped `noAuth` variants) are added here.
- **No specialized-format providers** — kiro/cursor/vertex/azure/cloudflare-ai/
  commandcode/perplexity-web/grok-web are w7-prov-special.
- **No new Go package, no adapter code** — catalog/models/aliases data entries ONLY
  (WAVE-7-MAP decision 5: "provider parity ≈ catalog config, NOT N new adapters").
- **No `internal/inference/factory.go` change** — confirmed unnecessary (§1.4).
- **No `routes_admin.go`, no HTTP route** — these providers have no dedicated admin
  route; the generic chat path serves them (§1.4).
- **No UI, no e2e, no mock** — WAVE-7-MAP row w7-prov-openai e2e impact = "none
  (catalog-only; no UI contract)". Confirmed: no `ui/e2e/mocks/handlers/*` models
  the provider catalog map; the catalog is consumed only by `internal/inference`.
- **No `ProviderConfig` struct change** — every field these providers need
  (`Name`, `BaseURL`, `Format`, `Headers`, `AuthHeader`, `NoAuth`, `Retry`) already
  exists (`catalog/catalog.go:6-18`). `enally`'s `authHeader:"x-api-key"` maps to
  `AuthHeader`; `uncloseai`/`opencode`'s `noAuth:true` maps to `NoAuth`.
- **No model `type`/`params`/`targetFormat` reshaping** — port `type`/`params`
  verbatim where the ref has them (`ModelEntry.Type`/`Params`, `models.go:8-14`);
  `targetFormat` has no `ModelEntry` field today and is NOT added (it is a
  read-site concern, deferred — note any provider whose default model needs it in
  §8 ESC-5; affects opencode-go's minimax-* entries and several `image` entries).

---

## 2. Architectural decisions grounding (evidence)

### 2.1 The generic adapter already routes every catalog provider (factory needs NO change)

`internal/inference/factory.go:104-109`:
```go
default:
    if _, ok := catalog.Lookup(providerID); !ok {
        return nil, fmt.Errorf("unknown provider %q", providerID)
    }
    return generic.New(providerID)
```
Any `providerID` present in `catalog.Providers` that is not one of the five built-in
switch cases (`openai`/`anthropic`/`gemini`/`ollama`/`ollama-local`,
`factory.go:96-103`, `isBuiltinProvider` `factory.go:113-118`) is constructed via
`generic.New(providerID)` (`internal/providers/generic/provider.go:22`). **Adding a
catalog entry is therefore sufficient to make a provider routable** — no factory
edit, no new package. **CONFIRMED: `factory.go` requires no change.** (The generic
adapter reads `catalog.Lookup` for BaseURL/Format/Headers/AuthHeader/NoAuth/Retry
at request time; the openai format is the generic adapter's native format.)

### 2.2 Template entries — 3 existing openai-format catalog entries (AGENTS.md "read 3 before writing")

Read these before adding new ones; they are the EXACT shape to mirror:
- **Plain openai (no headers):** `groq` (`catalog/catalog.go:29-33`) —
  `{Name,BaseURL,Format:"openai"}`. Western providers (nvidia, cerebras, nebius,
  hyperbolic, siliconflow, xiaomi-mimo, blackbox, glm-cn, alicode*, volcengine-ark,
  byteplus, gitlab, codebuddy, vercel-ai-gateway, chutes, opencode-go) follow this.
- **openai + custom headers:** `openrouter` (`catalog/catalog.go:59-67`) —
  `{Name,BaseURL,Format:"openai",Headers:{...}}`. No new openai provider in scope
  needs custom headers EXCEPT none (the ref free-tier/western entries carry empty
  `headers:{}` or none). `opencode` carries `headers:{"x-opencode-client":"desktop"}`
  — mirror that.
- **openai + NoAuth:** `ollama` is `format:"ollama"` NoAuth — for the openai-shaped
  NoAuth case the templates are `opencode` (`noAuth:true` + header) and `uncloseai`
  (`noAuth:true`). Set `NoAuth:true` (`catalog/catalog.go:11`,98) for these two.
- **AuthHeader override:** no existing entry sets `AuthHeader`; `enally` is the
  first (`AuthHeader:"x-api-key"`, ref `providers.js:416`). The field exists
  (`catalog/catalog.go:11`); this is its first use — assert it in the test.

### 2.3 Model catalog template (`catalog/models.go`)

`ModelEntry{ID,Name,UpstreamModelID,Type,Params}` (`models.go:8-14`). Ported
verbatim; `Type` empty when ref has none (NOT defaulted — `models.go:5-7,25-33`);
`Params` carried for `image`/`stt`/`tts` entries. `UpstreamModelID` defaults to
`ID` unless the ref sets `upstreamModelId` (e.g. deepseek `-max`/`-none`,
`models.go:21-22`). Providers with NO ref model block (gitlab, codebuddy,
vercel-ai-gateway, chutes) get **no `Models` entry** (or an explicit empty slice —
decide at T1; `opencode` has an empty block in ref → empty slice or omit; matrix
PAR-PROV-049 says "Empty static catalog (all models commented out)").

### 2.4 Aliases largely EXIST already (`catalog/aliases.go`)

`aliases_test.go:6-8` pins `ProviderAliasCount() == 133`. Most target aliases are
already present (`aliases.go`): `glm`(32), `kimi`(33), `minimax`(34),
`minimax-cn`(35), `cerebras`(49), `nvidia`(51), `nebius`(52), `siliconflow`(53),
`hyp`/`hyperbolic`(54-55), `ch`/`chutes`(62-63), `ark`/`volcengine-ark`(64-65),
`byteplus`/`bpm`(66-67), `mimo`/`xiaomi-mimo`(78-79), `vercel`/`vercel-ai-gateway`(27-28),
`bb`/`blackbox`(145-146), `ocg`→`opencode-go`(19), `oc`→`opencode`(18), and ALL 29
free-tier aliases (`aliases.go:101-147`). **Aliases that are MISSING and must be
ADDED** (verify at T1 with the §2.5 grep): `glm-cn`, `alicode`, `alicode-intl`,
`gitlab`, `codebuddy`. If adding aliases changes `ProviderAliasCount()`, the count
assertion in `aliases_test.go:7` MUST be updated to the new total in the SAME
commit (it is a w7-prov-openai-owned test file).

### 2.5 Pre-write verification greps (run at T1 to confirm the alias/format facts)

```bash
# which target aliases already resolve (expect most present):
for a in glm-cn alicode alicode-intl gitlab codebuddy nvidia cerebras nebius \
  hyperbolic siliconflow xiaomi-mimo blackbox vercel-ai-gateway chutes \
  volcengine-ark byteplus opencode-go opencode; do
  echo -n "$a: "; grep -c "\"$a\"" internal/providers/catalog/aliases.go
done
# confirm format:"claude" providers are NOT to be added as openai (ESC-1):
grep -nE '(glm|kimi|minimax|minimax-cn|agentrouter):.*format:.*"claude"' \
  /home/cortexos/Developer/github.com/bloodf/_refs/9router/open-sse/config/providers.js
```

---

## 3. Exclusive file ownership

After w7-prov-openai merges, the entries added below are owned by this plan; the
files are SHARED catalog files (other w7-prov-* plans append disjoint entries to
the same maps — coordinate as additive, key-disjoint appends; no two prov plans add
the same provider key).

**MODIFY — catalog data (ADDITIVE map entries only; no struct change):**

| File | Change |
|---|---|
| `internal/providers/catalog/catalog.go` | ADD openai-format `ProviderConfig` entries to the `Providers` map (`catalog.go:28`) per the §6 table: glm-cn, alicode, alicode-intl, volcengine-ark, byteplus, nvidia, cerebras, nebius, siliconflow, hyperbolic, xiaomi-mimo, blackbox, gitlab, codebuddy, vercel-ai-gateway, chutes, opencode-go, opencode (NoAuth+header), + 29 free-tier entries (enally with `AuthHeader:"x-api-key"`, uncloseai with `NoAuth:true`). VERIFY openrouter/perplexity already present (no-op). NO change to existing entries. NO `ProviderConfig` struct change. |
| `internal/providers/catalog/models.go` | ADD `Models` map entries (`models.go:18`) per §6 for every provider that HAS a ref model block (glm-cn, alicode, alicode-intl, volcengine-ark, byteplus, nvidia, cerebras, nebius, siliconflow, xiaomi-mimo, hyperbolic, blackbox, opencode-go, + the 28 free-tier providers that have a block; verify each at T2). NO `Models` entry for gitlab/codebuddy/vercel-ai-gateway/chutes (no ref block) and opencode (empty ref block). Port `Type`/`Params` verbatim. |
| `internal/providers/catalog/aliases.go` | ADD the MISSING aliases ONLY (§2.4: glm-cn, alicode, alicode-intl, gitlab, codebuddy — verify exact set at T1). Existing aliases unchanged. |

**MODIFY — tests (TDD; written/extended RED before the data entries, §4):**

| File | Change |
|---|---|
| `internal/providers/catalog/catalog_test.go` | ADD per-family table tests asserting each new provider Lookup returns the right `BaseURL`/`Format`/`NoAuth`/`AuthHeader` (§4 T-family tests). |
| `internal/providers/catalog/models_test.go` | ADD per-family table tests asserting `ModelsFor(p)` has the expected count and key model IDs; assert `Type`/`Params` for the typed entries (nvidia embedding/stt, minimax/cloudflare image — n/a here; the in-scope typed entries are nvidia `embedding`+`stt`). |
| `internal/providers/catalog/aliases_test.go` | UPDATE `ProviderAliasCount()` expected total if aliases were added (§2.4); ADD sample assertions for the newly-added aliases. |

**MODIFY — matrix + workflow (closeout):**

| File | Change |
|---|---|
| `.planning/parity/matrix/9router-providers.md` | Flip PAR-PROV-035/037/038/039/041/042/043/044/045/046/048/049/050/051/052/056/057/067 → HAVE; VERIFY 014/029 already HAVE. Annotate 013/034/036 with the ESC-1 format-claude note (stay MISSING). |
| `docs/WORKFLOW.md` | Record P0 base SHA, the resolved alias-count delta, the §8 escalations, and closeout. |

**FORBIDDEN:** everything else. Explicitly: `internal/inference/factory.go`,
`internal/inference/selection.go`, `internal/providers/generic/**`,
`internal/providers/{kiro,cursor,vertex,minimax,...}/**`,
`internal/server/routes_admin.go`, any `internal/admin/**`, any `ui/**`, any
`ui/e2e/**` mock/spec, `internal/schemas/**`, `internal/store/**`. NO
`ProviderConfig`/`ModelEntry` struct changes. The catalog files are touched ONLY to
add map entries + their covering tests.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always: test first, see it fail, minimum code to
pass; no mocks"): **no provider/model/alias entry is added before the failing test
that asserts it is committed RED.** `go test ./internal/providers/catalog/ -run
<Name>` must FAIL (asserting absence) before the data entry lands, then GREEN after.
`go test ./... && go vet ./... && go build ./...` green at EVERY commit. Use
golden/table tests per FAMILY (Chinese / Western / no-catalog / free-tier) so each
commit is a coherent RED→GREEN family slice.

### T1 — STEP(a): verify facts + write RED tests for the Chinese-openai family

Run the §2.5 greps; record the alias-present/absent set and re-confirm the
format-claude escalation set in WORKFLOW.md. Then ADD to `catalog_test.go` a
table test `TestChineseOpenAIProviders` asserting Lookup for **glm-cn, alicode,
alicode-intl, volcengine-ark, byteplus, xiaomi-mimo, opencode-go** returns the §6
BaseURL + `Format=="openai"` (opencode-go via API key); ADD to `models_test.go`
`TestChineseOpenAIModels` asserting `ModelsFor` counts/key-IDs (glm-cn=5,
alicode=8, alicode-intl=7, volcengine-ark=9, byteplus=7, xiaomi-mimo=4,
opencode-go=10 per ref). Run `go test ./internal/providers/catalog/ -run
'Chinese'` → **FAILS** (entries absent). Commit RED:
`phase-1/w7-prov-openai: failing Chinese openai-family catalog/model tests (TDD red)`.

### T1 — STEP(b): add the Chinese-openai catalog + model + alias entries

Add the §6 `Providers`/`Models` entries; add the missing aliases (glm-cn, alicode,
alicode-intl). If alias count changed, update `aliases_test.go` count. Gates:
`go test ./internal/providers/catalog/ -run 'Chinese|Alias' -v` green;
`go test ./... && go vet ./... && go build ./...` green. Commit:
`phase-1/w7-prov-openai: Chinese openai providers (glm-cn, alicode*, ark, byteplus, xiaomi-mimo, opencode*)`.

### T2 — STEP(a)/(b): Western-openai family (RED → GREEN)

RED: `TestWesternOpenAIProviders` (nvidia, cerebras, nebius, hyperbolic,
siliconflow, blackbox + the no-catalog quartet gitlab/codebuddy/vercel-ai-gateway/
chutes) Lookup asserts §6 BaseURL+Format; `TestWesternOpenAIModels`
(nvidia=4 incl. 1 embedding + 1 stt with Params, cerebras=6, nebius=2 incl. 1
embedding, hyperbolic=8, siliconflow=10, blackbox=17) + assert gitlab/codebuddy/
vercel/chutes have EMPTY `ModelsFor` (no block). Run → FAILS. Commit RED. Then add
entries + missing aliases (gitlab, codebuddy). Gates green. Commit:
`phase-1/w7-prov-openai: Western openai providers (nvidia, cerebras, nebius, hyperbolic, siliconflow, blackbox, gitlab, codebuddy, vercel, chutes)`.

### T3 — STEP(a)/(b): Free-tier bundle (PAR-PROV-067, 29 providers minus agentrouter)

RED: `TestFreeTierProviders` table asserting Lookup for the 28 openai free-tier
providers (`providers.js:407-437`: aimlapi, novita, modal, reka, nlpcloud,
bazaarlink, completions, enally [AuthHeader x-api-key], freetheai, llm7, lepton,
kluster, ai21, inference-net, predibase, bytez, morph, longcat, puter, uncloseai
[NoAuth], scaleway, deepinfra, sambanova, nscale, baseten, publicai, nous-research,
glhf) returns §6 BaseURL+Format+NoAuth/AuthHeader where applicable; assert
`enally.AuthHeader=="x-api-key"` and `uncloseai.NoAuth==true`.
`TestFreeTierModels` asserts each has its ref model block (counts spot-checked:
agentrouter excluded; modal=1, etc. — port verbatim at T3(b)). Aliases all present
(`aliases.go:101-147`) — assert a sample. Run → FAILS. Commit RED. Add entries.
Gates green. Commit:
`phase-1/w7-prov-openai: free-tier openai bundle (28 providers, PAR-PROV-067)`.
**`agentrouter` (format:claude) is EXCLUDED — §8 ESC-4.**

### T4 — full gates + closeout

```bash
go test ./internal/providers/catalog/... -v
go test ./... && go vet ./... && go build ./...
```
Flip the §1 matrix rows in `.planning/parity/matrix/9router-providers.md`
(035/037/038/039/041/042/043/044/045/046/048/049/050/051/052/056/057/067 → HAVE;
014/029 verify HAVE; annotate 013/034/036 with ESC-1). Append the open questions
(§8) to `.planning/parity/plans/open-questions.md`. Update `docs/WORKFLOW.md`. Final
commit: `phase-1/w7-prov-openai: close — openai catalog providers; matrix flips`.

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0 (69f4981 at
authoring). Diff gate is w7-prov-openai commit-range-scoped (§7).

**Test gates**
- `go test ./internal/providers/catalog/ -run 'Chinese' -v` → exit 0.
- `go test ./internal/providers/catalog/ -run 'Western' -v` → exit 0.
- `go test ./internal/providers/catalog/ -run 'FreeTier' -v` → exit 0.
- `go test ./internal/providers/catalog/... -v` → exit 0 (all prior + new).
- `go test ./... && go vet ./... && go build ./...` → exit 0.

**TDD-order proof** — each family's data lands in a commit AFTER its RED test commit:
```bash
R="<first-w7>^..<last-w7>"
# Chinese family: RED test commit precedes the entry commit
rc=$(git log --format=%ct -1 --grep="failing Chinese openai-family")
dc=$(git log --format=%ct -1 --grep="Chinese openai providers")
[ "$rc" -le "$dc" ] || echo "TDD VIOLATION: Chinese family"   # prints nothing
# (repeat for Western and free-tier commits)
```

**Grep proofs (per provider family — each name present with the right base_url)**
```bash
C=internal/providers/catalog/catalog.go
M=internal/providers/catalog/models.go
# Chinese-openai base URLs:
grep -q 'open.bigmodel.cn/api/coding/paas/v4/chat/completions' $C   # glm-cn (035)
grep -q 'coding.dashscope.aliyuncs.com/v1/chat/completions' $C      # alicode (037)
grep -q 'coding-intl.dashscope.aliyuncs.com/v1/chat/completions' $C # alicode-intl (037)
grep -q 'ark.cn-beijing.volces.com/api/coding/v3/chat/completions' $C  # volcengine-ark (038)
grep -q 'ark.ap-southeast.bytepluses.com/api/coding/v3/chat/completions' $C  # byteplus (039)
grep -q 'api.xiaomimimo.com/v1/chat/completions' $C                 # xiaomi-mimo (046)
grep -q 'opencode.ai/zen/go/v1/chat/completions' $C                 # opencode-go (048)
grep -q '"opencode"' $C                                             # opencode (049, noAuth)
# Western-openai base URLs:
grep -q 'integrate.api.nvidia.com/v1/chat/completions' $C          # nvidia (041)
grep -q 'api.cerebras.ai/v1/chat/completions' $C                   # cerebras (042)
grep -q 'api.studio.nebius.ai/v1/chat/completions' $C              # nebius (043)
grep -q 'api.siliconflow.cn/v1/chat/completions' $C                # siliconflow (044)
grep -q 'api.hyperbolic.xyz/v1/chat/completions' $C                # hyperbolic (045)
grep -q 'gitlab.com/api/v4/chat/completions' $C                    # gitlab (050)
grep -q 'copilot.tencent.com/v1/chat/completions' $C               # codebuddy (051)
grep -q 'ai-gateway.vercel.sh/v1/chat/completions' $C              # vercel-ai-gateway (052)
grep -q 'llm.chutes.ai/v1/chat/completions' $C                     # chutes (056)
grep -q 'api.blackbox.ai/chat/completions' $C                      # blackbox (057)
# Free-tier (sample — 28 entries, PAR-PROV-067):
grep -q 'ai.enally.in/v1/chat/completions' $C                      # enally
grep -q 'AuthHeader: *"x-api-key"' $C                              # enally x-api-key (first use)
grep -q 'hermes.ai.unturf.com/v1/chat/completions' $C             # uncloseai
# format is openai for every new entry (no accidental "claude"):
! grep -nE 'glm-cn|alicode|nvidia|cerebras|blackbox|enally' $C | grep -q '"claude"' && echo "no stray claude OK"
# model-catalog presence/absence:
grep -q '"glm-cn"' $M && grep -q '"blackbox"' $M && grep -q '"nvidia"' $M  # have blocks
! grep -q '"gitlab"' $M && ! grep -q '"chutes"' $M && echo "no-catalog providers correctly absent from Models"
# format-claude escalation honored — NOT added as openai:
! grep -qE '"glm":|"kimi":|"minimax":' $C && echo "ESC-1 claude providers NOT added OK"
```

**No-struct-change / no-out-of-scope proofs**
```bash
git diff $R -- internal/providers/catalog/catalog.go | grep -E '^\+' | grep -qE 'type ProviderConfig|func ' && echo "STRUCT/FUNC CHANGE — REJECT" || echo "additive entries only OK"
git diff $R --name-only | grep -vE 'internal/providers/catalog/(catalog|models|aliases)(_test)?\.go|\.planning/parity/matrix/9router-providers\.md|\.planning/parity/plans/open-questions\.md|docs/WORKFLOW\.md' | wc -l  # = 0
```

**Freeze proofs (commit-range — §7)**
```bash
git diff $R --name-only -- internal/inference/factory.go internal/inference/selection.go | wc -l  # = 0 (factory unchanged)
git diff $R --name-only -- internal/server/routes_admin.go | wc -l   # = 0 (no serial slot)
git diff $R --name-only -- internal/providers/generic/ internal/admin/ ui/ | wc -l  # = 0
```

---

## 6. Per-provider data table (name → base_url → models/default → alias → PAR row)

All base URLs and model lists are transcribed from 9router @ 827e5c3
(`open-sse/config/providers.js`, `providerModels.js`). `Format="openai"` for every
row below. "default" = first model in the ref block (the catalog has no explicit
default field; first-entry is the convention). Aliases marked **(have)** already
exist in `aliases.go`; **(ADD)** must be added.

### Chinese-openai family

| Provider | base_url (ref line) | models (count; first=default) | alias | PAR |
|---|---|---|---|---|
| glm-cn | `https://open.bigmodel.cn/api/coding/paas/v4/chat/completions` (prov:137) | 5 — glm-5.1(def), glm-5, glm-4.7, glm-4.6, glm-4.5-air (mod:327-333) | `glm-cn` **(ADD)** | 035 |
| alicode | `https://coding.dashscope.aliyuncs.com/v1/chat/completions` (prov:157) | 8 — qwen3.5-plus(def), kimi-k2.5, glm-5, MiniMax-M2.5, qwen3-max-2026-01-23, qwen3-coder-next, qwen3-coder-plus, glm-4.7 (mod:373-382) | `alicode` **(ADD)** | 037 |
| alicode-intl | `https://coding-intl.dashscope.aliyuncs.com/v1/chat/completions` (prov:162) | 7 — qwen3.5-plus(def), kimi-k2.5, glm-5, MiniMax-M2.5, qwen3-coder-next, qwen3-coder-plus, glm-4.7 (mod:383-391) | `alicode-intl` **(ADD)** | 037 |
| volcengine-ark | `https://ark.cn-beijing.volces.com/api/coding/v3/chat/completions` (prov:167) | 9 — Doubao-Seed-2.0-Code(def), Doubao-Seed-2.0-pro, Doubao-Seed-2.0-lite, Doubao-Seed-Code, DeepSeek-V4-Flash, DeepSeek-V4-Pro, GLM-5.1, MiniMax-M2.7, Kimi-K2.6 (mod:392-402) | `ark`/`volcengine-ark` **(have)** | 038 |
| byteplus | `https://ark.ap-southeast.bytepluses.com/api/coding/v3/chat/completions` (prov:172) | 7 — seed-2-0-pro-260328(def), seed-2-0-code-preview-260328, seed-2-0-mini-260215, seed-2-0-lite-260228, kimi-k2-thinking-251104, glm-4-7-251222, gpt-oss-120b-250805 (mod:429-437) | `byteplus`/`bpm` **(have)** | 039 |
| xiaomi-mimo | `https://api.xiaomimimo.com/v1/chat/completions` (prov:395) | 4 — mimo-v2.5-pro(def), mimo-v2.5, mimo-v2-omni, mimo-v2-flash (mod:545-549) | `mimo`/`xiaomi-mimo` **(have)** | 046 |
| opencode-go | `https://opencode.ai/zen/go/v1/chat/completions` (prov:370) | 10 — kimi-k2.6(def), kimi-k2.5, glm-5.1, glm-5, qwen3.5-plus, qwen3.6-plus, mimo-v2-pro, mimo-v2-omni, minimax-m2.7*, minimax-m2.5* (mod:195-206) [*targetFormat:claude — §8 ESC-5] | `ocg`/`opencode-go` **(have)** | 048 |
| opencode | `https://opencode.ai` (prov:364) — `NoAuth:true`, `Headers:{"x-opencode-client":"desktop"}` | none (ref block all commented out, mod:207-213) | `oc`/`opencode` **(have)** | 049 |

### Western-openai family

| Provider | base_url (ref line) | models (count; first=default) | alias | PAR |
|---|---|---|---|---|
| nvidia | `https://integrate.api.nvidia.com/v1/chat/completions` (prov:249) | 4 — minimaxai/minimax-m2.7(def), z-ai/glm4.7, nvidia/nv-embedqa-e5-v5 (embedding), nvidia/parakeet-ctc-1.1b-asr (stt, Params:[language]) (mod:513-518) | `nvidia` **(have)** | 041 |
| cerebras | `https://api.cerebras.ai/v1/chat/completions` (prov:298) | 6 — gpt-oss-120b(def), zai-glm-4.7, llama-3.3-70b, llama-4-scout-17b-16e-instruct, qwen-3-235b-a22b-instruct-2507, qwen-3-32b (mod:500-506) | `cerebras` **(have)** | 042 |
| nebius | `https://api.studio.nebius.ai/v1/chat/completions` (prov:306) | 2 — meta-llama/Llama-3.3-70B-Instruct(def), Qwen/Qwen3-Embedding-8B (embedding) (mod:520-522) | `nb`? NO — `nebius` **(have)** | 043 |
| siliconflow | `https://api.siliconflow.cn/v1/chat/completions` (prov:310) | 10 — deepseek-ai/DeepSeek-V3.2(def) … baidu/ERNIE-4.5-300B-A47B (mod:533-543) | `siliconflow` **(have)** | 044 |
| hyperbolic | `https://api.hyperbolic.xyz/v1/chat/completions` (prov:314) | 8 — Qwen/QwQ-32B(def), deepseek-ai/DeepSeek-R1, deepseek-ai/DeepSeek-V3, meta-llama/Llama-3.3-70B-Instruct, meta-llama/Llama-3.2-3B-Instruct, Qwen/Qwen2.5-72B-Instruct, Qwen/Qwen2.5-Coder-32B-Instruct, NousResearch/Hermes-3-Llama-3.1-70B (mod:562-570) | `hyp`/`hyperbolic` **(have)** | 045 |
| blackbox | `https://api.blackbox.ai/chat/completions` (prov:437) | 17 — gpt-4o(def), gpt-4o-mini, claude-sonnet-4.6, claude-sonnet-4.5, claude-opus-4.6, claude-sonnet-4-6, claude-opus-4-6, deepseek-chat, deepseek-v3-671b, deepseek-r1, o1, o3-mini, gemini-2.5-flash, gemini-3-flash-preview, qwen3-coder-plus, qwen3-max, qwen3-vl-plus (mod:348-365) | `bb`/`blackbox` **(have)** | 057 |
| gitlab | `https://gitlab.com/api/v4/chat/completions` (prov:355) | none (no ref block) | `gitlab` **(ADD)** | 050 |
| codebuddy | `https://copilot.tencent.com/v1/chat/completions` (prov:360) | none (no ref block; device-code auth §8 ESC-3) | `codebuddy` **(ADD)** | 051 |
| vercel-ai-gateway | `https://ai-gateway.vercel.sh/v1/chat/completions` (prov:128) | none (no ref block) | `vercel`/`vercel-ai-gateway` **(have)** | 052 |
| chutes | `https://llm.chutes.ai/v1/chat/completions` (prov:330) | none (no ref block) | `ch`/`chutes` **(have)** | 056 |

### Free-tier bundle (PAR-PROV-067) — base_urls from `providers.js:406-437`

28 openai entries (agentrouter excluded → ESC-4). All aliases present
(`aliases.go:101-147`). All have a ref model block (`providerModels.js:641-783`).

| Provider | base_url | special | alias |
|---|---|---|---|
| aimlapi | `https://api.aimlapi.com/v1/chat/completions` | — | `aiml`/`aimlapi` (have) |
| novita | `https://api.novita.ai/v3/openai/chat/completions` | — | `novita` (have) |
| modal | `https://api.modal.com/v1/chat/completions` | — | `mdl`/`modal` (have) |
| reka | `https://api.reka.ai/v1/chat/completions` | — | `reka` (have) |
| nlpcloud | `https://api.nlpcloud.io/v1/gpu/chatbot` | — | `nlpc`/`nlpcloud` (have) |
| bazaarlink | `https://bazaarlink.ai/api/v1/chat/completions` | — | `bzl`/`bazaarlink` (have) |
| completions | `https://completions.me/api/v1/chat/completions` | — | `cpl`/`completions` (have) |
| enally | `https://ai.enally.in/v1/chat/completions` | `AuthHeader:"x-api-key"` | `enly`/`enally` (have) |
| freetheai | `https://api.freetheai.xyz/v1/chat/completions` | — | `fta`/`freetheai` (have) |
| llm7 | `https://api.llm7.io/v1/chat/completions` | — | `llm7` (have) |
| lepton | `https://api.lepton.ai/api/v1/chat/completions` | — | `lepton` (have) |
| kluster | `https://api.kluster.ai/v1/chat/completions` | — | `kluster` (have) |
| ai21 | `https://api.ai21.com/studio/v1/chat/completions` | — | `ai21` (have) |
| inference-net | `https://api.inference.net/v1/chat/completions` | — | `inet`/`inference-net` (have) |
| predibase | `https://serving.app.predibase.com/v1/chat/completions` | — | `predibase` (have) |
| bytez | `https://api.bytez.com/models/v2` | — | `bytez` (have) |
| morph | `https://api.morphllm.com/v1/chat/completions` | — | `morph` (have) |
| longcat | `https://api.longcat.chat/openai/v1/chat/completions` | — | `lc`/`longcat` (have) |
| puter | `https://api.puter.com/puterai/openai/v1/chat/completions` | — | `pu`/`puter` (have) |
| uncloseai | `https://hermes.ai.unturf.com/v1/chat/completions` | `NoAuth:true` | `unc`/`uncloseai` (have) |
| scaleway | `https://api.scaleway.ai/v1/chat/completions` | — | `scw`/`scaleway` (have) |
| deepinfra | `https://api.deepinfra.com/v1/openai/chat/completions` | — | `deepinfra` (have) |
| sambanova | `https://api.sambanova.ai/v1/chat/completions` | — | `samba`/`sambanova` (have) |
| nscale | `https://inference.api.nscale.com/v1/chat/completions` | — | `nscale` (have) |
| baseten | `https://inference.baseten.co/v1/chat/completions` | — | `baseten` (have) |
| publicai | `https://api.publicai.co/v1/chat/completions` | — | `publicai` (have) |
| nous-research | `https://inference-api.nousresearch.com/v1/chat/completions` | — | `nous`/`nous-research` (have) |
| glhf | `https://glhf.chat/api/openai/v1/chat/completions` | — | `glhf` (have) |

(Model lists for each free-tier provider: port verbatim from
`providerModels.js:641-783` at T3(b); spot-checked blocks present for all 28.)

### Already-present (VERIFY only, do not duplicate)

| Provider | base_url (catalog.go) | PAR |
|---|---|---|
| openrouter | `https://openrouter.ai/api/v1/chat/completions` (catalog.go:61) | 014 |
| perplexity | `https://api.perplexity.ai/chat/completions` (catalog.go:78) | 029 |

---

## 7. Diff-gate scope

Catalog plans may run concurrently with other w7-prov-* plans appending to the same
maps; isolate w7-prov-openai's commits:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w7-prov-openai:" | awk '{print $1}'`
then `git diff <first>^..<last> --name-only` must be exactly a subset of:

```
internal/providers/catalog/catalog.go
internal/providers/catalog/catalog_test.go
internal/providers/catalog/models.go
internal/providers/catalog/models_test.go
internal/providers/catalog/aliases.go
internal/providers/catalog/aliases_test.go
.planning/parity/matrix/9router-providers.md
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```

Any file outside this list is an automatic REJECT. `internal/inference/factory.go`,
`internal/server/routes_admin.go`, `internal/providers/generic/**`, all `ui/**`,
all `internal/admin/**`, and any struct change to `ProviderConfig`/`ModelEntry` are
deliberately ABSENT — touching them is an automatic REJECT. If a concurrent
w7-prov-* plan adds the SAME provider key to a shared map, that is a merge conflict
to resolve at the orchestrator level (key-disjoint append rule).

---

## 8. Escalations / open questions

- **ESC-1 (format-claude — RESOLVED disposition, binding): PAR-PROV-013, 034, 036
  are NOT openai.** The 9router ref defines minimax/minimax-cn (`providers.js:146-154`),
  glm (`providers.js:131-134`), kimi (`providers.js:141-144`) as `format:"claude"`
  with `…/anthropic/v1/messages` / `…/v1/messages` base URLs. The g0router generic
  adapter is OpenAI-format only (`factory.go:108`). **Decision:** EXCLUDE them from
  this plan; they belong to the claude-format/specialized track (the WAVE-7-MAP
  w7-prov-special plan handles specialized formats; claude-format catalog entries
  need either a claude-capable generic path or per-provider adapters). These rows
  STAY MISSING after w7-prov-openai. Annotate the matrix rows with this rationale.
  **Open question for the orchestrator:** which W7 plan owns the `format:"claude"`
  API-key providers (glm/kimi/minimax/minimax-cn/agentrouter)? They are not
  specialized adapters (no protobuf/eventstream) — they are Anthropic-wire-format
  API-key providers that need a claude generic path. NOT a w7-prov-openai blocker.
- **ESC-2 (opencode/opencode-go specialized executor — recorded, non-blocking):**
  the matrix notes `OpenCodeGoExecutor` is specialized (PAR-PROV-048) and the first
  `opencode` ref entry (line 233) is overwritten by the line-363 noAuth entry
  (PAR-PROV-049). For this catalog-track plan we add the catalog config (base_url +
  format + noAuth) which makes them routable via the generic adapter for the basic
  OpenAI-compatible path. If opencode-go's subscription auth / token-exchange
  differs from a plain bearer key, that auth-flow work is a separate
  w7-prov-oauth/special concern — the catalog entry is still correct and sufficient
  for parity HAVE of the catalog row. Flag in open-questions.
- **ESC-3 (codebuddy device-code auth — recorded):** matrix says codebuddy "uses
  device_code polling auth" (`providers.js:359-361`). The base_url is a normal
  openai chat-completions endpoint, so the catalog entry is correct; the
  device-code OAuth acquisition is a w7-prov-oauth concern, not this plan. Catalog
  HAVE is satisfied by the config entry.
- **ESC-4 (agentrouter is claude — EXCLUDED from PAR-PROV-067 openai set):** within
  the free-tier bundle, `agentrouter` is `format:"claude"` (`providers.js:406`,
  `headers:{...CLAUDE_CLI_SPOOF_HEADERS}`). It is EXCLUDED here and rides ESC-1.
  PAR-PROV-067 flips to HAVE for the **28 openai** free-tier providers; agentrouter
  remains a tracked sub-item under ESC-1. **Open question:** does the operator
  accept PAR-PROV-067 as HAVE with agentrouter carved out, or hold the row until
  the claude path lands? Recommend: flip 067 HAVE (28/29 openai), footnote
  agentrouter.
- **ESC-5 (targetFormat field absent — recorded, deferred):** several ref model
  entries carry `targetFormat:"claude"` (opencode-go minimax-m2.7/m2.5,
  minimax MiniMax-M3, xiaomi-tokenplan mimo-v2.5-pro-claude). `ModelEntry` has no
  `TargetFormat` field (`models.go:8-14`) and this plan does NOT add one (no struct
  change). The models are still ported (ID/Name/UpstreamModelID); the per-model
  claude routing is a read-site/Stage-2 concern. Note: for opencode-go this means
  the two minimax-* models would route as openai until targetFormat is honored —
  acceptable for catalog-row HAVE; flag in open-questions for the read-site wave.
- **ESC-6 (no-catalog providers — informational):** gitlab, codebuddy,
  vercel-ai-gateway, chutes have NO static model block in the ref. Their rows flip
  HAVE on the catalog (base_url+format) entry alone; models are discovered at
  runtime via the provider's `/models` endpoint (out of this plan's scope). Tests
  assert `ModelsFor(p)` is EMPTY for these (not a failure).

All ESC items above are appended to `.planning/parity/plans/open-questions.md` at
T4 closeout (per the Planner Open_Questions protocol).
```
