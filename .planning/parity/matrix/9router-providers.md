# 9router Provider Adapter Matrix

Reference SHA: `827e5c3` (v0.4.71).  
Generated: 2026-06-09.

---

## Row table

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|---|---|---|---|---|
| PAR-PROV-001 | **openai** — API-key auth, baseUrl `https://api.openai.com/v1/chat/completions`, format `openai`, static model catalog (gpt-5.4, gpt-4o, o3, embedding, tts, stt, image models). | `open-sse/config/providers.js:123`, `open-sse/config/providerModels.js:227-266` | HAVE | Chat + embeddings implemented. Most other methods stubbed. `internal/providers/openai/chat.go:15`, `internal/providers/openai/embedding.go:12` |
| PAR-PROV-002 | **anthropic** — API-key auth (x-api-key header), baseUrl `https://api.anthropic.com/v1/messages`, format `claude`, static catalog (claude-sonnet-4, claude-opus-4, claude-3-5-sonnet). | `open-sse/config/providers.js:252-255`, `open-sse/config/providerModels.js:267-271` | HAVE | Chat implemented. Embeddings stubbed. `internal/providers/anthropic/chat.go:15` |
| PAR-PROV-003 | **gemini** — OAuth or API-key (x-goog-api-key header), baseUrl `https://generativelanguage.googleapis.com/v1beta/models`, format `gemini`, static catalog (gemini-3.1-pro, gemini-2.5-pro, embedding, image, stt models). URL built as `/model:streamGenerateContent?alt=sse`. | `open-sse/config/providers.js:58-62`, `open-sse/config/providerModels.js:272-301`, `open-sse/executors/default.js:60-61` | HAVE | Chat + embeddings implemented. `internal/providers/gemini/chat.go:16`, `internal/providers/gemini/embedding.go:12` |
| PAR-PROV-004 | **groq** — API-key auth, baseUrl `https://api.groq.com/openai/v1/chat/completions`, format `openai`, static catalog (llama-3.3-70b, llama-4-maverick, qwen3-32b, whisper stt models). | `open-sse/config/providers.js:269-271`, `open-sse/config/providerModels.js:459-468` | HAVE | Declared in `internal/schemas/provider.go:11`. Directory exists but only `doc.go` + test stub. |
| PAR-PROV-005 | **deepseek** — API-key auth, baseUrl `https://api.deepseek.com/chat/completions`, format `openai`, static catalog (deepseek-v4-pro, deepseek-chat, deepseek-reasoner). | `open-sse/config/providers.js:257-259`, `open-sse/config/providerModels.js:438-445` | HAVE | Declared in `internal/schemas/provider.go:16`. Directory exists but only `doc.go` + test stub. |
| PAR-PROV-006 | **mistral** — API-key auth, baseUrl `https://api.mistral.ai/v1/chat/completions`, format `openai`, static catalog (mistral-large, codestral, mistral-embed). | `open-sse/config/providers.js:281-283`, `open-sse/config/providerModels.js:476-481` | HAVE | Declared in `internal/schemas/provider.go:12`. Directory exists but only `doc.go` + test stub. |
| PAR-PROV-007 | **cohere** — API-key auth, baseUrl `https://api.cohere.ai/v1/chat/completions`, format `openai`, static catalog (command-r-plus, command-r, command-a). | `open-sse/config/providers.js:301-303`, `open-sse/config/providerModels.js:508-512` | HAVE | Declared in `internal/schemas/provider.go:13`. Directory exists but only `doc.go` + test stub. |
| PAR-PROV-008 | **together** — API-key auth, baseUrl `https://api.together.xyz/v1/chat/completions`, format `openai`, static catalog (llama-3.3-70b, deepseek-r1, qwen3-235b, embedding models). | `open-sse/config/providers.js:289-291`, `open-sse/config/providerModels.js:486-493` | HAVE | Declared in `internal/schemas/provider.go:15`. Directory exists but only `doc.go` + test stub. |
| PAR-PROV-009 | **fireworks** — API-key auth, baseUrl `https://api.fireworks.ai/inference/v1/chat/completions`, format `openai`, static catalog (deepseek-v3p1, llama-v3p3-70b, nomic-embed). | `open-sse/config/providers.js:293-295`, `open-sse/config/providerModels.js:494-499` | HAVE | Declared in `internal/schemas/provider.go:14`. Directory exists but only `doc.go` + test stub. |
| PAR-PROV-010 | **ollama / ollama-local** — No auth (local), baseUrl `https://ollama.com/api/chat` or `http://localhost:11434/api/chat`, format `ollama`, static catalog (gpt-oss:120b, kimi-k2.5, glm-5). | `open-sse/config/providers.js:333-339`, `open-sse/config/providerModels.js:572-578`, `open-sse/config/providers.js:440` | HAVE | Declared in `internal/schemas/provider.go:18`. Directory exists but only `doc.go` + test stub. Ollama uses NDJSON format, not SSE. |
| PAR-PROV-011 | **bedrock** — Not present in 9router providers.js. g0router declares `ProviderBedrock` in schemas. | `internal/schemas/provider.go:19` | EXTRA | g0router has schema constant but no implementation directory content (empty dir). 9router has no bedrock adapter. |
| PAR-PROV-012 | **vertex** — Service-account JSON auth, baseUrl dynamically built by `VertexExecutor.buildUrl()`, format `vertex` (native) or `openai` (partner). Static catalog for both vertex and vertex-partner. | `open-sse/config/providers.js:343-352`, `open-sse/config/providerModels.js:580-591`, `open-sse/executors/vertex.js` | MISSING | Declared in `internal/schemas/provider.go:20`. Directory exists but only `doc.go` + test stub. |
| PAR-PROV-013 | **minimax / minimax-cn** — API-key auth (x-api-key header), baseUrl `https://api.minimax.io/anthropic/v1/messages` or `https://api.minimaxi.com/anthropic/v1/messages`, format `claude`, URL suffix `?beta=true`. Static catalog (MiniMax-M3, M2.7, M2.5, image model). | `open-sse/config/providers.js:146-154`, `open-sse/config/providerModels.js:340-347`, `open-sse/executors/default.js:52-55` | MISSING | ESC-1 (w7-prov-openai): format `claude` (Anthropic Messages wire format). Generic adapter is openai-only. Excluded from openai catalog track; belongs to claude-format/specialized track. |
| PAR-PROV-014 | **openrouter** — API-key auth, baseUrl `https://openrouter.ai/api/v1/chat/completions`, format `openai`, static catalog (embedding, tts, image models). Headers include `HTTP-Referer` and `X-Title`. | `open-sse/config/providers.js:115-121`, `open-sse/config/providerModels.js:302-320` | HAVE | Declared in `internal/schemas/provider.go:21`. No `internal/providers/openrouter` directory exists at all. |
| PAR-PROV-015 | **claude (OAuth alias `cc`)** — OAuth device-code flow, baseUrl `https://api.anthropic.com/v1/messages`, format `claude`, spoofed Claude CLI headers, tokenUrl `https://api.anthropic.com/v1/oauth/token`. Static catalog (claude-opus-4-8 through claude-haiku-4-5). | `open-sse/config/providers.js:51-56`, `open-sse/config/providerModels.js:31-39`, `open-sse/executors/default.js:81-113` | MISSING | g0router has `anthropic` (API-key) but not OAuth `claude` flow. `DefaultExecutor.refreshCredentials` has `claude` refresher at `open-sse/executors/default.js:190`. |
| PAR-PROV-016 | **codex (OAuth alias `cx`)** — OAuth, baseUrl `https://chatgpt.com/backend-api/codex/responses`, format `openai-responses`, headers `originator: codex_cli_rs`, tokenUrl `https://auth.openai.com/oauth/token`. Static catalog with review-model suffix generation. | `open-sse/config/providers.js:70-78`, `open-sse/config/providerModels.js:40-55`, `open-sse/executors/codex.js` | MISSING | No g0router equivalent. Specialized `CodexExecutor` handles responses API. `DefaultExecutor.refreshCredentials` has `codex` refresher at `open-sse/executors/default.js:191`. |
| PAR-PROV-017 | **gemini-cli (OAuth alias `gc`)** — OAuth, baseUrl `https://cloudcode-pa.googleapis.com/v1internal`, format `gemini-cli`. Static catalog (gemini-3-flash-preview, gemini-3-pro-preview). | `open-sse/config/providers.js:64-68`, `open-sse/config/providerModels.js:56-59` | MISSING | No g0router equivalent. `GeminiCLIExecutor` is specialized. |
| PAR-PROV-018 | **qwen (OAuth alias `qw`)** — OAuth device-code flow, baseUrl `https://portal.qwen.ai/v1/chat/completions`, format `openai`, tokenUrl `https://chat.qwen.ai/api/v1/oauth2/token`, authUrl `https://chat.qwen.ai/api/v1/oauth2/device/code`. Static catalog (qwen3-coder-plus, qwen3-coder-flash, vision-model, coder-model). | `open-sse/config/providers.js:80-85`, `open-sse/config/providerModels.js:60-66`, `open-sse/executors/qwen.js` | MISSING | No g0router equivalent. `QwenExecutor` is specialized. `DefaultExecutor.refreshCredentials` has `qwen` refresher at `open-sse/executors/default.js:192`. |
| PAR-PROV-019 | **iflow (OAuth alias `if`)** — OAuth, baseUrl `https://apis.iflow.cn/v1/chat/completions`, format `openai`, tokenUrl `https://iflow.cn/oauth/token`, authUrl `https://iflow.cn/oauth`. Client credentials used in refresh. Static catalog (qwen3-coder-plus, qwen3-max, deepseek-v3.2, kimi-k2, glm-4.7, iflow-rome). | `open-sse/config/providers.js:87-94`, `open-sse/config/providerModels.js:67-83`, `open-sse/executors/default.js:236-246` | MISSING | No g0router equivalent. Refresh uses Basic auth with clientId:clientSecret. |
| PAR-PROV-020 | **antigravity (OAuth alias `ag`)** — OAuth, baseUrls list with fallback (`daily-cloudcode-pa.googleapis.com` and sandbox), format `antigravity`, headers include `User-Agent: antigravity/1.107.0`. Static catalog calls different backends (gemini, claude, gpt-oss). | `open-sse/config/providers.js:105-113`, `open-sse/config/providerModels.js:84-94`, `open-sse/executors/antigravity.js` | MISSING | No g0router equivalent. `AntigravityExecutor` is specialized, handles backend routing per model. |
| PAR-PROV-021 | **github (OAuth alias `gh`)** — OAuth, baseUrl `https://api.githubcopilot.com/chat/completions`, responsesUrl `https://api.githubcopilot.com/responses`, format `openai`, extensive Copilot IDE headers (editor-version, copilot-integration-id, x-github-api-version). Static catalog spans OpenAI, Anthropic, Google, Grok, embedding models. | `open-sse/config/providers.js:176-192`, `open-sse/config/providerModels.js:95-126`, `open-sse/executors/github.js` | MISSING | No g0router equivalent. `GithubExecutor` is specialized. |
| PAR-PROV-022 | **kiro (OAuth alias `kr`)** — OAuth, baseUrl `https://codewhisperer.us-east-1.amazonaws.com/generateAssistantResponse`, format `kiro`, AWS eventstream headers (`Accept: application/vnd.amazon.eventstream`, `X-Amz-Target`), custom JSON refresh. Static catalog with thinking/agentic variants and strip lists. | `open-sse/config/providers.js:194-206`, `open-sse/config/providerModels.js:127-146`, `open-sse/executors/kiro.js` | MISSING | No g0router equivalent. `KiroExecutor` is specialized. Refresh at `open-sse/executors/default.js:259-268`. |
| PAR-PROV-023 | **cursor (OAuth alias `cu`)** — OAuth, baseUrl `https://api2.cursor.sh`, chatPath `/aiserver.v1.ChatService/StreamUnifiedChatWithTools`, format `cursor`, connect+proto headers (`Content-Type: application/connect+proto`). Static catalog (claude-4.5-opus, gpt-5.2-codex, kimi-k2.5). | `open-sse/config/providers.js:208-218`, `open-sse/config/providerModels.js:163-178`, `open-sse/executors/cursor.js` | MISSING | No g0router equivalent. `CursorExecutor` is specialized, uses protobuf streaming. |
| PAR-PROV-024 | **kimi-coding (OAuth alias `kmc`)** — OAuth, baseUrl `https://api.kimi.com/coding/v1/messages`, format `claude`, headers `x-api-key` + `buildKimiHeaders()`, tokenUrl `https://auth.kimi.com/api/oauth/token`. URL suffix `?beta=true`. Static catalog (kimi-k2.6, kimi-k2.5, kimi-latest). | `open-sse/config/providers.js:220-226`, `open-sse/config/providerModels.js:179-184`, `open-sse/executors/default.js:58-59`, `open-sse/executors/default.js:121-122` | MISSING | No g0router equivalent. g0router has `kimi` (API-key, static catalog) in 9router but no `kimi-coding` OAuth variant. |
| PAR-PROV-025 | **cline (OAuth alias `cl`)** — OAuth, baseUrl `https://api.cline.bot/api/v1/chat/completions`, format `openai`, tokenUrl `https://api.cline.bot/api/v1/auth/token`, refreshUrl `https://api.cline.bot/api/v1/auth/refresh`, headers built by `buildClineHeaders`. Static catalog (claude-opus-4.7, gpt-5.3-codex, gemini-3.1-pro). | `open-sse/config/providers.js:238-246`, `open-sse/config/providerModels.js:215-224`, `open-sse/executors/default.js:144-145`, `open-sse/executors/default.js:270-290` | MISSING | No g0router equivalent. |
| PAR-PROV-026 | **kilocode (OAuth alias `kc`)** — OAuth, baseUrl `https://api.kilo.ai/api/openrouter/chat/completions`, format `openai`, device-code flow (no refresh token support). Headers include `X-Kilocode-OrganizationID` when `orgId` in providerSpecificData. Static catalog (claude-sonnet-4, gemini-2.5-pro, gpt-4.1, deepseek-chat). | `open-sse/config/providers.js:228-232`, `open-sse/config/providerModels.js:185-194`, `open-sse/executors/default.js:139-143`, `open-sse/executors/default.js:308-311` | MISSING | No g0router equivalent. Refresh returns null explicitly. |
| PAR-PROV-027 | **xai** — OAuth (clientId, tokenUrl, refreshUrl), baseUrl `https://api.x.ai/v1/chat/completions`, responsesUrl `https://api.x.ai/v1/responses`, format `openai`. Static catalog (grok-4, grok-4-fast-reasoning, grok-code-fast-1, grok-3, grok-2-image). | `open-sse/config/providers.js:273-279`, `open-sse/config/providerModels.js:469-475` | HAVE | No g0router equivalent. |
| PAR-PROV-028 | **qoder (alias `qd`)** — No OAuth fields; executor builds full URL itself with `?Encode=1` + sigPath query params. baseUrl kept for introspection only. format `openai`. Static tier + frontier catalog (auto, ultimate, qmodel, dmodel, etc.). | `open-sse/config/providers.js:96-103`, `open-sse/config/providerModels.js:147-162`, `open-sse/executors/qoder.js` | MISSING | No g0router equivalent. `QoderExecutor` is specialized. |
| PAR-PROV-029 | **perplexity** — API-key auth, baseUrl `https://api.perplexity.ai/chat/completions`, format `openai`. Static catalog (sonar-pro, sonar). | `open-sse/config/providers.js:285-287`, `open-sse/config/providerModels.js:482-485` | HAVE | No g0router equivalent. |
| PAR-PROV-030 | **perplexity-web** — Cookie auth (`authType: "cookie"`), baseUrl `https://www.perplexity.ai/rest/sse/perplexity_ask`, format `perplexity-web`. Static catalog (pplx-auto, pplx-sonar, pplx-gpt, pplx-gemini, etc.). | `open-sse/config/providers.js:379-382`, `open-sse/config/providerModels.js:606-614`, `open-sse/executors/perplexity-web.js` | MISSING | No g0router equivalent. Reverse-engineered web endpoint. |
| PAR-PROV-031 | **grok-web** — Cookie auth (`authType: "cookie"`), baseUrl `https://grok.com/rest/app-chat/conversations/new`, format `grok-web`. Static catalog (grok-3 through grok-4.2). | `open-sse/config/providers.js:374-377`, `open-sse/config/providerModels.js:592-605`, `open-sse/executors/grok-web.js` | MISSING | No g0router equivalent. Reverse-engineered web endpoint. |
| PAR-PROV-032 | **azure** — API-key auth, baseUrl empty string (`""`), format `openai`, headers empty. No static catalog in providerModels. | `open-sse/config/providers.js:384-387`, `open-sse/executors/azure.js` | MISSING | No g0router equivalent. `AzureExecutor` is specialized (likely builds resource-specific URL). |
| PAR-PROV-033 | **cloudflare-ai** — API-key auth, baseUrl template `https://api.cloudflare.com/client/v4/accounts/{accountId}/ai/v1/chat/completions`, format `openai`. `{accountId}` resolved from `credentials.providerSpecificData.accountId`. Static catalog (Llama, Mistral, DeepSeek, Moonshot, Qwen, FLUX image models). | `open-sse/config/providers.js:390-392`, `open-sse/config/providerModels.js:403-428`, `open-sse/executors/default.js:64-68` | MISSING | No g0router equivalent. URL template substitution at runtime. |
| PAR-PROV-034 | **glm** — API-key auth (x-api-key header), baseUrl `https://api.z.ai/api/anthropic/v1/messages`, format `claude`, URL suffix `?beta=true`. Static catalog (glm-5.1, glm-5, glm-4.7, glm-4.6v). | `open-sse/config/providers.js:131-134`, `open-sse/config/providerModels.js:321-326`, `open-sse/executors/default.js:52-55` | MISSING | ESC-1 (w7-prov-openai): format `claude` (Anthropic Messages wire format). Generic adapter is openai-only. Excluded from openai catalog track; belongs to claude-format/specialized track. Note: glm-cn (PAR-PROV-035) IS openai-format and IS implemented. |
| PAR-PROV-035 | **glm-cn** — API-key auth, baseUrl `https://open.bigmodel.cn/api/coding/paas/v4/chat/completions`, format `openai`. Static catalog (glm-5.1, glm-5, glm-4.7, glm-4.6, glm-4.5-air). | `open-sse/config/providers.js:136-139`, `open-sse/config/providerModels.js:327-333` | HAVE | w7-prov-openai: `internal/providers/catalog/catalog.go` + `models.go`. Alias `glm-cn` added to `aliases.go`. 5-model static block. |
| PAR-PROV-036 | **kimi** — API-key auth (x-api-key header), baseUrl `https://api.kimi.com/coding/v1/messages`, format `claude`, URL suffix `?beta=true`. Static catalog (kimi-k2.6, kimi-k2.5, kimi-latest). | `open-sse/config/providers.js:141-144`, `open-sse/config/providerModels.js:334-339`, `open-sse/executors/default.js:52-55` | MISSING | ESC-1 (w7-prov-openai): format `claude` (Anthropic Messages wire format). Generic adapter is openai-only. Excluded from openai catalog track; belongs to claude-format/specialized track. |
| PAR-PROV-037 | **alicode / alicode-intl** — API-key auth, baseUrl `https://coding.dashscope.aliyuncs.com/v1/chat/completions` or `https://coding-intl.dashscope.aliyuncs.com/v1/chat/completions`, format `openai`. Static catalog (qwen3.5-plus, kimi-k2.5, glm-5, MiniMax-M2.5, qwen3-coder-next). | `open-sse/config/providers.js:156-164`, `open-sse/config/providerModels.js:373-391` | HAVE | w7-prov-openai: `internal/providers/catalog/catalog.go` + `models.go`. Aliases `alicode`, `alicode-intl` added. 8-model and 7-model static blocks. |
| PAR-PROV-038 | **volcengine-ark** — API-key auth, baseUrl `https://ark.cn-beijing.volces.com/api/coding/v3/chat/completions`, format `openai`. Static catalog (Doubao-Seed-2.0, DeepSeek-V4, GLM-5.1, MiniMax-M2.7, Kimi-K2.6). | `open-sse/config/providers.js:166-169`, `open-sse/config/providerModels.js:392-402` | HAVE | w7-prov-openai: `internal/providers/catalog/catalog.go` + `models.go`. Aliases `ark`/`volcengine-ark` pre-existing. 9-model static block. |
| PAR-PROV-039 | **byteplus** — API-key auth, baseUrl `https://ark.ap-southeast.bytepluses.com/api/coding/v3/chat/completions`, format `openai`. Static catalog (seed-2-0-pro, seed-2-0-code-preview, kimi-k2-thinking, glm-4-7, gpt-oss-120b). | `open-sse/config/providers.js:171-174`, `open-sse/config/providerModels.js:429-437` | HAVE | w7-prov-openai: `internal/providers/catalog/catalog.go` + `models.go`. Aliases `byteplus`/`bpm` pre-existing. 7-model static block. |
| PAR-PROV-040 | **commandcode** — API-key auth, baseUrl `https://api.commandcode.ai/alpha/generate`, format `commandcode`, headers `x-command-code-version`, `x-cli-environment`. Static catalog (deepseek-v4-pro, moonshotai/Kimi-K2.6, zai-org/GLM-5.1, MiniMaxAI/MiniMax-M2.7, Qwen/Qwen3.6-Max-Preview). | `open-sse/config/providers.js:261-267`, `open-sse/config/providerModels.js:446-458`, `open-sse/executors/commandcode.js` | MISSING | No g0router equivalent. `CommandCodeExecutor` is specialized. |
| PAR-PROV-041 | **nvidia** — API-key auth, baseUrl `https://integrate.api.nvidia.com/v1/chat/completions`, format `openai`. Static catalog (minimaxai/minimax-m2.7, z-ai/glm4.7, nv-embedqa-e5-v5 embedding, parakeet stt). | `open-sse/config/providers.js:248-250`, `open-sse/config/providerModels.js:513-518` | HAVE | w7-prov-openai: `internal/providers/catalog/catalog.go` + `models.go`. Alias `nvidia` pre-existing. 4-model static block (embedding+stt typed). |
| PAR-PROV-042 | **cerebras** — API-key auth, baseUrl `https://api.cerebras.ai/v1/chat/completions`, format `openai`. Static catalog (gpt-oss-120b, zai-glm-4.7, llama-3.3-70b, llama-4-scout, qwen-3-235b). | `open-sse/config/providers.js:297-299`, `open-sse/config/providerModels.js:500-507` | HAVE | w7-prov-openai: catalog.go + models.go. Alias `cerebras` pre-existing. 6-model block. |
| PAR-PROV-043 | **nebius** — API-key auth, baseUrl `https://api.studio.nebius.ai/v1/chat/completions`, format `openai`. Static catalog (llama-3.3-70b, qwen3-embedding-8b). | `open-sse/config/providers.js:305-307`, `open-sse/config/providerModels.js:520-523` | HAVE | w7-prov-openai: catalog.go + models.go. Alias `nebius` pre-existing. 2-model block (embedding typed). |
| PAR-PROV-044 | **siliconflow** — API-key auth, baseUrl `https://api.siliconflow.cn/v1/chat/completions`, format `openai`. Static catalog (deepseek-v3.2, qwen3-235b, kimi-k2.5, glm-4.7, gpt-oss-120b, ernie-4.5). | `open-sse/config/providers.js:309-311`, `open-sse/config/providerModels.js:533-544` | HAVE | w7-prov-openai: catalog.go + models.go. Alias `siliconflow` pre-existing. 10-model block. |
| PAR-PROV-045 | **hyperbolic** — API-key auth, baseUrl `https://api.hyperbolic.xyz/v1/chat/completions`, format `openai`. Static catalog (qwq-32b, deepseek-r1, deepseek-v3, llama-3.3-70b, hermes-3-70b). | `open-sse/config/providers.js:313-315`, `open-sse/config/providerModels.js:562-571` | HAVE | w7-prov-openai: catalog.go + models.go. Aliases `hyp`/`hyperbolic` pre-existing. 8-model block. |
| PAR-PROV-046 | **xiaomi-mimo** — API-key auth, baseUrl `https://api.xiaomimimo.com/v1/chat/completions`, format `openai`. Static catalog (mimo-v2.5-pro, mimo-v2.5, mimo-v2-omni, mimo-v2-flash). | `open-sse/config/providers.js:394-396`, `open-sse/config/providerModels.js:545-549` | HAVE | w7-prov-openai: catalog.go + models.go. Aliases `mimo`/`xiaomi-mimo` pre-existing. 4-model block. |
| PAR-PROV-047 | **xiaomi-tokenplan** — API-key auth, region-based baseUrl resolution (`sgp`, `cn`, `ams`), format `openai`. Static catalog (mimo-v2.5-pro with claude native variant, tts models, voice clone/design). `XiaomiTokenplanExecutor` handles region routing. | `open-sse/config/providers.js:398-401`, `open-sse/config/providerModels.js:551-561`, `open-sse/config/providers.js:447-457`, `open-sse/executors/xiaomi-tokenplan.js` | MISSING | No g0router equivalent. |
| PAR-PROV-048 | **opencode-go** — API-key auth, baseUrl `https://opencode.ai/zen/go/v1/chat/completions`, format `openai`. Static catalog (kimi-k2.6, glm-5.1, qwen3.5-plus, minimax-m2.7 with targetFormat claude). | `open-sse/config/providers.js:369-372`, `open-sse/config/providerModels.js:195-206` | HAVE | w7-prov-openai: catalog.go + models.go. Aliases `ocg`/`opencode-go` pre-existing. 10-model block. ESC-2: subscription auth/ESC-5: targetFormat deferred. |
| PAR-PROV-049 | **opencode** — No auth (`noAuth: true`), baseUrl `https://opencode.ai`, format `openai`, headers `x-opencode-client: desktop`. Empty static catalog (all models commented out). | `open-sse/config/providers.js:363-367`, `open-sse/config/providerModels.js:207-213` | HAVE | w7-prov-openai: catalog.go (NoAuth+header). No model block (empty ref). Aliases `oc`/`opencode` pre-existing. |
| PAR-PROV-050 | **gitlab** — API-key auth (Bearer token), baseUrl `https://gitlab.com/api/v4/chat/completions`, format `openai`. No static catalog in providerModels. | `open-sse/config/providers.js:354-357` | HAVE | w7-prov-openai: catalog.go. Alias `gitlab` added. No model block (ESC-6). |
| PAR-PROV-051 | **codebuddy** — API-key auth, baseUrl `https://copilot.tencent.com/v1/chat/completions`, format `openai`. Comment says "uses device_code polling auth". No static catalog in providerModels. | `open-sse/config/providers.js:359-361` | HAVE | w7-prov-openai: catalog.go. Alias `codebuddy` added. No model block (ESC-6). Device-code OAuth is ESC-3, deferred to w7-prov-oauth. |
| PAR-PROV-052 | **vercel-ai-gateway** — API-key auth, baseUrl `https://ai-gateway.vercel.sh/v1/chat/completions`, format `openai`. No static catalog in providerModels. | `open-sse/config/providers.js:127-129` | HAVE | w7-prov-openai: catalog.go. Aliases `vercel`/`vercel-ai-gateway` pre-existing. No model block (ESC-6). |
| PAR-PROV-053 | **deepgram** — API-key auth, baseUrl `https://api.deepgram.com/v1/listen`, format `openai` (STT endpoint). Static catalog (nova-3, nova-2, whisper-large). | `open-sse/config/providers.js:317-319`, `open-sse/config/providerModels.js:785-788` | MISSING | No g0router equivalent. STT-only provider. |
| PAR-PROV-054 | **assemblyai** — API-key auth, baseUrl `https://api.assemblyai.com/v1/audio/transcriptions`, format `openai`. Static catalog (universal-3-pro, universal-2). | `open-sse/config/providers.js:321-323`, `open-sse/config/providerModels.js:790-793` | MISSING | No g0router equivalent. STT-only provider. |
| PAR-PROV-055 | **nanobanana** — API-key auth, baseUrl `https://api.nanobananaapi.ai/v1/chat/completions`, format `openai`. Static catalog (nanobanana-flash, nanobanana-pro image models). | `open-sse/config/providers.js:325-327`, `open-sse/config/providerModels.js:620-623` | MISSING | No g0router equivalent. Image provider. |
| PAR-PROV-056 | **chutes** — API-key auth, baseUrl `https://llm.chutes.ai/v1/chat/completions`, format `openai`. No static catalog in providerModels. | `open-sse/config/providers.js:329-331` | HAVE | w7-prov-openai: catalog.go. Aliases `ch`/`chutes` pre-existing. No model block (ESC-6). |
| PAR-PROV-057 | **blackbox** — API-key auth, baseUrl `https://api.blackbox.ai/chat/completions`, format `openai`. Static catalog (gpt-4o, claude-sonnet-4.6, deepseek-chat, o1, gemini-2.5-flash, qwen3-coder-plus). | `open-sse/config/providers.js:437`, `open-sse/config/providerModels.js:348-366` | HAVE | w7-prov-openai: catalog.go + models.go. Aliases `bb`/`blackbox` pre-existing. 17-model block. |
| PAR-PROV-058 | **fal-ai** — API-key auth, image-only. Static catalog (flux-schnell, flux-dev, flux-pro, recraft-v3, ideogram-v2, sd-3.5-large). | `open-sse/config/providerModels.js:794-802` | MISSING | No g0router equivalent. Image provider, no chat endpoint. |
| PAR-PROV-059 | **stability-ai** — API-key auth, image-only. Static catalog (stable-image-ultra, stable-image-core, sd3.5-large, sd3.5-large-turbo, sd3.5-medium). | `open-sse/config/providerModels.js:803-809` | MISSING | No g0router equivalent. Image provider. |
| PAR-PROV-060 | **black-forest-labs** — API-key auth, image-only. Static catalog (flux-pro-1.1, flux-pro-1.1-ultra, flux-dev, flux-kontext-pro/max with edit capability). | `open-sse/config/providerModels.js:810-817` | MISSING | No g0router equivalent. Image provider. |
| PAR-PROV-061 | **recraft** — API-key auth, image-only. Static catalog (recraftv3, recraftv2). | `open-sse/config/providerModels.js:818-821` | MISSING | No g0router equivalent. Image provider. |
| PAR-PROV-062 | **runwayml** — API-key auth, image/video. Static catalog (gen4_image, gen4_image_turbo, gen4_turbo video, gen3a_turbo video). | `open-sse/config/providerModels.js:822-827` | MISSING | No g0router equivalent. Video provider. |
| PAR-PROV-063 | **sdwebui** — API-key auth, image-only. Static catalog (stable-diffusion-v1-5, sdxl-base-1.0). | `open-sse/config/providerModels.js:624-627` | MISSING | No g0router equivalent. Image provider. |
| PAR-PROV-064 | **comfyui** — API-key auth, image-only. Static catalog (flux-dev, sdxl). | `open-sse/config/providerModels.js:628-631` | MISSING | No g0router equivalent. Image provider. |
| PAR-PROV-065 | **huggingface** — API-key auth, image + STT. Static catalog (flux-1-schnell, sd-xl-base, whisper-large-v3, whisper-small). | `open-sse/config/providerModels.js:632-638` | MISSING | No g0router equivalent. Image/STT provider. |
| PAR-PROV-066 | **voyage-ai** — API-key auth, embedding-only. Static catalog (voyage-3-large, voyage-3.5, voyage-code-3, voyage-finance-2, voyage-law-2, voyage-multilingual-2). | `open-sse/config/providerModels.js:524-532` | MISSING | No g0router equivalent. Embedding provider. |
| PAR-PROV-067 | **Free-tier providers (agentrouter, aimlapi, novita, modal, reka, nlpcloud, bazaarlink, completions, enally, freetheai, llm7, lepton, kluster, ai21, inference-net, predibase, bytez, morph, longcat, puter, uncloseai, scaleway, deepinfra, sambanova, nscale, baseten, publicai, nous-research, glhf)** — All API-key or noAuth, all format `openai`, all static catalogs in `providerModels.js`. | `open-sse/config/providers.js:406-437`, `open-sse/config/providerModels.js:641-783` | HAVE | w7-prov-openai: 28 openai providers added to catalog.go + models.go. All aliases pre-existing. `enally` uses AuthHeader `x-api-key`; `uncloseai` NoAuth. ESC-4: `agentrouter` excluded (format:claude, see ESC-1); PAR-PROV-067 HAVE for 28/29 openai providers. |

