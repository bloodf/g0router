package api

import (
	"strconv"
	"strings"

	"github.com/bloodf/g0router/api/handlers"
	"github.com/bloodf/g0router/internal/mcp"
	"github.com/valyala/fasthttp"
)

type route struct {
	method  string
	pattern string
	match   func(rawPath, method string) bool
	handler func(*fasthttp.RequestCtx)
}

func exactMatch(path string) func(rawPath, method string) bool {
	return func(rawPath, method string) bool {
		return rawPath == path
	}
}

func exactMatchWithMethod(path, expectedMethod string) func(rawPath, method string) bool {
	return func(rawPath, method string) bool {
		return rawPath == path && method == expectedMethod
	}
}

func apiExactMatch(path string) func(rawPath, method string) bool {
	return func(rawPath, method string) bool {
		return strings.TrimRight(rawPath, "/") == path
	}
}

func apiPathMatch(check func(parts []string) bool) func(rawPath, method string) bool {
	return func(rawPath, method string) bool {
		parts := pathParts(strings.TrimRight(rawPath, "/"))
		return check(parts)
	}
}

func (s *Server) withAudit(handler func(*fasthttp.RequestCtx)) func(*fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		handler(ctx)
		s.recordAuditIfMutation(ctx)
	}
}

func (s *Server) withClientIP(handler func(*fasthttp.RequestCtx)) func(*fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.SetUserValue("g0router.client_ip", s.clientIP(ctx))
		handler(ctx)
	}
}

