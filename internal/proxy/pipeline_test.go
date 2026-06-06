package proxy

import (
	"context"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

// fakeModelResolver is a test fake for ModelResolver.
type fakeModelResolver struct {
	resolveFunc func(ctx context.Context, model string) (string, error)
}

func (f *fakeModelResolver) ResolveModel(ctx context.Context, model string) (string, error) {
	return f.resolveFunc(ctx, model)
}

// fakeSettingsProvider is a test fake for SettingsProvider.
type fakeSettingsProvider struct {
	rtkEnabled     bool
	cavemanEnabled bool
	cavemanLevel   string
}

func (f *fakeSettingsProvider) RTKEnabled() bool     { return f.rtkEnabled }
func (f *fakeSettingsProvider) CavemanEnabled() bool { return f.cavemanEnabled }
func (f *fakeSettingsProvider) CavemanLevel() string { return f.cavemanLevel }

// fakeToolProvider is a test fake for ToolProvider.
type fakeToolProvider struct {
	tools []providers.Tool
}

func (f *fakeToolProvider) CompactToolsForRequest(ctx context.Context) []providers.Tool {
	return f.tools
}

func TestPipeline_Process_NilRequest(t *testing.T) {
	p := NewPipeline(nil, nil, nil)
	got, err := p.Process(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestPipeline_ResolveModel(t *testing.T) {
	resolver := &fakeModelResolver{
		resolveFunc: func(_ context.Context, model string) (string, error) {
			if model == "alias-model" {
				return "resolved-model", nil
			}
			return model, nil
		},
	}
	p := NewPipeline(resolver, nil, nil)

	req := providers.ChatRequest{Model: "alias-model"}
	got, err := p.resolveModel(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Model != "resolved-model" {
		t.Fatalf("expected model resolved-model, got %s", got.Model)
	}
}

func TestPipeline_ResolveModel_Error(t *testing.T) {
	wantErr := errors.New("resolve error")
	resolver := &fakeModelResolver{
		resolveFunc: func(_ context.Context, model string) (string, error) {
			return "", wantErr
		},
	}
	p := NewPipeline(resolver, nil, nil)

	req := providers.ChatRequest{Model: "unknown"}
	_, err := p.resolveModel(context.Background(), req)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestPipeline_ResolveModel_NilResolver(t *testing.T) {
	p := NewPipeline(nil, nil, nil)
	req := providers.ChatRequest{Model: "unchanged"}
	got, err := p.resolveModel(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Model != "unchanged" {
		t.Fatalf("expected model unchanged, got %s", got.Model)
	}
}

func TestPipeline_RTKCompression(t *testing.T) {
	settings := &fakeSettingsProvider{rtkEnabled: true}
	p := NewPipeline(nil, settings, nil)

	req := providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	}
	got := p.compressRTK(req)

	// Messages should be preserved; RTK on plain text is a pass-through.
	if len(got.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got.Messages))
	}
}

func TestPipeline_RTKCompression_Disabled(t *testing.T) {
	settings := &fakeSettingsProvider{rtkEnabled: false}
	p := NewPipeline(nil, settings, nil)

	req := providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	}
	got := p.compressRTK(req)
	if got.Messages[0].Content != "hello" {
		t.Fatalf("expected unchanged content, got %v", got.Messages[0].Content)
	}
}

func TestPipeline_CavemanInjection(t *testing.T) {
	settings := &fakeSettingsProvider{cavemanEnabled: true, cavemanLevel: "full"}
	p := NewPipeline(nil, settings, nil)

	req := providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	}
	got := p.injectCaveman(req)

	// Caveman injection prepends a system message.
	if len(got.Messages) != 2 {
		t.Fatalf("expected 2 messages (system + user), got %d", len(got.Messages))
	}
	if got.Messages[0].Role != "system" {
		t.Fatalf("expected first message to be system, got %s", got.Messages[0].Role)
	}
}

func TestPipeline_CavemanInjection_Disabled(t *testing.T) {
	settings := &fakeSettingsProvider{cavemanEnabled: false}
	p := NewPipeline(nil, settings, nil)

	req := providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	}
	got := p.injectCaveman(req)
	if len(got.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got.Messages))
	}
}