---

## Data models

### 9router Provider config schema (from `PROVIDERS` object)

```
{
  baseUrl?: string,
  baseUrls?: string[],          // fallback URLs
  format: string,               // "openai" | "claude" | "gemini" | "ollama" | "kiro" | "cursor" | "grok-web" | "perplexity-web" | "commandcode" | "antigravity" | "vertex" | "openai-responses" | "gemini-cli"
  headers?: Record<string,string>,
  clientId?: string,            // OAuth client id
  clientSecret?: string,        // OAuth client secret
  tokenUrl?: string,            // OAuth token endpoint
  authUrl?: string,             // OAuth device-code / auth endpoint
  refreshUrl?: string,          // override refresh endpoint
  authType?: "cookie",          // cookie-based auth (grok-web, perplexity-web)
  noAuth?: boolean,             // no authentication required
  authHeader?: string,          // override auth header name (e.g. "x-api-key")
  retry?: Record<number,number> // status code -> retry count
}
```

Evidence: `open-sse/config/providers.js:50-457`.

### 9router Provider model entry schema (from `PROVIDER_MODELS`)

```
{
  id: string,
  name: string,
  type?: "llm" | "embedding" | "tts" | "stt" | "image" | "video",
  capabilities?: string[],      // e.g. ["text2img", "edit", "mask"]
  params?: string[],            // e.g. ["size", "quality", "style"]
  targetFormat?: string,        // e.g. "claude" for format translation
  upstreamModelId?: string,     // map to upstream model id
  quotaFamily?: string,         // e.g. "review" for Codex review variants
  thinking?: boolean,           // thinking mode override
  strip?: string[]              // content types to strip, e.g. ["image", "audio"]
}
```

