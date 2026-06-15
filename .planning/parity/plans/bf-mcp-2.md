# Micro-plan bf-mcp-2 — per-VK MCP scoping + tool filtering (Go)

```
program: bifrost-parity (Waves 6+; CLI_ORCHESTRATOR.md governs, HANDOFF.md retired)
plan: bf-mcp-2
status: READY (rev 1 — authored against the SHIPPED bf-mcp-1 server-mode tree
  @ HEAD 37a1c5d: internal/mcp/server.go (Dispatch over initialize/tools/list/
  tools/call), internal/admin/mcp.go (MCPServerPost/MCPServerSSE + resolveMCPVK +
  admitMCPVK + the SHARED catalog assembler mcpToolCatalog/assembleServerCatalog +
  serverCatalogSource + newMCPServer + the deferred recordAudit), internal/server/
  routes_mcp.go (POST/GET /mcp). REUSES the Wave-7 CLIENT-mode filter primitives
  internal/mcp/{filter,allowlist,toolpolicy}.go + internal/store/mcptoolgroups.go.
  BIFROST-MAP.md ledger row bf-mcp-2 §304; bifrost-mcp disposition §266-267; serial
  chain §332-346; freeze rules §384-399. PRIMARY structural template = bf-mcp-1.md
  (the server-mode seams this plan plugs into). PRIMARY additive-storage template =
  bf-gov-1.md (the VK↔Team team_id additive precedent; here generalized to a
  many-to-many VK↔MCP assignment table mirroring the mcp_tool_groups INTEGER-PK
  additive-table pattern). REUSE template = toolpolicy.go buildToolPolicy (the
  bare + "<server>-<tool>" dual-key wildcard primitive).)
runs: MCP track. bf-mcp-2 is the SECOND holder of the routes_admin.go MCP serial
  chain (BIFROST-MAP §343-346: bf-mcp-1 → bf-mcp-2 → bf-core-2). bf-mcp-1 RELEASED
  the slot on close. Reuses internal/mcp CLIENT filter infra (consume; do NOT
  rebuild). Runs ∥ openai + gov tracks (disjoint domain). RELEASES the
  routes_admin.go slot to bf-core-2 on close.
branch: main (direct commits, per AGENTS.md "No PR workflow")
commit-prefix: phase-1/bf-mcp-2:
footer: Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
ref-source: BLOCKED — ESC-REF-ABSENT (BIFROST-MAP §47-68). The frozen Bifrost ref
  @ ca21298 is ABSENT on this host. The ONLY ground truth is
  .planning/parity/matrix/bifrost-mcp.md. Build to documented matrix behavior +
  g0router's own conventions + the MCP/wildcard semantics the matrix rows name.
  NEVER build to a guessed Bifrost wire/schema detail. Any column, wildcard rule,
  or filtering edge the matrix note does not capture → STOP-escalate the affected
  row (§3); do NOT invent Bifrost's schema.
go-serial-slot: internal/server/routes_admin.go — bf-mcp-2 TAKES the SECOND MCP-chain
  slot (BIFROST-MAP §343-346). Additive admin routes for the VK↔MCP assignment
  surface register here (or in the existing internal/server/routes_mcp.go that is
  already CALLED from RegisterAdminRoutes — bf-mcp-1 D2). Either way the registration
  edit is the serial-held surface. RELEASE to bf-core-2 on close.
new-route: MAYBE — additive admin CRUD for the VK↔MCP assignment table IF the
  assignment must be settable over the API (D-routes). NO new /mcp JSON-RPC route
  (bf-mcp-1 owns POST/GET /mcp; bf-mcp-2 only CHANGES what a scoped VK sees there).
```

---

## 0. Objective + ground truth

### 0.1 Objective

Make the SHIPPED `/mcp` server-mode surface **per-VK scoped**. bf-mcp-1 resolves +
validates the VK (`admitMCPVK` @ `internal/admin/mcp.go:706`) but serves EVERY
admitted caller the SAME global catalog (`assembleServerCatalog` @ `:660`). bf-mcp-2
closes that documented next step: when a VK is resolved on `/mcp`, the `tools/list`
it sees (and the `tools/call` it may invoke) is **FILTERED to that VK's scope** via a
per-VK `executeOnlyTools` wildcard filter, with an `AllowOnAllVirtualKeys` per-MCP-
client flag that **bypasses** the filter for tools whose owning client is marked
all-VK-visible. The VK→tool scope is read from an **additive VK↔MCP assignment table**.

Additive-only. The plan adds:
- a NEW `internal/store/mcpvkconfigs.go` — the VK↔MCP assignment store (additive
  `ensureTable` `virtual_key_mcp_configs`, mirroring the `mcp_tool_groups`
  INTEGER-PK additive-table pattern, `internal/store/mcptoolgroups.go`);
- a NEW pure wildcard scope filter in `internal/mcp/` (REUSING the
  `toolpolicy.go buildToolPolicy` bare + `<server>-<tool>` dual-key primitive and the
  `stripServerPrefix` idempotent strip) that narrows a `[]ServerTool` to the names a
  VK's `executeOnlyTools` patterns admit;
- a per-VK `CatalogSource` adapter wired through the SHIPPED `newMCPServer` seam so a
  scoped VK's `tools/list` returns FEWER tools than the global catalog;
- `ToolsToExecute`/`ToolsToAutoExecute` subset-validation in the live assignment
  admin path (a `ToolsToAutoExecute` not ⊆ `ToolsToExecute` is rejected);
- the **config-only flags** the matrix names: `DisableAutoToolInject`,
  `AllowedExtraHeaders` whitelist, tool-annotations mapping, `IsCodeModeClient`,
  `ConfigHash` — STORED + operator-set/observable through the admin API
  (`ConfigHash` is computed on write AND exposed in the assignment GET DTO for
  drift-detection — its live reader); **code-mode VFS execution is ESC** (flag stored,
  execution engine NOT built).

This is the bifrost-mcp analogue of BIFROST-MAP decision #6: the per-VK scoping
layer is genuinely NEW; the wildcard/allowlist/tool-group primitives are REUSABLE
adjacent CLIENT-mode infrastructure. Per-user OAuth/header/sessions surfaces, the
plugin pipeline for nested tools, and code-mode VFS execution are OUT (§3 ESC).

### 0.2 Rows this plan closes (matrix = ground truth)

