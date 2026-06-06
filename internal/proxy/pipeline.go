package proxy

import (
	"context"
	"fmt"

	"github.com/bloodf/g0router/internal/guardrails"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/rtk"
)

// ModelResolver resolves a user-facing model name to an upstream model name.
type ModelResolver interface {
	ResolveModel(ctx context.Context, model string) (string, error)
}

// SettingsProvider supplies runtime settings for preprocessing decisions.
type SettingsProvider interface {
	RTKEnabled() bool
	CavemanEnabled() bool
	CavemanLevel() string
	GuardrailsEnabled() bool
	GuardrailsBlocklist() []string
	PIIRedactionEnabled() bool
	PIIRedactionTypes() []string
}

// ToolProvider supplies MCP tools for a request.
type ToolProvider interface {
	CompactToolsForRequest(ctx context.Context) []providers.Tool
}

// Pipeline applies an ordered sequence of preprocessing stages to a
// ChatRequest before it reaches provider dispatch.
//
// Stages (in order):
//   1. Model resolution (alias → combo → catalog route)
//   2. Guardrails (blocklist + PII redaction)
//   3. RTK compression
//   4. Caveman injection
//   5. MCP tool injection
type Pipeline struct {
	resolver ModelResolver
	settings SettingsProvider
	tools    ToolProvider
}

// NewPipeline creates a Pipeline with the given stage dependencies.
// Any dependency may be nil; the corresponding stage becomes a no-op.
func NewPipeline(resolver ModelResolver, settings SettingsProvider, tools ToolProvider) *Pipeline {
	return &Pipeline{
		resolver: resolver,
		settings: settings,
		tools:    tools,
	}
}

// Process runs the pipeline stages in order and returns the processed request.
// A nil request is returned unchanged.
func (p *Pipeline) Process(ctx context.Context, req *providers.ChatRequest) (*providers.ChatRequest, error) {
	if req == nil {
		return nil, nil
	}

	processed := *req

	processed, err := p.resolveModel(ctx, processed)
	if err != nil {
		return nil, fmt.Errorf("pipeline resolve model: %w", err)
	}

	processed, err = p.applyGuardrails(processed)
	if err != nil {
		return nil, err
	}

	processed = p.compressRTK(processed)
	processed = p.injectCaveman(processed)
	processed = p.injectTools(ctx, processed)

	return &processed, nil
}

// applyGuardrails is stage 2: check blocklist and redact PII when enabled.
func (p *Pipeline) applyGuardrails(req providers.ChatRequest) (providers.ChatRequest, error) {
	if p.settings == nil {
		return req, nil
	}

	grCfg := guardrails.Config{
		Enabled:   p.settings.GuardrailsEnabled(),
		Blocklist: p.settings.GuardrailsBlocklist(),
	}
	blocked, _, err := guardrails.CheckRequest(grCfg, &req)
	if err != nil {
		return providers.ChatRequest{}, err
	}
	if blocked {
		return providers.ChatRequest{}, guardrails.ErrBlocklistMatch
	}

	piiCfg := guardrails.PIIConfig{
		Enabled: p.settings.PIIRedactionEnabled(),
		Types:   p.settings.PIIRedactionTypes(),
	}
	redacted := guardrails.RedactRequest(piiCfg, &req)
	if redacted != nil {
		return *redacted, nil
	}
	return req, nil
}

// resolveModel is stage 1: alias → combo → catalog route → provider.
func (p *Pipeline) resolveModel(ctx context.Context, req providers.ChatRequest) (providers.ChatRequest, error) {
	if p.resolver == nil {
		return req, nil
	}
	resolved, err := p.resolver.ResolveModel(ctx, req.Model)
	if err != nil {
		return providers.ChatRequest{}, fmt.Errorf("resolve model: %w", err)
	}
	req.Model = resolved
	return req, nil
}

// compressRTK is stage 2: apply RTK compression when enabled.
func (p *Pipeline) compressRTK(req providers.ChatRequest) providers.ChatRequest {
	if p.settings != nil && p.settings.RTKEnabled() {
		return rtk.CompressRequest(req)
	}
	return req
}

// injectCaveman is stage 3: prepend caveman prompt when enabled.
func (p *Pipeline) injectCaveman(req providers.ChatRequest) providers.ChatRequest {
	if p.settings != nil && p.settings.CavemanEnabled() {
		return rtk.InjectCaveman(req, rtk.CavemanLevel(p.settings.CavemanLevel()))
	}
	return req
}

// injectTools is stage 4: inject MCP tools when no client tools are present.
func (p *Pipeline) injectTools(ctx context.Context, req providers.ChatRequest) providers.ChatRequest {
	if len(req.Tools) == 0 && p.tools != nil {
		req.Tools = p.tools.CompactToolsForRequest(ctx)
	}
	return req
}