Evidence: `open-sse/config/providerModels.js:29-828`.

### 9router OAuth alias map

| Provider | Alias |
|---|---|
| claude | cc |
| codex | cx |
| gemini-cli | gc |
| qwen | qw |
| iflow | if |
| antigravity | ag |
| github | gh |
| kiro | kr |
| cursor | cu |
| kimi-coding | kmc |
| kilocode | kc |
| cline | cl |
| opencode | oc |
| qoder | qd |
| vertex | vertex |
| vertex-partner | vertex-partner |

Evidence: `open-sse/config/providerModels.js:885-900`.

### g0router Provider interface

```go
type Provider interface {
    GetProvider() ModelProvider
    SetNetworkConfig(config NetworkConfig)
    ListModels(...) (*ListModelsResponse, *ProviderError)
    ChatCompletion(...) (*ChatResponse, *ProviderError)
    ChatCompletionStream(...) (chan *StreamChunk, *ProviderError)
    TextCompletion(...) (*TextCompletionResponse, *ProviderError)
    TextCompletionStream(...) (chan *StreamChunk, *ProviderError)
    Responses(...) (*ResponsesResponse, *ProviderError)
    ResponsesStream(...) (chan *StreamChunk, *ProviderError)
    Embedding(...) (*EmbeddingResponse, *ProviderError)
    ImageGeneration(...) (*ImageGenerationResponse, *ProviderError)
    // ... plus Speech, Transcription, File*, Batch*, CountTokens
}
```