| Row | Behavior (matrix) | Disposition | Flip target after bf-mcp-2 |
|---|---|---|---|
| PAR-BF-MCP-004 | Per-VK MCP server with lazy creation | **BUILD (VAR — request-time variant)** | g0router has ONE global server; "lazy creation" = the per-VK scope computed on-demand at request time from the assignment table (D2/D3), NOT a long-lived per-VK server object. MISSING→**HAVE-by-variant** (request-time-filter variant; per-VK long-lived server-instance architecture is VAR, recorded). |
| PAR-BF-MCP-017 | Per-client allow-list `ToolsToExecute` (`["*"]`=all, nil/`[]`=deny) | **BUILD** | The wildcard scope filter (D4) implements the `*`/empty=deny semantics over a VK's assigned tool patterns. MISSING→**HAVE**. |
| PAR-BF-MCP-018 | Auto-execute list `ToolsToAutoExecute` ⊆ `ToolsToExecute` | **BUILD (storage + validation; no agent loop)** | The subset constraint is VALIDATED in the live assignment path (D5); the value is STORED. g0router has no server-mode agent loop, so "auto-execute" is config-only here (the agent loop is the CLIENT-mode `ExecuteTool` path, not server-mode). MISSING→**PARTIAL** (subset validation + storage HAVE; server-mode auto-exec agent loop ESC). |
| PAR-BF-MCP-019 | Per-VK `executeOnlyTools` with wildcard `clientName-*` and `clientName-toolName` | **BUILD** | The core deliverable: the wildcard filter (D4) matches `*` (all), `<client>-*` (client-prefix wildcard), and exact `<client>-<tool>`; REUSES `buildToolPolicy`/`stripServerPrefix`. MISSING→**HAVE**. |
| PAR-BF-MCP-020 | `AllowOnAllVirtualKeys` flag on client config | **BUILD** | A per-MCP-client flag that BYPASSES the per-VK filter (tools from an all-VK client are always visible) (D6). MISSING→**HAVE**. |
| PAR-BF-MCP-033 | VK↔MCP assignment table (`virtual_key_mcp_configs`) | **BUILD (additive table)** | NEW additive `virtual_key_mcp_configs` table + store (D2), the SOURCE the filter reads. MISSING→**HAVE**. |
| PAR-BF-MCP-049 | Validation: `ToolsToAutoExecute` must be subset of `ToolsToExecute` | **BUILD** | Inline subset validation rejecting an invalid assignment in the live admin path (D5). MISSING→**HAVE**. |
| PAR-BF-MCP-057 | `DisableAutoToolInject` flag | **BUILD (config flag + live read)** | Stored on the per-VK assignment OR per-client config; READ to suppress the auto-injected tool surface for that VK (D7). MISSING→**HAVE** if a live read narrows behavior; **PARTIAL** if no consumer exists (then it is config-only, recorded). |
| PAR-BF-MCP-071 | `AllowedExtraHeaders` whitelist for request-level header forwarding | **BUILD (config-only — NO forwarding path)** | g0router's `/mcp` server-mode does NOT forward request headers upstream (no per-request upstream call in server mode). The whitelist is STORED + validated (canonicalized, no empties) but has NO live forwarding consumer. MISSING→**PARTIAL** (stored + validated; forwarding ESC — no server-mode upstream header path). |
| PAR-BF-MCP-077 | Tool annotations mapping (Title, ReadOnlyHint, DestructiveHint, IdempotentHint, OpenWorldHint) | **BUILD** | Map annotation fields onto the `ServerTool`/catalog shape so `tools/list` carries annotations (D8). MISSING→**HAVE** if annotations flow into `tools/list`; **PARTIAL** if the probe source supplies none (stored shape present, no data). |
| PAR-BF-MCP-078 | `IsCodeModeClient` flag on client config | **BUILD (config-only — execution ESC)** | Flag STORED on client config; NO code-mode execution engine (VFS) built. MISSING→**PARTIAL** (flag stored; code-mode VFS execution ESC, §3). |
| PAR-BF-MCP-079 | `ConfigHash` field for reconciliation | **BUILD (computed on write + EXPOSED for drift-detection; reconciliation worker ESC)** | A deterministic hash over the assignment/config computed on write AND **exposed in the assignment GET DTO** so an operator/client can read it to detect config drift (D8 — the live reader). The auto-reconciliation worker that ACTS on drift is ESC. MISSING→**PARTIAL** (hash computed + observable for drift-detection; reconciliation worker ESC). |

**Honest scoping note:** 004 is **HAVE-by-variant** (request-time filter, not a
long-lived per-VK server). 017/019/020/033/049 close fully **HAVE** (the wildcard
filter, the bypass flag, the assignment table, the subset validation — all with live
consumers + behavior proofs in §5). 018/057/071/077/078/079 close **PARTIAL** where
the matrix field's *live behavioral* consumer is absent in g0router's server-mode
shape (auto-exec agent loop, header forwarding, code-mode VFS, reconciliation worker)
— in each case the field is STORED + validated + **observable through the live admin
API** (071/078 are operator-set config that round-trips through the assignment/client
admin API; 079's `config_hash` is computed on write AND exposed in the GET DTO for
drift-detection — D8), with the missing *execution/enforcement* recorded ESC (§3) and
the residual in `open-questions.md` (§7). **No field is added write-only:** every
stored field is either READ to change behavior, or SET by an operator through the
admin API, or EXPOSED in a GET DTO for observability — a field that is computed/stored
and neither read, operator-set, nor exposed is dead → STOP+escalate (§3 no-leftovers). No
row is closed by inventing Bifrost's exact column set or wildcard edge — see the
STOP-condition in D4.

### 0.3 Preconditions already satisfied (evidence — read at HEAD 37a1c5d)

- **The SHARED catalog assembler EXISTS — the filter's INPUT (REUSE, do NOT rebuild).**
  `internal/admin/mcp.go:599 mcpToolCatalog() []catalogEntry` aggregates the global
  surface ONCE (probe-discovered tools, default-plugin baseline, antigravity
  ride-along); `:660 assembleServerCatalog() []mcp.ServerTool` maps it to the
  server-mode `tools/list` shape; `:673 serverCatalogSource{h}` adapts it to the
  `mcp.CatalogSource` the dispatcher consumes. bf-mcp-2's per-VK filter wraps THIS
  output — it does NOT add a second catalog source (the bf-mcp-1 D3 one-source
  invariant holds).
