package api

import (
	"encoding/json"
	"path"
	"strconv"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/usage"
	"github.com/valyala/fasthttp"
)

// enforceKeyPolicy applies per-key scope, RPM, and daily-spend checks before an
// inference request is dispatched. It returns true when the request may
// proceed; otherwise it writes the appropriate status (403 / 429 / 402) and
// returns false. Requests without a key policy (no authenticated key) pass
// through unchanged. All ctx reads happen here on the request goroutine.
func (s *Server) enforceKeyPolicy(ctx *fasthttp.RequestCtx) bool {
	identity, ok := ctx.UserValue(requestAPIKeyPolicyKey).(APIKeyIdentity)
	if !ok || identity.ID == "" || s.limiter == nil {
		return true
	}

	model := modelFromBody(ctx.PostBody())

	// (a) model scope check.
	if !modelInScopes(model, identity.Scopes) {
		writePolicyError(ctx, fasthttp.StatusForbidden, "model not permitted for this api key")
		return false
	}

	// (b) requests-per-minute check.
	if !s.limiter.AllowRequest(identity.ID, identity.RateLimitRPM) {
		s.metrics.IncRateLimitRejected()
		writePolicyError(ctx, fasthttp.StatusTooManyRequests, "request rate limit exceeded")
		return false
	}

	// (b') tokens-per-minute check, based on usage recorded so far.
	if !s.limiter.AllowTokens(identity.ID, identity.RateLimitTPM) {
		s.metrics.IncRateLimitRejected()
		writePolicyError(ctx, fasthttp.StatusTooManyRequests, "token rate limit exceeded")
		return false
	}

	// (c) daily spend cap check.
	if !s.limiter.WithinSpendCap(identity.ID, identity.DailySpendCapUSD) {
		s.metrics.IncSpendCapRejected()
		writePolicyError(ctx, fasthttp.StatusPaymentRequired, "daily spend cap reached")
		return false
	}

	return true
}

// recordKeyUsage records token and spend usage for a key after a response is
// known. keyID may be nil (no authenticated key). It never reads the pooled
// fasthttp ctx, so it is safe to call from the streaming-complete goroutine.
func (s *Server) recordKeyUsage(keyID *string, streamModel string, request *providers.ChatRequest, response *providers.ChatResponse, streamUsage *providers.Usage) {
	if keyID == nil || *keyID == "" || s.limiter == nil {
		return
	}

	var extracted *usage.Usage
	if response != nil {
		if value, ok := usage.FromChatResponse(*response); ok {
			extracted = &value
		}
	} else if streamUsage != nil {
		if value, ok := usage.FromChatResponse(providers.ChatResponse{Usage: streamUsage}); ok {
			extracted = &value
		}
	}
	if extracted == nil {
		return
	}

	s.limiter.AddTokens(*keyID, extracted.TotalTokens)

	metadata := inferenceLogMetadataWithAuth(request, response, streamModel, "", false, keyID)
	if cost := costForUsage(s.config.Store, metadata.provider, metadata.model, extracted); cost != nil {
		s.limiter.AddSpend(*keyID, *cost)
	}
}

// modelInScopes reports whether model matches one of the scope glob patterns.
// An empty scope list means all models are allowed.
// recordVirtualKeyUsage accumulates budget and tokens for a virtual key after a
// non-streaming inference response is known.
func (s *Server) recordVirtualKeyUsage(virtualKeyID, virtualKeyTeamID *string, model string, request *providers.ChatRequest, response *providers.ChatResponse, streamUsage *providers.Usage) {
	if virtualKeyID == nil || *virtualKeyID == "" || s.config.Governance == nil {
		return
	}
	keyID, err := strconv.ParseInt(*virtualKeyID, 10, 64)
	if err != nil {
		return
	}
	var teamID *int64
	if virtualKeyTeamID != nil && *virtualKeyTeamID != "" {
		if tid, err := strconv.ParseInt(*virtualKeyTeamID, 10, 64); err == nil {
			teamID = &tid
		}
	}
	var extracted *usage.Usage
	if response != nil {
		if value, ok := usage.FromChatResponse(*response); ok {
			extracted = &value
		}
	} else if streamUsage != nil {
		if value, ok := usage.FromChatResponse(providers.ChatResponse{Usage: streamUsage}); ok {
			extracted = &value
		}
	}
	if extracted == nil {
		return
	}
	modelName := model
	if modelName == "" && request != nil {
		modelName = request.Model
	}
	provider := providerFromModel(modelName)
	if response != nil && response.Provider != "" {
		provider = response.Provider.String()
	}
	_ = s.config.Governance.RecordUsage(keyID, teamID, providers.ModelProvider(provider), modelName, *extracted)
}

// recordVirtualKeyUsageSnapshot is the streaming-safe variant that takes a
// snapshot instead of the pooled ctx.
func (s *Server) recordVirtualKeyUsageSnapshot(snapshot streamLogSnapshot, model string, request *providers.ChatRequest, streamUsage *providers.Usage) {
	s.recordVirtualKeyUsage(snapshot.virtualKeyID, snapshot.virtualKeyTeamID, model, request, nil, streamUsage)
}

func modelInScopes(model string, scopes []string) bool {
	if len(scopes) == 0 {
		return true
	}
	for _, pattern := range scopes {
		if ok, err := path.Match(pattern, model); err == nil && ok {
			return true
		}
	}
	return false
}

// modelFromBody extracts the "model" field from an inference request body. It
// returns "" when the body is not JSON or lacks the field.
func modelFromBody(body []byte) string {
	var parsed struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}
	return parsed.Model
}

func writePolicyError(ctx *fasthttp.RequestCtx, status int, message string) {
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(status)
	body, err := json.Marshal(struct {
		Error string `json:"error"`
	}{Error: message})
	if err != nil {
		return
	}
	ctx.SetBody(body)
}