Evidence: `internal/schemas/provider.go:68-107`.

---

## Edge cases and quirks

1. **Claude header cache overlay**: `DefaultExecutor.buildHeaders` for `claude` overlays live cached headers from a real Claude Code client over static defaults, merging `Anthropic-Beta` flags. Evidence: `open-sse/executors/default.js:81-110`.

2. **Anthropic-compatible provider header stripping**: For providers starting with `anthropic-compatible-`, first-party Claude Code identity headers (`anthropic-dangerous-direct-browser-access`, `x-app`, `claude-code-20250219` beta flag) are stripped unless the baseUrl is official `api.anthropic.com`. Evidence: `open-sse/executors/default.js:155-180`.

3. **Gemini URL construction**: Gemini chat URL is built as `${baseUrl}/${model}:streamGenerateContent?alt=sse` for streaming, or `:generateContent` for non-streaming. Evidence: `open-sse/executors/default.js:60-61`.

4. **Kimi/minimax claude-format suffix**: Claude-format providers (`claude`, `glm`, `kimi`, `minimax`, `minimax-cn`, `kimi-coding`) append `?beta=true` to the baseUrl. Evidence: `open-sse/executors/default.js:52-59`.

5. **Cloudflare accountId template substitution**: `cloudflare-ai` baseUrl contains `{accountId}`; `DefaultExecutor.buildUrl` substitutes from `credentials.providerSpecificData.accountId` and throws if missing. Evidence: `open-sse/executors/default.js:64-68`.

