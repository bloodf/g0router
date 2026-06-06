# Docs Audit — g0router

**Date**: 2026-06-05  
**Scope**: `docs/WORKFLOW.md`, `docs/phases/*.md`, `docs/README.md`, `docs/PROVIDERS.md`, `docs/SCHEMA.md`, `docs/ARCHITECTURE.md`, `docs/PLAN.md`, `docs/REFERENCES.md`, `docs/DIRECTORY_STRUCTURE.md`, `docs/ORCHESTRATION.md`, `docs/CONFIG.md`, `docs/DEPLOYMENT.md`, `README.md`, `AGENTS.md`  
**Method**: Read-only source verification. No edits made.

---

## Summary Counts

| Category | Count |
|---|---|
| False/stale capability claims | 6 |
| Phantom file/path references | 3 |
| Brand refs — `bifrost` (non-evaluation docs) | 20 + 1 (README.md) |
| Brand refs — `9router` (non-evaluation docs) | 14 + 1 (README.md) + 1 (AGENTS.md) |
| Brand refs — `oh-my-pi` / OMP (non-evaluation docs) | 12 + 1 (README.md) + 1 (AGENTS.md) |
| Evaluation files with brand refs | 96 files |

---

## FALSE / STALE CLAIMS

### CRITICAL

**C1. `docs/ARCHITECTURE.md:6` — "23+ providers" wrong; matrix has 44**

> `- **bifrost's** provider engine (23+ providers, fasthttp, object pooling, MCP gateway)`

Source: `internal/provider/matrix.go:42–86` has exactly **43 entries** (openai, anthropic, azure, bedrock, cerebras, cohere, deepseek, fireworks, gemini, groq, huggingface, mistral, nebius, nvidia, ollama, openrouter, perplexity, replicate, together, vertex, antigravity, github-copilot, cursor, gitlab-duo, kimi, kiro, xai, xiaomi, alibaba, minimax, zhipu, cloudflare-ai-gateway, kagi, kilo, litellm, lm-studio, ollama-cloud, opencode, qianfan, qwen, tavily, vercel-ai-gateway, vllm). Claim of "23+" is off by 20.

**C2. `docs/ARCHITECTURE.md:8` — "50+ providers" (oh-my-pi OAuth catalog) is stale framing**

> `- **oh-my-pi's** OAuth catalog (50+ providers) and credential storage patterns`

This attributes a provider count to a legacy project. The actual g0router matrix (source of truth) has 43 entries. The "50+" figure does not correspond to anything in the current codebase and cannot be verified.

**C3. `docs/ARCHITECTURE.md:110` — "bifrost's 30+ methods" is misleading**

> `// implements this. Simplified from bifrost's 30+ methods to LLM essentials.`

The current `Provider` interface (`internal/providers/interface.go`) has 3 methods: `Name()`, `ChatCompletion()`, `ChatCompletionStream()`, plus `ListModels()` — not simplified from a real 30-method bifrost interface that exists in this repo. This is historical attribution, not a verifiable code comment.

**C4. `docs/ARCHITECTURE.md:6` — "object pooling" is false**

> `- **bifrost's** provider engine (23+ providers, fasthttp, object pooling, MCP gateway)`

`internal/proxy/pool.go:9–14` shows `providerPool` is a `map[providers.ModelProvider]providers.Provider` — a plain registry map. No `sync.Pool`, no object reuse. `grep -rn "sync.Pool"` across all non-test Go source returns zero results.

### HIGH

**H1. `docs/ARCHITECTURE.md:99` — phantom package `api/integrations/` does not exist**

> `└── api/integrations/        (SDK compatibility)`

Actual directory: `api/` contains only `handlers/`, `middleware.go`, `server.go`, and test files. No `integrations/` subdirectory exists. Confirmed by `ls api/` and `find . -name "integrations" -type d` returning nothing.

**H2. `docs/DIRECTORY_STRUCTURE.md:30–31` — `api/integrations/openai.go` phantom**

> `│   └── integrations/`  
> `│       └── openai.go   # OpenAI SDK compatibility helpers`

This file does not exist. The doc's own preamble says "Target repository layout after all phases are complete," but the project is marked `project_status: COMPLETE` in `docs/WORKFLOW.md:31`. This path was never created; the OpenAI-compat logic lives in `internal/providers/openaicompat/`.

**H3. `docs/DIRECTORY_STRUCTURE.md:36–41` — CLI file names phantom vs actual**

Listed phantom files:
- `internal/cli/serve.go` — does not exist; `serve` command is in `root.go:87`
- `internal/cli/login.go` — does not exist; login is in `auth.go`
- `internal/cli/logout.go` — does not exist; logout is in `auth.go:181`
- `internal/cli/keys.go` — does not exist; keys are in `root.go:399`
- `internal/cli/status.go` — does not exist; status is in `root.go:551`
- `internal/cli/providers.go` — does not exist; providers are in `root.go:492`

