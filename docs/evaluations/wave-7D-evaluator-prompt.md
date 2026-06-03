# g0router Wave 7.D Evaluation Prompt

Evaluate completed wave `7.D` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

## Review Inputs

- `AGENTS.md`
- `CLAUDE.md`
- `README.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PROVIDERS.md`
- `docs/SCHEMA.md`
- Relevant provider/auth/runtime code:
  - `internal/provider/matrix.go`
  - `internal/provider/ids.go`
  - `internal/provider/oauth/`
  - `internal/providers/`
  - `internal/cli/provider_runtime.go`
  - `internal/proxy/engine.go`
  - `api/handlers/providers.go`
- Commit refs:
  - range `253b157..HEAD`

## Check

- `internal/provider/matrix.go` is the source of truth for provider status and capability metadata.
- The first remediation tier is represented explicitly:
  - OpenAI/Codex
  - Anthropic Claude
  - Gemini/Antigravity
  - GitHub Copilot
  - Cursor
  - DeepSeek
  - Kimi/Moonshot
  - Qwen
  - Perplexity
  - OpenRouter
  - Groq
  - Mistral
  - Cerebras
  - Fireworks
  - Together
  - Nvidia
  - HuggingFace
  - xAI
  - Azure
  - Vertex
  - Bedrock
  - Ollama
- The second remediation tier is represented explicitly:
  - Vercel AI Gateway
  - Cloudflare AI Gateway
  - LiteLLM
  - vLLM
  - LM Studio
  - Ollama Cloud
  - Kilo/OpenCode
  - Zhipu/ZAI
  - Xiaomi
  - MiniMax
  - Alibaba/Qianfan
  - Tavily/Kagi
- Additional implemented registry/auth surfaces are represented, including Cohere, Replicate, Nebius, GitLab, and Kiro.
- `supported` means public direct-dispatch inference works today, not merely that code exists.
- Registered but not generally routable adapters are `adapter_only`, not `supported`.
- Auth-only providers are `auth_only`, not advertised through `g0router providers list`.
- Unsupported providers are explicitly `unsupported` and not exposed through placeholder models or inert CLI provider names.
- `GET /api/providers` returns status/capability fields derived from the matrix.
- `g0router providers list` prints only public direct-dispatch providers.
- README and provider docs do not advertise adapter-only, auth-only, or unsupported providers as generally supported.
- Workflow status accurately marks Wave 7.D complete and advances to Wave 7.E.
- Existing `.DS_Store`, `.pi/`, and untracked `AGENTS.md` state was not cleaned up or committed.

## Known Deferred Work

- Catalog-driven model routing, aliases, provider capabilities, combo hardening, fallback/backoff, usage/cost/quota path integration, and `/v1/messages`/`/v1/responses` route hardening remain Wave 7.E work.
- Live streaming correctness and provider-specific error mapping remain Wave 7.F work.
- Real dashboard data replacement remains Wave 7.H work.

## Gates

Run:

```bash
go test ./... -count=1
go vet ./...
go build ./cmd/g0router
npm --prefix ui test -- --run
npm --prefix ui run build
make build
```

## Return

```markdown
## Verdict

PASS or FAIL

## Blocking Findings

Issues that must be fixed before Wave 7.E.

## Non-Blocking Findings

Risks or cleanup notes.

## Gate Results

Command results with exact failures if any.

## Workflow Status Review

Whether `docs/WORKFLOW.md` accurately reflects Wave 7.D and advances to Wave 7.E.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
```