6. **Ollama NDJSON usage extraction**: `usageTracking.js` handles Ollama's raw NDJSON format (`prompt_eval_count`, `eval_count`) before translation. Evidence: `open-sse/utils/usageTracking.js:225-231`.

7. **Codex review model suffixes**: `withCodexReviewModels` dynamically generates `-review` suffixed variants for every Codex LLM model, with `quotaFamily: "review"`. Evidence: `open-sse/config/providerModels.js:8-27`.

8. **Xiaomi tokenplan region resolution**: `resolveXiaomiTokenplanBaseUrl` maps regions (`sgp`, `cn`, `ams`) to cluster-specific URLs. Evidence: `open-sse/config/providers.js:447-457`.

9. **Opencode duplicate key overwrite**: `providers.js` defines `opencode` twice (lines 233 and 363); the second definition (`noAuth: true`, `baseUrl: https://opencode.ai`) overwrites the first (`localhost:4096`). Evidence: `open-sse/config/providers.js:233-237`, `363-367`.

10. **Kiro AWS eventstream**: Kiro uses `Accept: application/vnd.amazon.eventstream` and `X-Amz-Target` headers, not JSON/SSE. Evidence: `open-sse/config/providers.js:198-204`.

11. **Cursor connect+proto**: Cursor uses `Content-Type: application/connect+proto` and a gRPC-style path `/aiserver.v1.ChatService/StreamUnifiedChatWithTools`. Evidence: `open-sse/config/providers.js:210-217`.