Actual `internal/cli/` files: `auth.go`, `auth_test.go`, `install.go`, `install_test.go`, `keys_test.go`, `mcp_auth.go`, `mcp_auth_test.go`, `mcp_http_runtime_test.go`, `mcp_runtime.go`, `mcp_runtime_test.go`, `mcp_test.go`, `provider_runtime.go`, `providers_test.go`, `root.go`, `root_test.go`. All CLI commands are consolidated in `root.go` rather than split into per-command files.

Again: doc's own preamble says "target layout," but project is COMPLETE. Target != actual.

### MEDIUM

**M1. `docs/REFERENCES.md:61` — maps to `api/integrations/openai.go` which was never created**

> `| transports/bifrost-http/integrations/openai.go (115KB) | api/integrations/openai.go (simplified) |`

The g0router target path `api/integrations/openai.go` does not exist. The reference guide points to a phantom destination. REFERENCES.md is now a historical artifact (project complete) but misleads anyone reading it.

**M2. `docs/ARCHITECTURE.md:15` — diagram shows "fasthttp server" as a separate box suggesting it replaces cobra**

```
│  │  cobra   │   │  fasthttp      │
│  │  CLI     │   │  server        │
```

fasthttp IS the HTTP server (`api/server.go:25,49` confirms `*fasthttp.Server`). This part is accurate. However the diagram omits the fact that serve, login, logout, keys, mcp, providers, status, healthcheck, install, uninstall, version commands are all registered in a single `root.go` cobra tree — not split files as the DIRECTORY_STRUCTURE.md implies.

**M3. `docs/DIRECTORY_STRUCTURE.md` title claim: "~140 Go files"**

> `Target repository layout after all phases are complete. Files organized by Go package convention.`

Actual non-test Go source file count: **135** (`find . -name "*.go" -not -name "*_test.go" | wc -l`). Close but not exactly 140; acceptable tolerance given it's a "~" estimate.

### LOW

**L1. `docs/PLAN.md:85` — REFERENCES.md described as "Source mapping from bifrost/9router/oh-my-pi"**

The project is COMPLETE. REFERENCES.md is a historical porting guide for a code migration that already happened. Its description should reflect "historical lineage reference" rather than implying active porting work.

**L2. `docs/WORKFLOW.md` — project_status COMPLETE but doc is 227KB**

The WORKFLOW.md is 5315 lines of historical wave data. `project_status: COMPLETE` is correct at line 31. No stale PENDING/IN_PROGRESS statuses found in current-state block. This is Low because it's correct but unwieldy.

---

## CAPABILITIES IN SOURCE NOT DOCUMENTED

**U1. `g0router healthcheck` and `g0router uninstall` commands not in SCHEMA.md CLI section**

`root.go:78,82` registers `newHealthcheckCommand()` and `newUninstallCommand()`. `docs/SCHEMA.md:285–300` CLI section does not list `g0router healthcheck` or `g0router uninstall`.

**U2. `g0router auth` sub-command not listed in SCHEMA.md**

`root.go:71` adds `newAuthCommand()`. SCHEMA.md CLI section omits `g0router auth`. `g0router login` is an alias for backwards compat; `auth login` / `auth logout` / `auth list` are the canonical forms per `auth.go`.

**U3. `g0router mcp` sub-commands not fully listed in SCHEMA.md**

`root.go:75` adds `newMCPCommand()`. SCHEMA.md CLI section (`docs/SCHEMA.md:293–298`) lists only `g0router mcp add`, `mcp auth start`, `mcp auth complete`, `mcp list`, `mcp rm`. Source (`mcp_auth.go`, `mcp_runtime.go`) implements additional sub-commands but SCHEMA.md does not provide a complete listing.

---

## BRAND REFERENCES TO STRIP

All occurrences of `bifrost`, `9router`, `oh-my-pi`, `omp`, `OMP`, `9Router`, `Bifrost` in non-evaluation documentation that should be rewritten as neutral standalone descriptions or removed.

### `README.md` (root)

| Line | Text | Action |
|---|---|---|
| 81 | `\| [docs/REFERENCES.md](docs/REFERENCES.md) \| Source mapping from 9router/bifrost/oh-my-pi \|` | Rewrite: "Historical lineage and porting reference" |
| 112 | `- **[bifrost](https://github.com/maximhq/bifrost)** — Provider engine, fasthttp, object pooling, MCP` | Remove entire Lineage section or rewrite as neutral "Inspiration" note without URLs |
| 113 | `- **[9router](https://github.com/decolua/9router)** — OAuth flows, RTK, caveman, cost tracking, combos, UI` | Same |
| 114 | `- **[oh-my-pi](https://github.com/can1357/oh-my-pi)** — OAuth catalog (50+ providers), credential storage` | Same |

### `AGENTS.md` (root)

| Line | Text | Action |
|---|---|---|
| 71 | `**What**: Go LLM gateway/proxy combining 9router + bifrost + oh-my-pi patterns.` | Rewrite: "Go LLM gateway/proxy. Single binary." |

### `docs/ARCHITECTURE.md`