func TestPipeline_MCPToolsInjection(t *testing.T) {
	tools := &fakeToolProvider{
		tools: []providers.Tool{
			{Type: "function", Function: providers.ToolFunction{Name: "test_tool"}},
		},
	}
	p := NewPipeline(nil, nil, tools)

	req := providers.ChatRequest{Tools: nil}
	got := p.injectTools(context.Background(), req)

	if len(got.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(got.Tools))
	}
	if got.Tools[0].Function.Name != "test_tool" {
		t.Fatalf("expected test_tool, got %s", got.Tools[0].Function.Name)
	}
}

func TestPipeline_MCPToolsInjection_SkipsWhenToolsPresent(t *testing.T) {
	tools := &fakeToolProvider{
		tools: []providers.Tool{
			{Type: "function", Function: providers.ToolFunction{Name: "mcp_tool"}},
		},
	}
	p := NewPipeline(nil, nil, tools)

	req := providers.ChatRequest{
		Tools: []providers.Tool{
			{Type: "function", Function: providers.ToolFunction{Name: "existing_tool"}},
		},
	}
	got := p.injectTools(context.Background(), req)

	if len(got.Tools) != 1 {
		t.Fatalf("expected 1 tool (existing), got %d", len(got.Tools))
	}
	if got.Tools[0].Function.Name != "existing_tool" {
		t.Fatalf("expected existing_tool, got %s", got.Tools[0].Function.Name)
	}
}

func TestPipeline_MCPToolsInjection_NilProvider(t *testing.T) {
	p := NewPipeline(nil, nil, nil)
	req := providers.ChatRequest{Tools: nil}
	got := p.injectTools(context.Background(), req)
	if len(got.Tools) != 0 {
		t.Fatalf("expected 0 tools, got %d", len(got.Tools))
	}
}

func TestPipeline_Process_OrderedStages(t *testing.T) {
	resolver := &fakeModelResolver{
		resolveFunc: func(_ context.Context, model string) (string, error) {
			return "resolved-" + model, nil
		},
	}
	settings := &fakeSettingsProvider{
		rtkEnabled:     true,
		cavemanEnabled: true,
		cavemanLevel:   "full",
	}
	tools := &fakeToolProvider{
		tools: []providers.Tool{
			{Type: "function", Function: providers.ToolFunction{Name: "test_tool"}},
		},
	}
	p := NewPipeline(resolver, settings, tools)

	req := &providers.ChatRequest{
		Model: "my-model",
		Messages: []providers.Message{
			{Role: "user", Content: "hello world this is a test message for compression"},
		},
	}

	got, err := p.Process(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Model should be resolved first.
	if got.Model != "resolved-my-model" {
		t.Fatalf("expected resolved model, got %s", got.Model)
	}

	// Should have caveman system message prepended.
	if len(got.Messages) < 1 || got.Messages[0].Role != "system" {
		t.Fatalf("expected caveman system message prepended")
	}

	// Should have tools injected.
	if len(got.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(got.Tools))
	}

	// Original request should be unmodified.
	if req.Model != "my-model" {
		t.Fatal("original request was modified")
	}
}

func TestPipeline_Process_ResolveError(t *testing.T) {
	wantErr := errors.New("model not found")
	resolver := &fakeModelResolver{
		resolveFunc: func(_ context.Context, model string) (string, error) {
			return "", wantErr
		},
	}
	p := NewPipeline(resolver, nil, nil)

	req := &providers.ChatRequest{Model: "unknown"}
	_, err := p.Process(context.Background(), req)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestPipeline_Process_NoSettings(t *testing.T) {
	p := NewPipeline(nil, nil, nil)

	req := &providers.ChatRequest{
		Model: "m",
		Messages: []providers.Message{
			{Role: "user", Content: "hi"},
		},
	}

	got, err := p.Process(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Model != "m" {
		t.Fatalf("expected model m, got %s", got.Model)
	}
	if len(got.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got.Messages))
	}
}