12. **Usage buffer tokens**: All usage objects get `BUFFER_TOKENS = 2000` added to input/prompt and total tokens to prevent context errors. Evidence: `open-sse/utils/usageTracking.js:19`, `31-55`.

13. **Connect timeout abort**: `BaseExecutor.execute` creates an internal `AbortController` for connection timeouts (default 30s) merged with caller signal via `AbortSignal.any`. Evidence: `open-sse/executors/base.js:125-128`.

14. **Retry by status code**: `BaseExecutor` supports per-provider `retry` config mapping status codes to `{attempts, delayMs}`. `kiro` explicitly sets `retry: { 429: 2 }`. Evidence: `open-sse/executors/base.js:104-115`, `open-sse/config/providers.js:197`.

15. **No quota fetcher code found**: 9router tracks usage via `saveRequestUsage` / `appendRequestLog` but has no explicit upstream quota/limit fetchers for any provider. Evidence: `open-sse/utils/usageTracking.js:338-346`.

---

## Go-port considerations

1. g0router's `Provider` interface is large (~20 methods); most 9router providers only implement chat + streaming. Stubs are acceptable for Stage 1.
2. 9router's `DefaultExecutor` covers ~80% of providers via OpenAI-compatible passthrough. A generic Go executor with format-specific request/response translators would collapse many adapters into one struct.
3. OAuth flows (Claude, Codex, Gemini, GitHub, Kiro, Cursor, etc.) require device-code polling, refresh token storage, and PKCE. Port these only after API-key providers are stable.
4. Cookie-auth providers (grok-web, perplexity-web) are reverse-engineered and fragile; defer until after GA.
5. Specialized formats (Cursor protobuf, Kiro AWS eventstream, CommandCode custom JSON) need dedicated converter packages; they are high-effort, low-ROI for Stage 1.
6. Image/STT/TTS-only providers map to different g0router interface methods; they can be added incrementally after chat providers are complete.