func (s *Server) routes() []route {
	return []route{
		{method: "GET", pattern: "/healthz", match: exactMatch("/healthz"), handler: func(ctx *fasthttp.RequestCtx) {
			handlers.Health(ctx, s.config.Version)
		}},
		{method: "GET", pattern: "/metrics", match: exactMatch("/metrics"), handler: s.handleMetrics},
		{method: "POST", pattern: "/v1/chat/completions", match: exactMatchWithMethod("/v1/chat/completions", fasthttp.MethodPost), handler: s.handleInference},
		{method: "POST", pattern: "/v1/messages", match: exactMatchWithMethod("/v1/messages", fasthttp.MethodPost), handler: s.handleMessages},
		{method: "POST", pattern: "/v1/responses", match: exactMatchWithMethod("/v1/responses", fasthttp.MethodPost), handler: s.handleResponses},
		{method: "POST", pattern: "/v1/embeddings", match: exactMatch("/v1/embeddings"), handler: func(ctx *fasthttp.RequestCtx) {
			s.handleExtra(ctx, handlers.Embeddings)
		}},
		{method: "POST", pattern: "/v1/images/generations", match: exactMatch("/v1/images/generations"), handler: func(ctx *fasthttp.RequestCtx) {
			s.handleExtra(ctx, handlers.Images)
		}},
		{method: "POST", pattern: "/v1/audio/transcriptions", match: exactMatch("/v1/audio/transcriptions"), handler: func(ctx *fasthttp.RequestCtx) {
			s.handleExtra(ctx, handlers.AudioTranscription)
		}},
		{method: "POST", pattern: "/v1/audio/speech", match: exactMatch("/v1/audio/speech"), handler: func(ctx *fasthttp.RequestCtx) {
			s.handleExtra(ctx, handlers.Speech)
		}},
		{method: "GET", pattern: "/v1/models", match: exactMatchWithMethod("/v1/models", fasthttp.MethodGet), handler: func(ctx *fasthttp.RequestCtx) {
			handlers.Models(ctx, s.config.InferenceEngine)
		}},

		// API routes
		{method: "", pattern: "/api/providers", match: apiExactMatch("/api/providers"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			handlers.Providers(ctx, s.config.ModelSource, "")
		})},
		{method: "", pattern: "/api/providers/:provider/models", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 4 && parts[0] == "api" && parts[1] == "providers" && parts[3] == "models"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.Providers(ctx, s.config.ModelSource, parts[2])
		})},
		{method: "POST", pattern: "/api/providers/test-batch", match: apiExactMatch("/api/providers/test-batch"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodPost) {
				return
			}
			handlers.ModelTestBatch(ctx, s.config.Store, s.config.ProviderAdapterSource)
		})},
		{method: "", pattern: "/api/providers/:id/models/:model/test", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 6 && parts[0] == "api" && parts[1] == "providers" && parts[3] == "models" && parts[5] == "test"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodPost) {
				return
			}
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.ModelTest(ctx, s.config.Store, s.config.ProviderAdapterSource, parts[2], parts[4])
		})},
		{method: "", pattern: "/api/providers/:id/connections", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 4 && parts[0] == "api" && parts[1] == "providers" && parts[3] == "connections"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodGet) {
				return
			}
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.ProviderConnections(ctx, s.config.Store, parts[2])
		})},
		{method: "", pattern: "/api/providers/:id/suggested-models", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 4 && parts[0] == "api" && parts[1] == "providers" && parts[3] == "suggested-models"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodGet) {
				return
			}
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.ProviderSuggestedModels(ctx, s.config.Store, s.config.ProviderAdapterSource, parts[2])
		})},
		{method: "", pattern: "/api/providers/:id", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 3 && parts[0] == "api" && parts[1] == "providers"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodGet) {
				return
			}
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.ProviderDetail(ctx, s.config.Store, s.config.ModelSource, parts[2])
		})},
		{method: "", pattern: "/api/connections", match: apiExactMatch("/api/connections"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			handlers.Connections(ctx, s.config.Store, "")
		})},
		{method: "", pattern: "/api/connections/:id", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 3 && parts[0] == "api" && parts[1] == "connections"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.Connections(ctx, s.config.Store, parts[2])
		})},
		{method: "", pattern: "/api/connections/:id/test", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 4 && parts[0] == "api" && parts[1] == "connections" && parts[3] == "test"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.ConnectionTest(ctx, s.config.Store, parts[2])
		})},
		{method: "", pattern: "/api/settings", match: apiExactMatch("/api/settings"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			handlers.Settings(ctx, s.config.Store)
		})},
		{method: "", pattern: "/api/keys", match: apiExactMatch("/api/keys"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			handlers.APIKeys(ctx, s.config.Store, s.config.APIKeySecret, "")
		})},
		{method: "", pattern: "/api/keys/:id", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 3 && parts[0] == "api" && parts[1] == "keys"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.APIKeys(ctx, s.config.Store, s.config.APIKeySecret, parts[2])
		})},
		{method: "", pattern: "/api/combos", match: apiExactMatch("/api/combos"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			handlers.Combos(ctx, s.config.Store, "")
		})},
		{method: "", pattern: "/api/combos/:id", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 3 && parts[0] == "api" && parts[1] == "combos"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.Combos(ctx, s.config.Store, parts[2])
		})},
		{method: "", pattern: "/api/aliases", match: apiExactMatch("/api/aliases"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			handlers.Aliases(ctx, s.config.Store, "")
		})},
		{method: "", pattern: "/api/aliases/:id", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 3 && parts[0] == "api" && parts[1] == "aliases"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.Aliases(ctx, s.config.Store, parts[2])
		})},
		{method: "", pattern: "/api/pricing", match: apiExactMatch("/api/pricing"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			handlers.Pricing(ctx, s.config.Store, "", "")
		})},
		{method: "", pattern: "/api/pricing/:provider/:model", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 4 && parts[0] == "api" && parts[1] == "pricing"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.Pricing(ctx, s.config.Store, parts[2], parts[3])
		})},
		{method: "", pattern: "/api/models/disabled", match: apiExactMatch("/api/models/disabled"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			switch string(ctx.Method()) {
			case fasthttp.MethodGet:
				handlers.DisabledModelsList(ctx, s.config.Store)
			case fasthttp.MethodPost:
				handlers.DisabledModelsCreate(ctx, s.config.Store, s.config.Store)
			case fasthttp.MethodDelete:
				handlers.DisabledModelsDelete(ctx, s.config.Store, s.config.Store)
			default:
				ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			}
		})},
		{method: "", pattern: "/api/models/custom", match: apiExactMatch("/api/models/custom"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			switch string(ctx.Method()) {
			case fasthttp.MethodGet:
				handlers.CustomModelsList(ctx, s.config.Store)
			case fasthttp.MethodPost:
				handlers.CustomModelsCreate(ctx, s.config.Store, s.config.Store)
			default:
				ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			}
		})},
		{method: "", pattern: "/api/models/custom/:id", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 4 && parts[0] == "api" && parts[1] == "models" && parts[2] == "custom"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodDelete) {
				return
			}
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.CustomModelsDelete(ctx, s.config.Store, s.config.Store, parts[3])
		})},
		{method: "GET", pattern: "/api/oauth/callback", match: apiExactMatch("/api/oauth/callback"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodGet) {
				return
			}
			handlers.OAuthCallback(ctx, s.config.Store, s.config.OAuthFlows)
		})},
		{method: "POST", pattern: "/api/oauth/:provider/authorize", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 4 && parts[0] == "api" && parts[1] == "oauth" && parts[3] == "authorize"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodPost) {
				return
			}
			handlers.OAuthStart(ctx, s.config.Store, s.config.OAuthFlows)
		})},
		{method: "GET", pattern: "/api/oauth/:provider/poll", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 4 && parts[0] == "api" && parts[1] == "oauth" && parts[3] == "poll"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodGet) {
				return
			}
			handlers.OAuthPoll(ctx, s.config.Store, s.config.OAuthFlows)
		})},
		{method: "POST", pattern: "/api/oauth/:provider/exchange", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 4 && parts[0] == "api" && parts[1] == "oauth" && parts[3] == "exchange"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodPost) {
				return
			}
			handlers.OAuthExchange(ctx, s.config.Store, s.config.OAuthFlows)
		})},
		{method: "", pattern: "/api/usage", match: apiExactMatch("/api/usage"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			handlers.Usage(ctx, s.config.UsageStore)
		})},
		{method: "", pattern: "/api/usage/summary", match: apiExactMatch("/api/usage/summary"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			handlers.UsageSummary(ctx, s.config.UsageStore)
		})},
		{method: "", pattern: "/api/usage/quota/*", match: func(rawPath, method string) bool {
			return strings.HasPrefix(strings.TrimRight(rawPath, "/"), "/api/usage/quota/")
		}, handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			handlers.UsageQuota(ctx, s.config.Store, s.config.QuotaFetchers, s.config.QuotaKey)
		})},
		{method: "", pattern: "/api/logs", match: apiExactMatch("/api/logs"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			handlers.Logs(ctx, s.config.UsageStore)
		})},
		{method: "", pattern: "/api/audit", match: apiExactMatch("/api/audit"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			handlers.Audit(ctx, s.config.Store)
		})},
		{method: "", pattern: "/api/traffic/stream", match: apiExactMatch("/api/traffic/stream"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			s.handleTrafficStream(ctx)
		})},
		{method: "", pattern: "/api/mcp/clients", match: apiExactMatch("/api/mcp/clients"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			handlers.MCPClients(ctx, s.config.Store, s.config.MCPClientManager, s.config.MCPToolManager, "")
		})},
		{method: "", pattern: "/api/mcp/clients/:id", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 4 && parts[0] == "api" && parts[1] == "mcp" && parts[2] == "clients"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.MCPClients(ctx, s.config.Store, s.config.MCPClientManager, s.config.MCPToolManager, parts[3])
		})},
		{method: "", pattern: "/api/mcp/instances", match: apiExactMatch("/api/mcp/instances"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			handlers.MCPInstances(ctx, s.config.Store, s.config.MCPInstanceRuntime, "")
		})},
		{method: "", pattern: "/api/mcp/instances/:id", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 4 && parts[0] == "api" && parts[1] == "mcp" && parts[2] == "instances"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.MCPInstances(ctx, s.config.Store, s.config.MCPInstanceRuntime, parts[3])
		})},
		{method: "POST", pattern: "/api/mcp/instances/:id/auth/start", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 6 && parts[0] == "api" && parts[1] == "mcp" && parts[2] == "instances" && parts[4] == "auth" && parts[5] == "start"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodPost) {
				return
			}
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.MCPOAuthStart(ctx, s.config.Store, parts[3])
		})},
		{method: "GET", pattern: "/api/mcp/instances/:id/accounts", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 5 && parts[0] == "api" && parts[1] == "mcp" && parts[2] == "instances" && parts[4] == "accounts"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodGet) {
				return
			}
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.MCPOAuthAccounts(ctx, s.config.Store, parts[3])
		})},
		{method: "", pattern: "/api/mcp/tools", match: apiExactMatch("/api/mcp/tools"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			handlers.MCPTools(ctx, s.config.Store, s.config.MCPToolManager, "")
		})},
		{method: "", pattern: "/api/mcp/tools/:id/execute", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 5 && parts[0] == "api" && parts[1] == "mcp" && parts[2] == "tools" && parts[4] == "execute"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.MCPTools(ctx, s.config.Store, s.config.MCPToolManager, parts[3])
		})},
		{method: "GET", pattern: "/api/mcp/oauth/callback", match: apiExactMatch("/api/mcp/oauth/callback"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodGet) {
				return
			}
			handlers.MCPOAuthCallback(ctx, mcp.NewOAuthEngine(s.config.Store, nil), s.config.MCPInstanceRuntime, s.config.Store)
		})},
		{method: "POST", pattern: "/api/mcp/instances/:id/oauth/complete", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 6 && parts[0] == "api" && parts[1] == "mcp" && parts[2] == "instances" && parts[4] == "oauth" && parts[5] == "complete"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodPost) {
				return
			}
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.MCPOAuthComplete(ctx, mcp.NewOAuthEngine(s.config.Store, nil), s.config.MCPInstanceRuntime, s.config.Store, parts[3])
		})},

		// Proxy pool routes
		{method: "", pattern: "/api/proxy-pools", match: apiExactMatch("/api/proxy-pools"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			switch string(ctx.Method()) {
			case fasthttp.MethodGet:
				handlers.ProxyPoolList(ctx, s.config.Store)
			case fasthttp.MethodPost:
				handlers.ProxyPoolCreate(ctx, s.config.Store, s.config.Store)
			default:
				ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			}
		})},
		{method: "POST", pattern: "/api/proxy-pools/batch", match: apiExactMatch("/api/proxy-pools/batch"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodPost) {
				return
			}
			handlers.ProxyPoolBatchImport(ctx, s.config.Store, s.config.Store)
		})},
		{method: "", pattern: "/api/proxy-pools/:id", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 3 && parts[0] == "api" && parts[1] == "proxy-pools"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			switch string(ctx.Method()) {
			case fasthttp.MethodGet:
				handlers.ProxyPoolGet(ctx, s.config.Store, parts[2])
			case fasthttp.MethodPut:
				handlers.ProxyPoolUpdate(ctx, s.config.Store, s.config.Store, parts[2])
			case fasthttp.MethodDelete:
				handlers.ProxyPoolDelete(ctx, s.config.Store, s.config.Store, parts[2])
			default:
				ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			}
		})},
		{method: "POST", pattern: "/api/proxy-pools/:id/test", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 4 && parts[0] == "api" && parts[1] == "proxy-pools" && parts[3] == "test"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodPost) {
				return
			}
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.ProxyPoolTest(ctx, s.config.Store, parts[2])
		})},

		// Auth routes
		{method: "POST", pattern: "/api/auth/setup", match: apiExactMatch("/api/auth/setup"), handler: s.withAudit(s.withClientIP(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodPost) {
				return
			}
			handlers.AuthSetup(ctx, s.config.Store, s.config.Store, s.config.Store)
		}))},
		{method: "POST", pattern: "/api/auth/login", match: apiExactMatch("/api/auth/login"), handler: s.withAudit(s.withClientIP(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodPost) {
				return
			}
			handlers.AuthLogin(ctx, s.config.Store, s.config.Store, s.loginRateLimiter, s.config.Store)
		}))},
		{method: "POST", pattern: "/api/auth/logout", match: apiExactMatch("/api/auth/logout"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodPost) {
				return
			}
			handlers.AuthLogout(ctx, s.config.Store)
		})},
		{method: "GET", pattern: "/api/auth/status", match: apiExactMatch("/api/auth/status"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodGet) {
				return
			}
			handlers.AuthStatus(ctx, s.config.Store, s.config.Store, s.runtimeSettings().RequireLogin)
		})},
		{method: "PUT", pattern: "/api/auth/password", match: apiExactMatch("/api/auth/password"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodPut) {
				return
			}
			handlers.AuthPasswordChange(ctx, s.config.Store, s.config.Store, s.config.Store)
		})},
		{method: "GET", pattern: "/api/auth/users", match: apiExactMatch("/api/auth/users"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodGet) {
				return
			}
			handlers.AuthUsersList(ctx, s.config.Store)
		})},
		{method: "POST", pattern: "/api/auth/users", match: apiExactMatch("/api/auth/users"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodPost) {
				return
			}
			handlers.AuthUsersCreate(ctx, s.config.Store, s.config.Store)
		})},
		{method: "", pattern: "/api/auth/users/:id", match: apiPathMatch(func(parts []string) bool {
			return len(parts) == 4 && parts[0] == "api" && parts[1] == "auth" && parts[2] == "users"
		}), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodDelete) {
				return
			}
			parts := pathParts(strings.TrimRight(string(ctx.Path()), "/"))
			handlers.AuthUsersDelete(ctx, s.config.Store, s.config.Store, s.config.Store, parts[3])
		})},

		// Tunnel routes
		{method: "GET", pattern: "/api/tunnels", match: apiExactMatch("/api/tunnels"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodGet) {
				return
			}
			handlers.TunnelList(ctx, s.config.Store)
		})},
		{method: "POST", pattern: "/api/tunnels/cloudflare", match: apiExactMatch("/api/tunnels/cloudflare"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodPost) {
				return
			}
			handlers.TunnelCloudflareCreate(ctx, s.config.Store, s.config.TunnelManager, s.config.Store, strconv.Itoa(s.config.Port))
		})},
		{method: "DELETE", pattern: "/api/tunnels/cloudflare", match: apiExactMatch("/api/tunnels/cloudflare"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodDelete) {
				return
			}
			handlers.TunnelCloudflareDelete(ctx, s.config.Store, s.config.TunnelManager, s.config.Store)
		})},
		{method: "POST", pattern: "/api/tunnels/tailscale", match: apiExactMatch("/api/tunnels/tailscale"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodPost) {
				return
			}
			handlers.TunnelTailscaleCreate(ctx, s.config.Store, s.config.TunnelManager, s.config.Store, strconv.Itoa(s.config.Port))
		})},
		{method: "DELETE", pattern: "/api/tunnels/tailscale", match: apiExactMatch("/api/tunnels/tailscale"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodDelete) {
				return
			}
			handlers.TunnelTailscaleDelete(ctx, s.config.Store, s.config.TunnelManager, s.config.Store)
		})},
		{method: "GET", pattern: "/api/tunnels/health", match: apiExactMatch("/api/tunnels/health"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodGet) {
				return
			}
			handlers.TunnelHealth(ctx, s.config.Store)
		})},
		{method: "POST", pattern: "/api/settings/proxy-test", match: apiExactMatch("/api/settings/proxy-test"), handler: s.withAudit(func(ctx *fasthttp.RequestCtx) {
			if !requireMethod(ctx, fasthttp.MethodPost) {
				return
			}
			handlers.ProxyTest(ctx)
		})},

		// catch-all
		{method: "", pattern: "/*", match: func(rawPath, method string) bool { return true }, handler: func(ctx *fasthttp.RequestCtx) {
			path := string(ctx.Path())
			if strings.HasPrefix(path, "/v1/") {
				ctx.SetStatusCode(fasthttp.StatusNotFound)
				return
			}
			trimmed := strings.TrimRight(path, "/")
			if strings.HasPrefix(path, "/api/") || trimmed == "/api" {
				ctx.SetStatusCode(fasthttp.StatusNotFound)
				s.recordAuditIfMutation(ctx)
				return
			}
			s.handleUI(ctx)
			s.recordAuditIfMutation(ctx)
		}},
	}
}
