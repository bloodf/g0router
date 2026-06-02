# Source References

Reference repos cloned to `/tmp/` for analysis. Clone them locally before working:

```bash
git clone --depth=1 https://github.com/decolua/9router.git /tmp/9router-ref
git clone --depth=1 https://github.com/maximhq/bifrost.git /tmp/bifrost-ref
git clone --depth=1 https://github.com/can1357/oh-my-pi.git /tmp/ohpi-ref
```

## bifrost → g0router Mapping

### Core Types (adapt, simplify)
| bifrost file | g0router target | What to extract |
|---|---|---|
| `core/schemas/bifrost.go` | `internal/providers/types.go` | ModelProvider enum (23 values), RequestType |
| `core/schemas/chatcompletions.go` (71KB) | `internal/providers/types.go` | BifrostChatRequest, BifrostChatResponse — strip batch/file/video fields |
| `core/schemas/responses.go` (113KB) | `internal/providers/types.go` | BifrostResponsesRequest/Response — strip non-LLM fields |
| `core/schemas/context.go` | `internal/proxy/context.go` | Request context with mutable values |
| `core/schemas/streaming.go` | `internal/streaming/types.go` | StreamChunk, StreamDelta |

### Provider Implementations (adapt)
| bifrost file | g0router target | Notes |
|---|---|---|
| `core/providers/openai/openai.go` (254KB) | `internal/providers/openai/openai.go` | Extract chat.go + responses.go only (~50KB relevant) |
| `core/providers/openai/chat.go` (7KB) | `internal/providers/openai/chat.go` | Direct adaptation |
| `core/providers/openai/responses.go` (17KB) | `internal/providers/openai/responses.go` | Direct adaptation |
| `core/providers/openai/types.go` (40KB) | `internal/providers/openai/types.go` | Strip non-chat types |
| `core/providers/openai/errors.go` (1.5KB) | `internal/providers/openai/errors.go` | Direct copy |
| `core/providers/anthropic/` | `internal/providers/anthropic/` | Same pattern as OpenAI |
| `core/providers/gemini/` | `internal/providers/gemini/` | Gemini-specific wire format |
| `core/providers/utils/utils.go` (108KB) | `internal/providers/utils/` | HTTP client, SSE parser (~200 lines core) |

### MCP Gateway (adapt)
| bifrost file | g0router target | Notes |
|---|---|---|
| `core/mcp/agent.go` (23KB) | `internal/mcp/agent.go` | Agent loop — remove plugin hooks |
| `core/mcp/agentadaptors.go` (22KB) | `internal/mcp/agent.go` | Merge into agent.go |
| `core/mcp/clientmanager.go` (75KB) | `internal/mcp/clientmanager.go` | Simplify — remove enterprise auth modes |
| `core/mcp/toolmanager.go` (32KB) | `internal/mcp/toolmanager.go` | Keep tool registration + execution |
| `core/mcp/healthmonitor.go` (10KB) | `internal/mcp/healthmonitor.go` | Direct adaptation |
| `core/mcp/discovery.go` | NEW | Compact manifest protocol (not in bifrost) |

### Streaming (adapt)
| bifrost file | g0router target |
|---|---|
| `framework/streaming/accumulator.go` | `internal/streaming/accumulator.go` |
| `framework/streaming/chat.go` | `internal/streaming/chat.go` |
| `framework/streaming/responses.go` | `internal/streaming/responses.go` |

### Pricing (adapt)
| bifrost file | g0router target |
|---|---|
| `framework/modelcatalog/pricing.go` (62KB) | `internal/modelcatalog/pricing.go` |

### HTTP Transport (adapt)
| bifrost file | g0router target |
|---|---|
| `transports/bifrost-http/server/server.go` (72KB) | `api/server.go` (simplified) |
| `transports/bifrost-http/handlers/inference.go` | `api/handlers/inference.go` |
| `transports/bifrost-http/integrations/openai.go` (115KB) | `api/integrations/openai.go` (simplified) |

---

## 9router → g0router Mapping

### Provider Config (port JS → Go)
| 9router file | g0router target | What to port |
|---|---|---|
| `open-sse/config/providers.js` (458 lines) | `internal/config/defaults.go` | Provider baseURLs, formats, headers, client IDs, token URLs |
| `open-sse/config/providerModels.js` (909 lines) | `internal/modelcatalog/catalog.go` | Model lists per provider alias |
| `open-sse/config/errorConfig.js` | `internal/provider/fallback.go` | Error rules, backoff config |
| `open-sse/config/appConstants.js` | `internal/config/defaults.go` | OAuth endpoints, client metadata |

### Auth (port JS → Go)
| 9router file | g0router target |
|---|---|
| `src/sse/services/auth.js` (309 lines) | `internal/provider/connection.go` | Credential selection, round-robin, mutex |
| `open-sse/services/tokenRefresh.js` (877 lines) | `internal/provider/refresh.go` | Per-provider refresh, dedup cache |
| `open-sse/services/accountFallback.js` (216 lines) | `internal/provider/fallback.go` | Backoff, cooldown, model locks |