| Line | Text | Action |
|---|---|---|
| 6 | `- **bifrost's** provider engine (23+ providers, fasthttp, object pooling, MCP gateway)` | Rewrite: "Provider engine (43 providers, fasthttp HTTP transport, MCP gateway)" |
| 7 | `- **9router's** OAuth flows, RTK compression, caveman, cost tracking, combo models, minimal UI` | Rewrite: "OAuth flows, RTK compression, caveman, cost tracking, combo models, minimal UI" |
| 8 | `- **oh-my-pi's** OAuth catalog (50+ providers) and credential storage patterns` | Rewrite: "OAuth catalog (43 providers) and credential storage" |
| 110 | `// implements this. Simplified from bifrost's 30+ methods to LLM essentials.` | Rewrite: `// implements this. Three methods covering LLM inference essentials.` |

### `docs/PROVIDERS.md`

| Line | Text | Action |
|---|---|---|
| 23 | Column header `\| OMP ID \| 9Router ID \| Bifrost ID \|` | Remove these three columns entirely — they duplicate g0router ID for all 43 rows and are vestigial porting scaffolding. The `internal/provider/matrix.go` struct still has `OMPID`, `Router9ID`, `BifrostID` fields that are populated with identical values; those struct fields should also be pruned eventually (out of scope here). |

### `docs/PLAN.md`

| Line | Text | Action |
|---|---|---|
| 3 | `> **g0router** — Go LLM gateway combining 9router + bifrost + oh-my-pi.` | Rewrite: `> **g0router** — Go LLM gateway and provider router.` |
| 85 | `\| [REFERENCES.md](REFERENCES.md) \| Source mapping from bifrost/9router/oh-my-pi \|` | Rewrite: "Historical lineage reference" |

### `docs/REFERENCES.md`

Entire file is a porting guide for code migration from three legacy repos. The project is COMPLETE; no active porting occurs. Options:
1. Delete the file and update all links pointing to it.
2. Retitle as "Historical Lineage" and add a banner: `> This document records the original source mapping used during initial development. All ported code is now in the g0router codebase.`

Direct brand occurrences: lines 6, 7, 8, 11, 14, 16, 23, 35, 42, 45, 52, 57, 59, 60, 61, 65, 68, 76, 83, 102, 108, 119, 122, 141, 147 — the entire document structure is `bifrost → g0router`, `9router → g0router`, `oh-my-pi → g0router` mapping tables.

### `docs/README.md`

| Line | Text | Action |
|---|---|---|
| 25 | `\| [REFERENCES.md](REFERENCES.md) \| When porting code from bifrost, 9router, or oh-my-pi \|` | Rewrite: "Historical lineage reference" or remove row |

### `docs/DIRECTORY_STRUCTURE.md`

| Line | Text | Action |
|---|---|---|
| 227 | `# Source file mapping (bifrost/9router/oh-my-pi)` | Rewrite: "# Historical lineage reference" |

### `docs/phases/phase-02-http-server-proxy-engine.md`

| Line | Text | Action |
|---|---|---|
| 20 | `bifrost uses it` (rationale for fasthttp) | Rewrite: "used for high-throughput proxy workloads" |

### `docs/phases/phase-05-oauth-flows-cli.md`

| Line | Text | Action |
|---|---|---|
| 20 | `Same port as oh-my-pi; avoids random port issues with pre-registered redirect URIs` | Rewrite: "fixed port avoids random port issues with pre-registered redirect URIs" |
| 24 | `Same as 9router/oh-my-pi; these are public OAuth client IDs` | Rewrite: "these are public OAuth client IDs, not secrets" |

### `docs/phases/phase-06-account-fallback-combos.md`

| Line | Text | Action |
|---|---|---|
| 19 | `Standard; matches 9router behavior` | Rewrite: "Standard exponential backoff" |

### Evaluation files (96 files in `docs/evaluations/`)

All 96 evaluator prompt files contain brand references. These are historical test fixtures and their brand refs are inside evaluator instructions, not user-facing documentation. Recommend treating as an independent cleanup pass; the main docs above are the priority.

---

## TOP 3 DOC-VS-SOURCE MISMATCHES

1. **"23+ providers" (ARCHITECTURE.md:6) vs 43 entries in matrix.go** — the core capability count is off by nearly 2×, and the "object pooling" attribution is false (it's a plain map).

2. **`api/integrations/` package (ARCHITECTURE.md:99, DIRECTORY_STRUCTURE.md:30, REFERENCES.md:61) vs reality** — the package was planned, never created. OpenAI compat logic lives in `internal/providers/openaicompat/`. The project is marked COMPLETE with this phantom still in three doc files.

3. **`docs/DIRECTORY_STRUCTURE.md` CLI split-file layout vs `root.go`** — doc claims 6 separate files (`serve.go`, `login.go`, `logout.go`, `keys.go`, `status.go`, `providers.go`) that do not exist; all commands are in `root.go`. This is the biggest structural divergence between the "target layout" and actual implementation.