---

## Stage 1 Go-port ranking

### Include now (high code maturity + high uniqueness)

| Rank | Provider | Rationale |
|---|---|---|
| 1 | **deepseek** | Pure OpenAI format, API-key auth, massive user demand. Trivial adapter. |
| 2 | **groq** | Pure OpenAI format, API-key auth, fast inference niche. Already has STT models in catalog. |
| 3 | **mistral** | Pure OpenAI format, API-key auth, major European provider. |
| 4 | **together** | Pure OpenAI format, API-key auth, open-model hub. |
| 5 | **fireworks** | Pure OpenAI format, API-key auth, enterprise inference. |
| 6 | **cohere** | Pure OpenAI format, API-key auth, Command R models. |
| 7 | **xai** | OpenAI format, has OAuth but also API-key path (grok models). High visibility. |
| 8 | **openrouter** | OpenAI format, API-key auth, gateway to 100+ models. g0router schema constant exists but no dir. |
| 9 | **perplexity** | OpenAI format, API-key auth, search-augmented LLM niche. |
| 10 | **ollama** | Local deployment, no auth, ollama-native format (not OpenAI). High demand for on-premise. |

### Defer to Stage 2+ (OAuth complexity or niche value)

- **OAuth providers**: claude, codex, gemini-cli, qwen, iflow, antigravity, github, kiro, cursor, kimi-coding, cline, kilocode, xai (if OAuth path required).
- **Custom format / reverse-engineered**: cursor, kiro, commandcode, qoder, grok-web, perplexity-web, azure.
- **GCP / enterprise**: vertex, vertex-partner, cloudflare-ai, bedrock.
- **Chinese ecosystems**: glm, glm-cn, kimi, alicode, alicode-intl, volcengine-ark, byteplus, xiaomi-mimo, xiaomi-tokenplan, siliconflow.
- **Image / video / STT / TTS / embedding specialists**: fal-ai, stability-ai, black-forest-labs, recraft, runwayml, sdwebui, comfyui, huggingface, deepgram, assemblyai, nanobanana, voyage-ai, nvidia (embed/stt).
- **Free-tier / experimental**: All 29 OmniRoute free-tier providers (agentrouter, aimlapi, novita, modal, reka, nlpcloud, bazaarlink, completions, enally, freetheai, llm7, lepton, kluster, ai21, inference-net, predibase, bytez, morph, longcat, puter, uncloseai, scaleway, deepinfra, sambanova, nscale, baseten, publicai, nous-research, glhf, blackbox, chutes).
- **No-op / stub**: opencode (noAuth, empty catalog), opencode-go, gitlab, codebuddy, vercel-ai-gateway.

---

ANALYSIS-COMPLETE /Users/heitor/Developer/github.com/bloodf/g0router/.planning/parity/matrix/9router-providers.md