### RTK (port JS → Go)
| 9router file | g0router target |
|---|---|
| `open-sse/rtk/autodetect.js` (111 lines) | `internal/rtk/autodetect.go` |
| `open-sse/rtk/index.js` (31 lines) | `internal/rtk/rtk.go` |
| `open-sse/rtk/caveman.js` (24 lines) | `internal/rtk/caveman.go` |
| `open-sse/rtk/constants.js` (56 lines) | `internal/rtk/constants.go` |
| `open-sse/rtk/filters/gitDiff.js` | `internal/rtk/filters/gitdiff.go` |
| `open-sse/rtk/filters/gitStatus.js` | `internal/rtk/filters/gitstatus.go` |
| `open-sse/rtk/filters/grep.js` | `internal/rtk/filters/grep.go` |
| `open-sse/rtk/filters/find.js` | `internal/rtk/filters/find.go` |
| `open-sse/rtk/filters/ls.js` | `internal/rtk/filters/ls.go` |
| `open-sse/rtk/filters/tree.js` | `internal/rtk/filters/tree.go` |
| `open-sse/rtk/filters/buildOutput.js` | `internal/rtk/filters/buildoutput.go` |
| `open-sse/rtk/filters/dedupLog.js` | `internal/rtk/filters/deduplog.go` |
| `open-sse/rtk/filters/smartTruncate.js` | `internal/rtk/filters/smarttruncate.go` |
| `open-sse/rtk/filters/readNumbered.js` | `internal/rtk/filters/readnumbered.go` |
| `open-sse/rtk/filters/searchList.js` | `internal/rtk/filters/searchlist.go` |

### Usage (port JS → Go)
| 9router file | g0router target |
|---|---|
| `open-sse/utils/usageTracking.js` (347 lines) | `internal/usage/tracker.go` |
| `open-sse/services/usage.js` (1216 lines) | `internal/usage/quota.go` |

### Translator (port JS → Go)
| 9router file | g0router target |
|---|---|
| `open-sse/translator/request/openai-to-claude.js` | `internal/translate/anthropic.go` |
| `open-sse/translator/response/claude-to-openai.js` | `internal/translate/anthropic.go` |
| `open-sse/translator/request/openai-to-gemini.js` | `internal/translate/gemini.go` |
| `open-sse/translator/response/gemini-to-openai.js` | `internal/translate/gemini.go` |
| `open-sse/translator/request/openai-responses.js` | `internal/translate/responses.go` |
| `open-sse/translator/response/openai-responses.js` | `internal/translate/responses.go` |

---

## oh-my-pi → g0router Mapping

### OAuth Flows (port TS → Go)
| oh-my-pi file | g0router target | Auth type |
|---|---|---|
| `packages/ai/src/utils/oauth/anthropic.ts` (274 lines) | `internal/provider/oauth/anthropic.go` | PKCE + callback |
| `packages/ai/src/utils/oauth/openai-codex.ts` (256 lines) | `internal/provider/oauth/codex.go` | Device-code + callback |
| `packages/ai/src/utils/oauth/github-copilot.ts` (287 lines) | `internal/provider/oauth/github.go` | Device-code |
| `packages/ai/src/utils/oauth/cursor.ts` (157 lines) | `internal/provider/oauth/cursor.go` | PKCE + polling |
| `packages/ai/src/utils/oauth/google-gemini-cli.ts` (198 lines) | `internal/provider/oauth/gemini.go` | OAuth2 + callback |
| `packages/ai/src/utils/oauth/google-antigravity.ts` (168 lines) | `internal/provider/oauth/antigravity.go` | OAuth2 + callback |
| `packages/ai/src/utils/oauth/xai-oauth.ts` (315 lines) | `internal/provider/oauth/xai.go` | OAuth2 |
| `packages/ai/src/utils/oauth/deepseek.ts` (48 lines) | `internal/provider/oauth/deepseek.go` | Password login |
| `packages/ai/src/utils/oauth/gitlab-duo.ts` (97 lines) | `internal/provider/oauth/gitlab.go` | OAuth2 |
| `packages/ai/src/utils/oauth/kimi.ts` (201 lines) | `internal/provider/oauth/kimi.go` | Device-code |
| `packages/ai/src/utils/oauth/minimax-code.ts` (64 lines) | `internal/provider/oauth/minimax.go` | API key |
| `packages/ai/src/utils/oauth/alibaba-coding-plan.ts` (46 lines) | `internal/provider/oauth/alibaba.go` | API key |
| `packages/ai/src/utils/oauth/zhipu.ts` (46 lines) | `internal/provider/oauth/zhipu.go` | API key |
| `packages/ai/src/utils/oauth/xiaomi.ts` (127 lines) | `internal/provider/oauth/xiaomi.go` | OAuth2 |
| `packages/ai/src/utils/oauth/perplexity.ts` (181 lines) | `internal/provider/oauth/perplexity.go` | OAuth2 |

### Credential Storage (pattern reference)
| oh-my-pi file | g0router target | What to learn |
|---|---|---|
| `packages/ai/src/auth-storage.ts` (4418 lines) | `internal/store/connections.go` | SQLite schema, round-robin, credential health |
| `packages/ai/src/auth-gateway/server.ts` (819 lines) | `api/handlers/inference.go` | Format routing, credential injection |

### Usage/Quota (port TS → Go)
| oh-my-pi file | g0router target |
|---|---|
| `packages/ai/src/usage/claude.ts` (400 lines) | `internal/usage/quota.go` |
| `packages/ai/src/usage/openai-codex.ts` (408 lines) | `internal/usage/quota.go` |
| `packages/ai/src/usage/github-copilot.ts` (336 lines) | `internal/usage/quota.go` |
| `packages/ai/src/usage/gemini.ts` (180 lines) | `internal/usage/quota.go` |
| `packages/ai/src/usage/google-antigravity.ts` (158 lines) | `internal/usage/quota.go` |
| `packages/ai/src/usage/kimi.ts` (228 lines) | `internal/usage/quota.go` |
| `packages/ai/src/usage/zai.ts` (174 lines) | `internal/usage/quota.go` |
| `packages/ai/src/usage/minimax-code.ts` (35 lines) | `internal/usage/quota.go` |