- **The VK resolve+validate seam EXISTS — the filter's KEY (EXTEND additively).**
  `internal/admin/mcp.go:706 admitMCPVK(ctx) (vk string, admitted bool)` already
  resolves (`:51 resolveMCPVK`) and validates the VK (`:711 GetVirtualKeyByKey` +
  `IsActive`). bf-mcp-2 consumes the returned `vk` to LOOK UP the assignment scope —
  bf-mcp-1 explicitly left the resolved VK non-scoping (`mcp.go:727` "global
  un-scoped tool surface").
- **The per-request server construction seam EXISTS — the injection point (EXTEND).**
  `internal/admin/mcp.go:685 newMCPServer() *mcp.Server` builds the dispatcher with
  `serverCatalogSource{h}` + the running bridge dispatcher each request. This IS the
  "lazy creation" seam (D1): bf-mcp-2 makes it construct a VK-SCOPED `CatalogSource`
  for the resolved VK at request time. No long-lived per-VK object is needed.
- **The wildcard dual-key primitive EXISTS (REUSE — D4).**
  `internal/mcp/toolpolicy.go:48 buildToolPolicy(server, toolNames)` already emits
  BOTH the bare name AND `<server>-<tool>` as keys; `:35 stripServerPrefix` is the
  idempotent prefix strip. The `executeOnlyTools` wildcard (`clientName-*` /
  `clientName-toolName`) is the SAME bare-vs-prefixed dual shape — the filter REUSES
  these rather than re-deriving prefix logic.
- **The tool-group additive-table pattern EXISTS — the assignment-table template.**
  `internal/store/mcptoolgroups.go` (INTEGER-PK `mcp_tool_groups`, JSON tool_ids,
  `is_active`, ISO timestamps) + `migrate.go:265` `CREATE TABLE IF NOT EXISTS
  mcp_tool_groups`. The VK↔MCP assignment store + table mirror this exact pattern.
- **The additive-migration mechanism EXISTS.** `internal/store/migrate.go:12`
  "tables are created if missing and columns are appended via ensureColumn";
  `:478 ensureColumn`; the additive `ensureColumn` loop (`:399-403`, incl. the
  bf-gov-1 `{"virtual_keys","team_id",…}` precedent). New tables append to the
  `CREATE TABLE IF NOT EXISTS` list (`:223-272`).
- **The additive-blob precedent EXISTS — the flag-storage decision point (D6/D7).**
  `internal/store/virtualkeys.go:24 virtualKeyConfig` snake_case JSON blob in
  `config_json` (the bf-gov-1 / governance additive-blob home). Per-VK flags that are
  truly per-VK can ride here; the per-MCP-client `AllowOnAllVirtualKeys` rides on the
  `mcp_clients.config_json` blob (`migrate.go:227`) instead (D6).
- **The masked, no-token DTO discipline EXISTS.** `internal/admin/mcp.go:109
  accountDTO` carries no token fields. The assignment DTO follows the SAME no-leak
  discipline (it carries only tool patterns + flags, never secrets).
- **The audit seam EXISTS (REUSE).** `internal/admin/audit.go recordAudit` (used
  throughout `admin/mcp.go`, e.g. `:743 "mcp_server.tools_call"`,
  `:995 "mcp_tool_group.create"`). The assignment write path audits the same way.
- **The route registration seam EXISTS (SERIAL).** `internal/server/routes_mcp.go`
  (`RegisterMCPRoutes`) is already CALLED from `RegisterAdminRoutes` (bf-mcp-1 D2).
  Any new assignment CRUD route registers there (additive) — the serial-held surface.

---

## 1. Decisions made (and why) — binding

### D1 — "Lazy creation" for g0router = request-time scope computation (VAR vs per-VK server object)

PAR-BF-MCP-004 wants a "per-virtual-key MCP server with lazy creation"
(Bifrost: a `vkMCPServers` map + `ensureVKMCPServer` first-request build,
matrix quirk #11 "created lazily to avoid O(100k) startup stall").

**Decision (binding):** g0router has ONE global `mcp.Server` constructed PER REQUEST
by `newMCPServer()` (`admin/mcp.go:685`). g0router does NOT have — and does NOT add —
a long-lived per-VK server-instance architecture. The matrix's "lazy creation"
PURPOSE (avoid building N per-VK servers at startup) is satisfied trivially by
g0router's design: the per-VK scope is computed **on-demand at request time** from the
assignment table when a VK is resolved on `/mcp`, then thrown away. There is no map to
populate, no startup stall to avoid, no per-VK object lifecycle. **PAR-BF-MCP-004 is
HAVE-by-variant** (the BEHAVIOR — a VK sees only its scoped tools, computed lazily on
first/each request — is built; the per-VK long-lived `vkMCPServers` map ARCHITECTURE
is a deliberate VAR, recorded in `open-questions.md`). If, at impl, the matrix's
per-VK behavior turns out to REQUIRE per-VK server state that the request-time variant
cannot express (e.g. per-VK session affinity across requests), **STOP and escalate** —
do NOT build a per-VK server-object map on speculation.

### D2 — VK↔MCP scope storage = additive `virtual_key_mcp_configs` table (NOT a VK config-blob field)

PAR-BF-MCP-033 names a `virtual_key_mcp_configs` junction table
(`virtual_key_id` + `mcp_client_id` + `tools_to_execute_json`).

**Decision (binding):** the VK→MCP scope is a **many-to-many** relation (one VK scopes
N MCP clients; one client is scoped by N VKs), so it CANNOT ride the single-row
`virtualKeyConfig` blob the way bf-gov-1's scalar `team_id` did. It gets a NEW additive
table `virtual_key_mcp_configs` + a NEW store `internal/store/mcpvkconfigs.go`,
mirroring the `mcp_tool_groups` additive-table pattern (`mcptoolgroups.go` /
`migrate.go:265`):
```
virtual_key_mcp_configs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  virtual_key_id TEXT NOT NULL DEFAULT '',
  mcp_client_id  TEXT NOT NULL DEFAULT '',
  tools_to_execute_json      TEXT NOT NULL DEFAULT '[]',   -- executeOnlyTools patterns (017/019)
  tools_to_auto_execute_json TEXT NOT NULL DEFAULT '[]',   -- subset of the above (018/049)
  config_hash TEXT NOT NULL DEFAULT '',                     -- 079 (computed on write; EXPOSED in GET DTO for drift-detection; reconciliation worker ESC)
  created_at INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL DEFAULT 0
)
```
Appended to the `CREATE TABLE IF NOT EXISTS` list (`migrate.go:223-272`) — additive,
never destructive. The store exposes `CreateVKMCPConfig`/`ListVKMCPConfigsByVK`/
`UpdateVKMCPConfig`/`DeleteVKMCPConfig` mirroring `mcptoolgroups.go`. **The filter
READS `ListVKMCPConfigsByVK(vk)` at request time** (D1/D3) — this is the table's live
consumer; a table nothing reads is a dead table → STOP+escalate. The per-MCP-client
`AllowOnAllVirtualKeys` flag is NOT on this table — it is per-client, so it rides the
`mcp_clients.config_json` blob (D6). The exact Bifrost column names/types beyond the
matrix-named trio are **VAR** (g0router-shaped); a column the matrix does not evidence
is NOT added (no-leftovers).

### D3 — Where the scope filter plugs in: a VK-scoped `CatalogSource` via `newMCPServer`

**Decision:** `newMCPServer()` (`admin/mcp.go:685`) gains a VK parameter (or a
sibling `newScopedMCPServer(vk string)`); when a non-empty VK is resolved+admitted
(`admitMCPVK`), it constructs the dispatcher with a **VK-scoped `CatalogSource`**
instead of the bare `serverCatalogSource{h}`. The scoped source:
1. reads the global catalog ONCE via the SHIPPED `assembleServerCatalog()` (one
   source — bf-mcp-1 D3 invariant preserved);
2. reads the VK's assignment rows via `ListVKMCPConfigsByVK(vk)` (D2);
3. narrows the global `[]ServerTool` to the names the VK's `executeOnlyTools`
   patterns admit (D4), UNION the tools whose owning client is `AllowOnAllVirtualKeys`
   (D6).
An **absent VK** (anonymous, allowed per bf-mcp-1 D4) keeps the bare global source —
no behavior change for the un-scoped path. The dispatcher's `tools/call` ALSO gates on
the scoped set: a scoped VK calling a tool OUTSIDE its scope gets a JSON-RPC error
(the filter is enforced on BOTH `tools/list` and `tools/call`, else a VK could call a
tool it cannot see — no-leftovers). The `tools/call` dispatch itself still DELEGATES to
the shipped `NewBridgeDispatcher` (no re-implementation).

**No-leftovers (binding):** a test proves a VK with a restricted assignment sees
**strictly FEWER** tools on `tools/list` than the global catalog, AND cannot
`tools/call` an out-of-scope tool. If the scoped source ever returns the full catalog
for a restricted VK (filter never narrows), STOP+escalate per §3.

### D4 — `executeOnlyTools` wildcard semantics (REUSE the dual-key primitive)

PAR-BF-MCP-017/019: `["*"]`=all, `nil`/`[]`=deny; wildcard `clientName-*` and exact
`clientName-toolName`.

**Decision (binding):** a NEW pure function (in `internal/mcp/`, e.g.
`scopeTools(global []ServerTool, patterns []string, clientOf func(tool string) string)
[]ServerTool`) implements:
- `patterns == nil || len(patterns) == 0` → **deny-all** (empty result) (017);
- a pattern `"*"` → **allow-all** (the global slice, unfiltered) (017);
- a pattern `"<client>-*"` → allow every tool whose owning client == `<client>` (019
  prefix wildcard);
- an exact pattern `"<client>-<tool>"` or bare `"<tool>"` → allow that tool, matched
  via the SHIPPED dual-key shape (`buildToolPolicy` emits bare + `<server>-<tool>`;
  `stripServerPrefix` normalizes) (019).
The function is PURE + table-driven over every case (`*`, prefix-wildcard, exact,
bare, deny-empty, unknown-pattern→no match). It REUSES `toolpolicy.go`'s
`stripServerPrefix`/`buildToolPolicy` for the prefix/dual-key matching rather than
re-deriving it. **STOP-condition (ESC-REF-ABSENT):** the matrix names exactly these
three forms (`*`, `<client>-*`, `<client>-<tool>`). If a reviewer requires a Bifrost
wildcard edge NOT in the matrix note (e.g. glob `*-read`, regex, negation), the
affected behavior **STOPS and escalates** — do NOT invent Bifrost's wildcard grammar
beyond the three documented forms.

### D5 — `ToolsToAutoExecute` ⊆ `ToolsToExecute` subset validation (live admin path)

PAR-BF-MCP-018/049: auto-execute list must be a subset of the execute list.

**Decision:** a PURE `validateAutoExecuteSubset(execute, autoExecute []string) error`
returns a non-nil error when any `autoExecute` entry is not present in `execute`
(matched as literal patterns — both are the same pattern vocabulary as D4; a `"*"`
execute admits any auto-execute). It is CALLED in the LIVE assignment write path (the
create/update VK↔MCP-config admin handler, D-routes) so an invalid assignment is
**rejected with a 4xx `{error}`** before it is stored. **No-leftovers (binding):** a
test posts an assignment with `autoExecute` ⊄ `execute` and asserts the live path
REJECTS it (a `{error}` envelope, not a stored row). `ToolsToAutoExecute` is STORED
(D2) but there is NO server-mode agent loop that auto-executes (that is the CLIENT-mode
`ExecuteTool` path) — so 018 is **PARTIAL** (subset validation + storage HAVE; the
server-mode auto-exec agent loop is ESC, §3) and 049 is **HAVE** (the validation).

### D6 — `AllowOnAllVirtualKeys` = per-MCP-client bypass flag (on `mcp_clients` config blob)

PAR-BF-MCP-020: `AllowOnAllVirtualKeys` flag on the CLIENT config.

**Decision (binding):** this is a per-**client** flag (not per-VK), so it rides the
`mcp_clients.config_json` blob (`migrate.go:227`), NOT the `virtual_key_mcp_configs`
table (D2). When a client has `allow_on_all_virtual_keys: true`, ALL of its tools are
visible to EVERY VK regardless of that VK's `executeOnlyTools` scope — the scoped
`CatalogSource` (D3) UNIONs the all-VK clients' tools into every VK's result, bypassing
the per-VK filter. **No-leftovers (binding):** a test proves that a VK whose
`executeOnlyTools` does NOT name client X still sees client X's tools WHEN client X is
`AllowOnAllVirtualKeys`, and does NOT see them when the flag is false. A flag that
never bypasses is a dead flag → STOP+escalate.

### D7 — `DisableAutoToolInject` (057) — live read or config-only, decided at impl

PAR-BF-MCP-057: a flag suppressing auto-injected tools.

**Decision:** STORE the flag (per-VK on the assignment, or per-client on the client
blob — decided at impl by where the matrix evidence points; default per-client
mirroring `AllowOnAllVirtualKeys`). g0router's server-mode `tools/list` IS the
"injected" surface a client consumes; if `DisableAutoToolInject` is set, the scoped
source omits that client's tools from the un-requested baseline. **No-leftovers:** if a
live read narrowing the served surface exists, 057 → **HAVE** with a behavior proof; if
g0router's server-mode has no "auto-inject vs explicit-request" distinction to suppress
(plausible — g0router always serves the catalog on `tools/list`), the flag is **stored
+ marked config-only** and 057 → **PARTIAL**, recorded in `open-questions.md`. **A flag
neither read NOR explicitly marked config-only is a dead flag → STOP+escalate** —
decide one of the two at impl, never leave it dangling.

### D8 — `AllowedExtraHeaders` (071), annotations (077), `IsCodeModeClient` (078), `ConfigHash` (079)

**`AllowedExtraHeaders` (071) — config-only.** g0router's `/mcp` server mode does NOT
make a per-request upstream call carrying forwarded request headers (the server
re-exposes an already-discovered catalog; `tools/call` delegates to a LOCAL bridge).
There is no header-forwarding path to gate. The whitelist is STORED + validated
(canonicalized lowercase/trim, no empties/dupes — REUSE the canonicalization shape
where present, else a small pure helper) but has NO live forwarding consumer →
**PARTIAL** (stored + validated; forwarding ESC, §3). Marked config-only explicitly.

**Tool annotations (077) — mapped into `tools/list`.** Add the annotation fields
(`Title`, `ReadOnlyHint`, `DestructiveHint`, `IdempotentHint`, `OpenWorldHint`) to the
`ServerTool`/catalog shape and carry them through `assembleServerCatalog` →
`tools/list`. **No-leftovers:** if the probe/catalog source supplies annotation data,
077 → **HAVE** (a test asserts an annotated tool surfaces its annotations in
`tools/list`); if g0router's probe yields no annotation data today, the SHAPE is added
(stored/serialized) but carries no values → **PARTIAL**, recorded. The annotation shape
is additive on `ServerTool` (`omitempty`), preserving every existing serialization.

**`IsCodeModeClient` (078) — config-only; execution ESC.** STORE the flag on
`mcp_clients.config_json`. **Code-mode VFS execution is NOT built** (§3 ESC) — there is
no VFS, no nested-tool plugin pipeline. 078 → **PARTIAL** (flag stored; execution ESC),
recorded. **A code-mode execution engine is NEVER faked.**

**`ConfigHash` (079) — computed on write + EXPOSED in the GET DTO for drift-detection;
reconciliation ESC.** Compute a deterministic hash (e.g. SHA-256 over the
canonicalized assignment/config JSON) and STORE it on write
(`virtual_key_mcp_configs.config_hash`, D2). **The live reader is the assignment GET
DTO: `config_hash` is RETURNED to the operator/client on read so they can detect
config drift** (the hash changes iff the assignment changed) — a legitimate, cheap
observability consumer. The auto-reconciliation WORKER that would diff hashes to
trigger re-sync is ESC (§3). 079 → **PARTIAL** (hash computed on write + exposed in the
GET DTO for drift-detection/observability; the worker that ACTS on drift is ESC),
recorded. **A write-only hash that nothing reads is a dead column → STOP+escalate:**
the GET-DTO exposure is the live reader (a test asserts the GET DTO carries
`config_hash`); a self-computed value's WRITE is NOT its own consumer.

### D-routes — assignment admin surface (additive; serial slot)

The VK↔MCP assignment must be settable. **Decision:** add additive admin CRUD on the
existing `/api/mcp/` (session-gated, local-only) surface — e.g.
`POST/GET/PUT/DELETE /api/mcp/vk-configs` (or instance/VK-scoped paths; exact shape
decided at impl to match the existing `/api/mcp/*` conventions). Registered via the
SHIPPED `RegisterMCPRoutes` (`routes_mcp.go`, already called from
`RegisterAdminRoutes`) — additive lines only; this is the **serial-held edit**. The
create/update handlers run the D5 subset validation + the D8 `ConfigHash` compute. The
DTO follows the no-leak discipline (tool patterns + flags only). `{data,error}`
snake_case envelope on these admin routes; `/mcp` stays raw JSON-RPC (unchanged).

---

## 2. Target files

### IN-SCOPE — CREATE (NEW)

| File | Contract |
|---|---|
| `internal/store/mcpvkconfigs.go` | The VK↔MCP assignment store: `VKMCPConfig` struct + `CreateVKMCPConfig`/`ListVKMCPConfigsByVK`/`UpdateVKMCPConfig`/`DeleteVKMCPConfig` over the additive `virtual_key_mcp_configs` table (D2). Mirrors `mcptoolgroups.go` exactly (INTEGER PK, JSON pattern arrays, ISO/int timestamps, `ErrNotFound`, scan helper). No `init()`; errors-as-values; no global state. |
| `internal/store/mcpvkconfigs_test.go` | RED first: create→get→list-by-VK→update→delete round-trips; many-to-many (one VK → N clients; one client → N VKs); empty/`["*"]` pattern arrays round-trip; `ErrNotFound` on missing. Hermetic (in-memory SQLite test store). |
| `internal/mcp/scope.go` | The PURE wildcard scope filter (D4): `scopeTools(global []ServerTool, patterns []string, clientOf func(string) string) []ServerTool` + `validateAutoExecuteSubset(execute, autoExecute []string) error` (D5). REUSES `stripServerPrefix`/`buildToolPolicy` (toolpolicy.go). PURE — no I/O. No `init()`. |
| `internal/mcp/scope_test.go` | RED first: deny-empty; `*`=all; `<client>-*` prefix wildcard; exact `<client>-<tool>`; bare `<tool>`; unknown pattern→no match; subset-valid vs subset-invalid (049). Table-driven, hermetic. |

### IN-SCOPE — EXTEND (additive only)

| File | Change (additive ONLY) |
|---|---|
| `internal/admin/mcp.go` | ADD the VK-scoped `CatalogSource` adapter + make `newMCPServer` accept a VK (or a sibling `newScopedMCPServer(vk)`) so `MCPServerPost`/`MCPServerSSE` pass the resolved `vk` from `admitMCPVK` (D3). ADD the assignment CRUD handlers (D-routes) running D5 subset validation + D8 `ConfigHash` compute. ADD the `AllowOnAllVirtualKeys` / `DisableAutoToolInject` / `IsCodeModeClient` client-flag reads from `mcp_clients.config_json` (D6/D7/D8). ENFORCE the scope on `tools/call` too (D3). PRESERVE every existing handler body/signature; the un-scoped (absent-VK) path is unchanged. |
| `internal/admin/mcp_test.go` | RED first: a restricted VK sees FEWER tools on `tools/list` than the global catalog (D3 live-narrowing); a scoped VK cannot `tools/call` an out-of-scope tool (D3); `AllowOnAllVirtualKeys` bypass proof (D6); subset-validation reject in the live assignment path (D5/049); `config_hash` computed on write AND **returned in the assignment GET DTO** for drift-detection (D8/079 live reader); absent-VK path unchanged (regression). Hermetic. |
| `internal/mcp/server.go` | ADD annotation fields to `ServerTool` (`omitempty`) for 077 (D8). The dispatcher is otherwise unchanged (it consumes whatever `CatalogSource` it is given — the scoping happens in the source, not the dispatcher). PRESERVE the existing `Dispatch`/`handleToolsList`/`handleToolsCall`. |
| `internal/mcp/server_test.go` (EXTEND) | RED first: `tools/list` carries annotations when the catalog supplies them (077); the dispatcher narrows when given a scoped (canned) catalog source. |
| `internal/store/migrate.go` | ADD the `virtual_key_mcp_configs` `CREATE TABLE IF NOT EXISTS` to the additive table list (`:223-272`) (D2). NOTHING destructive. |
| `internal/store/virtualkeys.go` OR `internal/store/mcpinstances.go` (client config) | ADD the additive `allow_on_all_virtual_keys` / `disable_auto_tool_inject` / `is_code_mode_client` fields to the `mcp_clients` config blob marshal/unmarshal (D6/D7/D8) — additive JSON keys, never a struct-shape break. DECIDE at impl which file owns the `mcp_clients` config blob. |
| `internal/server/routes_mcp.go` | ADD the additive assignment-CRUD route lines via `RegisterMCPRoutes` (D-routes). SERIAL-held edit. RELEASE the routes_admin.go slot to bf-core-2 on close. |
| `internal/server/routes_mcp_test.go` (EXTEND) | RED first: assignment routes registered + session-gated; `/mcp` JSON-RPC unchanged (raw, not envelope). Hermetic. |
| `.planning/parity/matrix/bifrost-mcp.md` | Flip rows per §7 (at close). |
| `.planning/parity/plans/open-questions.md` | Append ESC + config-only + VAR items (§7). |
| `docs/WORKFLOW.md` | Add the bf-mcp-2 row (at close). |

### FORBIDDEN (automatic REJECT if touched)

- `internal/mcp/{bridge,sse,probe,agent,launcher,oauth,filter,allowlist,defaults,
  registry,healthmonitor,discovery,toolpolicy,process,runner}.go` — CONSUME the
  shipped CLIENT-mode primitives; do NOT edit them (BIFROST-MAP §397 "Wave-7 MCP
  client-mode is consume-only"). `toolpolicy.go`/`filter.go`/`allowlist.go` are REUSED
  by IMPORT, never edited.
- The bf-mcp-1 `/mcp` JSON-RPC contract — bf-mcp-2 CHANGES what a scoped VK SEES, it
  does NOT change the wire contract, add a `/mcp` route, or alter `resolveMCPVK`/
  `admitMCPVK`'s validation semantics (it only consumes the resolved VK).
- `internal/server/routes_openai.go` — not a routes_openai holder.
- `internal/server/guard.go` — `/mcp` local-only posture is bf-mcp-1's; unchanged.
- The `mark3labs/mcp-go` dependency — NOT added (bf-mcp-1 D1 VAR holds).
- **A code-mode VFS execution engine / nested-tool plugin pipeline** — ESC (§3); the
  `IsCodeModeClient` flag is STORED, the execution is NEVER built.
- Any per-user OAuth/header/sessions surface — ESC (§3).
- Any `init()`, any new free global state, any `New(...)`/`Register*(...)` signature
  change beyond the additive `newMCPServer` VK param (use additive setters/params),
  any GORM hook, any destructive DDL (`DROP`/`RENAME`).
- Any UI file (`ui/**`) — bf-mcp-2 is Go-only (BIFROST-MAP §378).

---

## 3. Scope / Non-goals — explicit ESC list

**bf-mcp-2 builds ONLY per-VK scope filtering + the assignment table + the
config-flag storage with their live reads.** The following are **ESC** (recorded in
`open-questions.md` at close):

| ESC item | Matrix row(s) | Why ESC |
|---|---|---|
| **Code-mode VFS execution engine** (server-vs-tool VFS binding; nested-tool execution) | 056, 078 (execution) | Presupposes a VFS + a nested-tool plugin pipeline g0router lacks (BIFROST-MAP §270). The `IsCodeModeClient` FLAG is stored (D8); the execution engine is NOT built and is NEVER faked. |
| **Plugin pipeline for nested tool calls in code mode** | 060 | Bifrost's `pluginPipelineProvider`/`releasePluginPipeline`. g0router has no plugin pipeline integration; deferred. |
| **Per-user OAuth / per-user header credential flows / sessions API** | 012-016, 027-031, 040-047, 058-059, 064-065 | The enterprise multi-tenant core (BIFROST-MAP §269); no per-user identity model. The `AllowedExtraHeaders` whitelist (071) is stored + validated but its per-request FORWARDING path is part of this surface → forwarding ESC. |
| **Server-mode auto-execute agent loop** | 018 (agent loop) | g0router's agent loop is the CLIENT-mode `ExecuteTool` path; server-mode `/mcp` has no auto-exec loop. `ToolsToAutoExecute` is STORED + subset-validated (D5); the server-mode auto-exec loop is ESC. |
| **`ConfigHash` reconciliation worker** | 079 (reconciliation) | The worker that diffs hashes to trigger re-sync (and the VK credential reconciliation hooks, 065) presuppose Bifrost's reconciliation architecture. The hash is COMPUTED on write + EXPOSED in the GET DTO for drift-detection (D8 — its live reader); only the worker that ACTS on drift is ESC. |
| **Flexible-duration / canonicalization quirks, tool-pricing catalog, two-phase create/update rollback, discovered-tool key migration, retry-on-in-flight-reconnect, header redaction-merge, OAuth-rotation-disabled guard, sentinel error types** | 022, 023, 050, 051, 055, 061, 062, 063, 068, 069, 070, 072, 073, 074 | Implementation-quirk + handler-architecture-coupled rows (BIFROST-MAP §270). Out of bf-mcp-2's per-VK-scoping scope. |
| **Per-VK long-lived server-instance map architecture** | 004 (architecture) | g0router uses the request-time-filter VARIANT (D1). The per-VK `vkMCPServers` map + lifecycle is VAR, recorded — NOT built on speculation. |

No-leftovers (binding, §3 CLI_ORCHESTRATOR): every NEW surface bf-mcp-2 adds — the
`virtual_key_mcp_configs` table (live read = the scoped `CatalogSource`, D2/D3), the
`scopeTools` filter (live consumer = the scoped source narrows `tools/list`+`tools/call`,
D3/D4), the subset validator (live consumer = the assignment write path rejects an
invalid subset, D5), the `AllowOnAllVirtualKeys` flag (live consumer = the bypass
union, D6), and every other flag (`DisableAutoToolInject`, `IsCodeModeClient`,
`ConfigHash`, `AllowedExtraHeaders`) — MUST have a live consumer with a grep AND a
behavior proof (§5), **OR be explicitly marked config-only** (stored + validated, with
its missing execution recorded ESC). **No dead table, no dead column, no flag that is
neither read nor marked config-only, no filter that never narrows, no bypass that never
bypasses.** Code-mode execution is ESC, never faked.

---

## 4. Task graph (TDD; `N. [step] -> verify: [check]`)

Cadence (AGENTS.md "TDD always"): no impl file/field lands before its covering
`_test.go` is committed RED. `go test ./... && go vet ./... && go build ./...` green,
**HERMETIC** (no real network, no subprocess, no real SSE timing) at EVERY commit.
Serial: bf-mcp-2 holds routes_admin.go (the additive assignment-route registration).

1. **[wildcard scope filter + subset validator, RED→GREEN]** Write
   `internal/mcp/scope_test.go` (deny-empty; `*`=all; `<client>-*`; exact
   `<client>-<tool>`; bare; unknown→no match; subset-valid/invalid). Implement
   `internal/mcp/scope.go` REUSING `stripServerPrefix`/`buildToolPolicy` (D4/D5). ->
   verify: `go test ./internal/mcp/ -run 'Scope|Subset'` green; gates exit 0; grep
   proof: `scope.go` imports/uses the shipped prefix primitives (no re-derived prefix
   logic). Commit RED then GREEN:
   `phase-1/bf-mcp-2: failing executeOnlyTools wildcard filter + subset validator (TDD red)` →
   `phase-1/bf-mcp-2: per-VK executeOnlyTools wildcard scope filter (reuse toolpolicy)`.

2. **[VK↔MCP assignment table + store, RED→GREEN]** Write
   `internal/store/mcpvkconfigs_test.go` (CRUD round-trips; many-to-many;
   `["*"]`/empty arrays; `ErrNotFound`). Add the additive
   `virtual_key_mcp_configs` `CREATE TABLE` to `migrate.go`; implement
   `internal/store/mcpvkconfigs.go` mirroring `mcptoolgroups.go` (D2). -> verify:
   `go test ./internal/store/ -run VKMCP` green; gates exit 0; grep proof: additive
   table only, no `DROP`/`RENAME` in the migrate diff. Commit RED then GREEN:
   `phase-1/bf-mcp-2: failing virtual_key_mcp_configs store (TDD red)` →
   `phase-1/bf-mcp-2: additive VK↔MCP assignment table + store`.

3. **[client config flags, RED→GREEN]** Write the `mcp_clients` config-blob
   marshal/unmarshal test for the additive `allow_on_all_virtual_keys` /
   `disable_auto_tool_inject` / `is_code_mode_client` keys (round-trip; default
   false; old blob without the keys still unmarshals). Implement the additive blob
   fields (D6/D7/D8). -> verify: `go test ./internal/store/ -run MCPClientConfig`
   green; gates exit 0; grep proof: additive JSON keys, no struct-shape break.
   Commit RED then GREEN:
   `phase-1/bf-mcp-2: additive mcp_clients flags (allow-on-all-vk, disable-auto-inject, code-mode)`.

4. **[scoped CatalogSource: live narrowing + bypass + tools/call gate, RED→GREEN]**
   Extend `internal/admin/mcp_test.go`: **a restricted VK sees FEWER tools on
   `tools/list` than the global catalog (D3 live-narrowing); a scoped VK cannot
   `tools/call` an out-of-scope tool; `AllowOnAllVirtualKeys` makes a non-assigned
   client's tools visible to that VK (D6 bypass); an absent VK still sees the full
   catalog (regression).** Implement the VK-scoped `CatalogSource` + the
   `newMCPServer(vk)` injection + the `tools/call` scope gate + the `AllowOnAllVK`
   union (D3/D6) reading `ListVKMCPConfigsByVK` + the client flags. -> verify:
   `go test ./internal/admin/ -run 'MCPScope|MCPServer'` green; the live-narrowing,
   bypass, and tools/call-gate behaviors assert in tests; gates exit 0. Commit RED
   then GREEN:
   `phase-1/bf-mcp-2: failing per-VK scoped tools/list + tools/call gate (TDD red)` →
   `phase-1/bf-mcp-2: per-VK scoped MCP catalog (request-time lazy filter) + AllowOnAllVK bypass`.

5. **[assignment CRUD admin + subset-validation reject + ConfigHash, RED→GREEN —
   SERIAL SLOT]** Extend `mcp_test.go` + `routes_mcp_test.go`: the assignment
   create/update path REJECTS an `autoExecute ⊄ execute` assignment with a 4xx
   `{error}` (D5/049); a valid assignment is stored with a computed `config_hash` that
   is **then returned in the assignment GET DTO** (D8/079 live reader — the GET
   response carries `config_hash` for drift-detection); the assignment routes are
   registered + session-gated; `/mcp` JSON-RPC stays raw (regression). Implement the
   assignment CRUD handlers (calling `validateAutoExecuteSubset` + the `ConfigHash`
   compute, exposing `config_hash` in the GET DTO) + the additive `RegisterMCPRoutes`
   lines. -> verify: `go test ./internal/admin/ ./internal/server/` green; `go test
   ./... && go vet ./... && go build ./...` exit 0; grep proof: subset validation
   called in the live path, `config_hash` exposed in the GET DTO (read, not write-only).
   Commit RED then GREEN:
   `phase-1/bf-mcp-2: failing assignment CRUD + subset reject + config-hash (TDD red)` →
   `phase-1/bf-mcp-2: VK↔MCP assignment CRUD (subset validation; config-hash; serial)`.

6. **[tool annotations mapping, RED→GREEN]** Extend `internal/mcp/server_test.go`
   (+ admin catalog test): an annotated tool surfaces its annotations in `tools/list`
   (077). Add the `omitempty` annotation fields to `ServerTool` + carry them through
   `assembleServerCatalog`. -> verify: `go test ./internal/mcp/ ./internal/admin/`
   green; gates exit 0. If the probe source yields no annotation data, the shape is
   added (serialized) and 077 → PARTIAL (§7) — do NOT ship a field nothing can ever
   populate without recording it. Commit RED then GREEN:
   `phase-1/bf-mcp-2: tool annotations mapping on tools/list`.

7. **[close]** Full validation (§6); flip matrix rows (§7); append
   `open-questions.md` (ESC §3 + config-only flags + the D1 per-VK-server-architecture
   VAR + the 018/057/071/077/078/079 PARTIAL residuals); update `docs/WORKFLOW.md`;
   RELEASE the routes_admin.go serial slot to bf-core-2. -> verify: §6 all green;
   matrix + WORKFLOW + open-questions committed. Commit:
   `phase-1/bf-mcp-2: close — per-VK MCP scoping; matrix flip; serial release`.

---

## 5. Acceptance criteria (binary; file:line / grep proofs)

**Test gates** (each yes/no, exit 0; HERMETIC — no network, no subprocess, no real
SSE timing):
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/mcp/ -run 'Scope|Subset|Server' -v` → filter/subset/annotation
  cases pass.
- `go test ./internal/store/ -run 'VKMCP|MCPClientConfig' -v` → store + flag cases pass.
- `go test ./internal/admin/ -run 'MCPScope|MCPServer' -v` → live-narrowing, bypass,
  tools/call-gate, subset-reject, config-hash pass.
- `go test ./internal/server/ -run MCP -v` → assignment routes + `/mcp`-raw-JSON-RPC
  regression pass.

**TDD-order proof** — each impl's covering test is in an earlier-or-equal commit:
```bash
for pair in \
  "internal/mcp/scope_test.go:internal/mcp/scope.go" \
  "internal/store/mcpvkconfigs_test.go:internal/store/mcpvkconfigs.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"   # prints nothing
done
```

**Grep / behavior proofs (no-leftovers):**
```bash
# the wildcard filter REUSES the shipped prefix primitive (no re-derived prefix logic)
grep -nE 'stripServerPrefix|buildToolPolicy|StripServerPrefix' internal/mcp/scope.go
! grep -nE 'func stripServerPrefix|func buildToolPolicy' internal/mcp/scope.go && echo "no duplicated prefix logic OK"
# the three matrix-evidenced wildcard forms are handled
grep -nE '"\*"|\-\*|HasPrefix|exact' internal/mcp/scope.go
# the assignment table is the filter's live SOURCE (D2/D3 — not a dead table)
grep -nE 'ListVKMCPConfigsByVK' internal/store/mcpvkconfigs.go internal/admin/mcp.go   # defined AND called
grep -nE 'ListVKMCPConfigsByVK\(' internal/admin/mcp.go | grep -v 'func '              # ≥1 live caller
# LIVE-NARROWING proof: a restricted VK sees FEWER tools than the global catalog
grep -niE 'fewer|narrow|scoped.*vk|restricted' internal/admin/mcp_test.go | grep -iE 'tools|catalog'
# the scoped source actually filters (the filter has a live consumer)
grep -nE 'scopeTools\(' internal/admin/mcp.go | grep -v 'func '                        # ≥1 live caller
# AllowOnAllVirtualKeys BYPASS proof (D6 — flag actually bypasses)
grep -niE 'allow_on_all_virtual_keys|AllowOnAllVirtualKeys' internal/admin/mcp.go internal/store/*.go
grep -niE 'bypass|allow.*all.*vk' internal/admin/mcp_test.go
# tools/call is ALSO scope-gated (a VK cannot call what it cannot see)
grep -niE 'out.?of.?scope|tools/call.*scope|scope.*tools/call' internal/admin/mcp_test.go
# subset validation REJECTS an invalid assignment in the LIVE path (D5/049)
grep -nE 'func validateAutoExecuteSubset' internal/mcp/scope.go
grep -nE 'validateAutoExecuteSubset\(' internal/admin/mcp.go | grep -v 'func '         # ≥1 live caller
grep -niE 'subset|auto.?execute.*reject|reject.*subset' internal/admin/mcp_test.go
# ConfigHash is computed on write AND EXPOSED in the GET DTO (D8/079 — live reader, NOT write-only)
grep -niE 'config_hash|ConfigHash|sha256|Sum256' internal/admin/mcp.go internal/store/mcpvkconfigs.go
grep -niE 'config_hash' internal/admin/mcp.go | grep -iE 'json:"'                      # the DTO field is serialized OUT
grep -niE 'config_hash|drift' internal/admin/mcp_test.go                                # a test asserts the GET DTO carries it
# every flag is either READ or explicitly marked config-only (no dangling flag)
grep -niE 'disable_auto_tool_inject|is_code_mode_client|allowed_extra_headers|config-only|config_only' internal/admin/mcp.go internal/store/*.go
# additive only: no destructive DDL, no init, no mcp-go dep
git diff <base>..HEAD -- internal/store/migrate.go | grep -E '^\+' | grep -iE 'DROP|RENAME' | wc -l   # = 0
! grep -rnE 'func init\(' internal/mcp/scope.go internal/store/mcpvkconfigs.go && echo "no init() OK"
! grep -rnE 'mark3labs/mcp-go' go.mod internal/ && echo "no mcp-go dep OK"
# /mcp stays raw JSON-RPC (regression — bf-mcp-1 contract unchanged)
! grep -nE 'writeData|writeError' internal/mcp/server.go && echo "/mcp dispatch still raw JSON-RPC OK"
# code-mode execution is NOT faked (flag stored only; no VFS engine)
! grep -rniE 'vfs|virtualfilesystem|codeModeExec|executeCodeMode' internal/mcp/ internal/admin/ && echo "no faked code-mode execution OK"
# no token leak on the assignment DTO
! grep -nE 'AccessToken|RefreshToken|access_token"|refresh_token"' internal/admin/mcp.go | grep -iE 'vk.?config|assignment' && echo "no token in assignment DTO OK"
```

---

## 6. Validation

- `go build ./...` → exit 0.
- `go vet ./...` → exit 0.
- `go test ./...` → exit 0, **HERMETIC**: no real network (the filter is pure +
  table-driven; the scoped `tools/list` is tested with an injected assignment + a
  canned catalog; the store tests use the in-memory SQLite test store), no subprocess
  spawn, no real SSE timing (`/mcp` SSE is bf-mcp-1's; bf-mcp-2 does not touch it).
- `govulncheck ./...` clean for any new code path (per AGENTS.md Tools).
- All §5 grep/behavior proofs pass (the live-narrowing, bypass, subset-reject,
  config-hash-exposed-in-GET-DTO, no-faked-code-mode, and additive-only proofs).

---

## 7. Freeze rules + matrix flips + open questions

### Freeze / serial

- **`internal/server/routes_admin.go` (via `routes_mcp.go RegisterMCPRoutes`)** is the
  MCP serial slot. bf-mcp-2 is the **SECOND** holder (bf-mcp-1 → **bf-mcp-2** →
  bf-core-2; BIFROST-MAP §343-346). bf-mcp-1 RELEASED it on close. bf-mcp-2 adds ONLY
  the additive assignment-CRUD route lines. **RELEASES the slot to bf-core-2 on close.**
- `internal/admin/mcp.go` is touched only additively (new scoped source + handlers +
  flag reads); every existing handler body/signature is preserved (the absent-VK path
  is a regression-tested no-change).
- The Wave-7 CLIENT-mode `internal/mcp/{filter,allowlist,toolpolicy,…}.go` are
  REUSED by import, NEVER edited (BIFROST-MAP §397).
- The bf-mcp-1 `/mcp` wire contract + `resolveMCPVK`/`admitMCPVK` validation are
  unchanged (bf-mcp-2 consumes the resolved VK; it does not alter resolution).

### Matrix-flip targets (at close — §0.2)

| Row | → |
|---|---|
| PAR-BF-MCP-004 | MISSING → **HAVE-by-variant** (request-time lazy scope; per-VK server-object map VAR) |
| PAR-BF-MCP-017 | MISSING → **HAVE** (wildcard allow-list `*`/empty=deny) |
| PAR-BF-MCP-018 | MISSING → **PARTIAL** (subset validation + storage; server-mode auto-exec loop ESC) |
| PAR-BF-MCP-019 | MISSING → **HAVE** (`*` / `<client>-*` / `<client>-<tool>` wildcards) |
| PAR-BF-MCP-020 | MISSING → **HAVE** (`AllowOnAllVirtualKeys` bypass) |
| PAR-BF-MCP-033 | MISSING → **HAVE** (additive `virtual_key_mcp_configs` table) |
| PAR-BF-MCP-049 | MISSING → **HAVE** (subset validation rejects in the live path) |
| PAR-BF-MCP-057 | MISSING → **HAVE** if live read narrows; else **PARTIAL** (config-only) — decided at impl (D7) |
| PAR-BF-MCP-071 | MISSING → **PARTIAL** (stored + validated; forwarding ESC) |
| PAR-BF-MCP-077 | MISSING → **HAVE** if probe supplies annotations; else **PARTIAL** (shape present, no data) |
| PAR-BF-MCP-078 | MISSING → **PARTIAL** (flag stored; code-mode VFS execution ESC) |
| PAR-BF-MCP-079 | MISSING → **PARTIAL** (hash computed on write + exposed in GET DTO for drift-detection; reconciliation worker ESC) |

### Open questions (append to `.planning/parity/plans/open-questions.md` at close)

```
## bf-mcp-2 — per-VK MCP scoping + tool filtering — 2026-06-15
- [ ] Per-VK lazy creation: g0router uses the request-time-filter VARIANT (D1), not a
      per-VK long-lived server-object map — does any matrix behavior require per-VK
      server state across requests (session affinity)? — if so it is ESC, not built.
- [ ] DisableAutoToolInject (057): does g0router's server-mode have an "auto-inject vs
      explicit-request" distinction to suppress? If not, the flag is config-only
      (PARTIAL), not HAVE. — decided at impl (D7).
- [ ] AllowedExtraHeaders (071): no server-mode upstream header-forwarding path exists
      to gate; stored + validated only. The forwarding consumer is the per-user
      surface (ESC). — what would consume it?
- [ ] Tool annotations (077): does the probe/catalog source supply annotation data
      today, or is the shape added empty (PARTIAL)? — decided at impl (D8).
- [ ] IsCodeModeClient (078): flag stored + operator-set via the client admin API;
      code-mode VFS execution is ESC (§3). — what would the execution engine add?
- [ ] ConfigHash (079): hash computed on write + exposed in the assignment GET DTO for
      drift-detection (the live reader); the auto-reconciliation worker that ACTS on
      drift is ESC (§3). — what would the worker diff/re-sync?
- [ ] ToolsToAutoExecute (018): stored + subset-validated; server-mode has no
      auto-exec agent loop (CLIENT-mode ExecuteTool is the only loop). — ESC residual.
```

### `docs/WORKFLOW.md` (update at close)

Add a bf-mcp-2 row: per-VK MCP scoping — `virtual_key_mcp_configs` table +
`executeOnlyTools` wildcard filter + `AllowOnAllVirtualKeys` bypass + subset
validation; config-only flags (disable-auto-inject, allowed-extra-headers, code-mode,
config-hash); rows 004/017/018/019/020/033/049/057/071/077/078/079 flipped; serial
slot released to bf-core-2.

### No-leftovers confirmation

Every NEW surface has a live consumer OR an explicit config-only marking: the
`virtual_key_mcp_configs` table (read by the scoped `CatalogSource`), `scopeTools`
(narrows `tools/list`+`tools/call`), `validateAutoExecuteSubset` (rejects invalid
assignments in the live path), `AllowOnAllVirtualKeys` (bypasses the filter),
`ConfigHash` (computed on write AND exposed in the assignment GET DTO for
drift-detection — its live reader), and the config-only flags
(`DisableAutoToolInject`/`AllowedExtraHeaders`/`IsCodeModeClient`) operator-set +
validated through the admin API with their missing execution/enforcement recorded ESC.
No field is write-only: each is read to change behavior, operator-set via the admin
API, or exposed in a GET DTO for observability. **No dead table, no dead column, no
dangling flag, no filter that never narrows, no bypass that never bypasses, no faked
code-mode execution.** Code-mode VFS execution and the reconciliation worker are ESC,
honestly recorded — never faked.
```
