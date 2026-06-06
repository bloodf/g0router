package api

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestWithAllowedToolsNilGuards(t *testing.T) {
	eng := pipelineInferenceEngine{store: nil, tools: nil}
	ctx := context.Background()
	req := &providers.ChatRequest{Model: "gpt-4"}

	if got := eng.withAllowedTools(ctx, nil); got != ctx {
		t.Fatal("expected nil req to return original context")
	}
	if got := eng.withAllowedTools(ctx, req); got != ctx {
		t.Fatal("expected nil store/tools to return original context")
	}
}

func TestWithAllowedToolsEmptyGroup(t *testing.T) {
	s, err := store.NewStore(filepath.Join(t.TempDir(), "toolgroup.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	eng := pipelineInferenceEngine{store: s, tools: &mcp.ToolManager{}}
	ctx := context.Background()
	req := &providers.ChatRequest{Model: "gpt-4"}

	if got := eng.withAllowedTools(ctx, req); got != ctx {
		t.Fatal("expected empty group to return original context")
	}
}

func TestWithAllowedToolsGroupNotFound(t *testing.T) {
	s, err := store.NewStore(filepath.Join(t.TempDir(), "toolgroup2.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	_ = s.CreateCombo(&store.Combo{
		Name:         "combo1",
		Steps:        []store.ComboStep{},
		Strategy:     store.ComboStrategyFallback,
		MCPToolGroup: "missing_group",
		IsActive:     true,
	})

	eng := pipelineInferenceEngine{store: s, tools: &mcp.ToolManager{}}
	ctx := context.Background()
	req := &providers.ChatRequest{Model: "combo/combo1"}

	if got := eng.withAllowedTools(ctx, req); got != ctx {
		t.Fatal("expected missing group to return original context")
	}
}

func TestWithAllowedToolsGroupInactive(t *testing.T) {
	s, err := store.NewStore(filepath.Join(t.TempDir(), "toolgroup3.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	g, err := s.CreateMCPToolGroup("inactive_group", []string{"tool1"}, false)
	if err != nil {
		t.Fatalf("CreateMCPToolGroup: %v", err)
	}
	_ = s.CreateCombo(&store.Combo{
		Name:         "combo2",
		Steps:        []store.ComboStep{},
		Strategy:     store.ComboStrategyFallback,
		MCPToolGroup: g.Name,
		IsActive:     true,
	})

	eng := pipelineInferenceEngine{store: s, tools: &mcp.ToolManager{}}
	ctx := context.Background()
	req := &providers.ChatRequest{Model: "combo/combo2"}

	if got := eng.withAllowedTools(ctx, req); got != ctx {
		t.Fatal("expected inactive group to return original context")
	}
}

func TestWithAllowedToolsSuccess(t *testing.T) {
	s, err := store.NewStore(filepath.Join(t.TempDir(), "toolgroup4.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	g, err := s.CreateMCPToolGroup("active_group", []string{"tool1", "tool2"}, true)
	if err != nil {
		t.Fatalf("CreateMCPToolGroup: %v", err)
	}
	_ = s.CreateCombo(&store.Combo{
		Name:         "combo3",
		Steps:        []store.ComboStep{},
		Strategy:     store.ComboStrategyFallback,
		MCPToolGroup: g.Name,
		IsActive:     true,
	})

	eng := pipelineInferenceEngine{store: s, tools: &mcp.ToolManager{}}
	ctx := context.Background()
	req := &providers.ChatRequest{Model: "combo/combo3"}

	got := eng.withAllowedTools(ctx, req)
	if got == ctx {
		t.Fatal("expected context to be modified when tools are injected")
	}
}

func TestResolveMCPToolGroupCombo(t *testing.T) {
	s, err := store.NewStore(filepath.Join(t.TempDir(), "resolve.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	_ = s.CreateCombo(&store.Combo{
		Name:         "mycombo",
		Steps:        []store.ComboStep{},
		Strategy:     store.ComboStrategyFallback,
		MCPToolGroup: "tg1",
		IsActive:     true,
	})

	eng := pipelineInferenceEngine{store: s}
	req := &providers.ChatRequest{Model: "combo/mycombo"}
	if got := eng.resolveMCPToolGroup(context.Background(), req); got != "tg1" {
		t.Fatalf("expected tg1, got %s", got)
	}
}

func TestResolveMCPToolGroupComboNotFound(t *testing.T) {
	s, err := store.NewStore(filepath.Join(t.TempDir(), "resolve2.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	eng := pipelineInferenceEngine{store: s}
	req := &providers.ChatRequest{Model: "combo/nope"}
	if got := eng.resolveMCPToolGroup(context.Background(), req); got != "" {
		t.Fatalf("expected empty, got %s", got)
	}
}

func TestResolveMCPToolGroupVirtualKey(t *testing.T) {
	s, err := store.NewStore(filepath.Join(t.TempDir(), "resolve3.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	vk, _, err := s.CreateVirtualKey("vk1", nil, nil, "", nil, nil, "tg2")
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}

	eng := pipelineInferenceEngine{store: s}
	req := &providers.ChatRequest{Model: "gpt-4"}

	ctx := &fasthttp.RequestCtx{}
	ctx.SetUserValue(requestVirtualKeyIDKey, strconv.FormatInt(vk.ID, 10))

	if got := eng.resolveMCPToolGroup(ctx, req); got != "tg2" {
		t.Fatalf("expected tg2, got %s", got)
	}
}

func TestResolveMCPToolGroupVirtualKeyInvalidID(t *testing.T) {
	s, err := store.NewStore(filepath.Join(t.TempDir(), "resolve4.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	eng := pipelineInferenceEngine{store: s}
	req := &providers.ChatRequest{Model: "gpt-4"}

	ctx := &fasthttp.RequestCtx{}
	ctx.SetUserValue(requestVirtualKeyIDKey, "notanumber")

	if got := eng.resolveMCPToolGroup(ctx, req); got != "" {
		t.Fatalf("expected empty, got %s", got)
	}
}

func TestResolveMCPToolGroupVirtualKeyNotFound(t *testing.T) {
	s, err := store.NewStore(filepath.Join(t.TempDir(), "resolve5.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	eng := pipelineInferenceEngine{store: s}
	req := &providers.ChatRequest{Model: "gpt-4"}

	ctx := &fasthttp.RequestCtx{}
	ctx.SetUserValue(requestVirtualKeyIDKey, "99999")

	if got := eng.resolveMCPToolGroup(ctx, req); got != "" {
		t.Fatalf("expected empty, got %s", got)
	}
}

func TestResolveMCPToolGroupNoMatch(t *testing.T) {
	s, err := store.NewStore(filepath.Join(t.TempDir(), "resolve6.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	eng := pipelineInferenceEngine{store: s}
	req := &providers.ChatRequest{Model: "gpt-4"}

	if got := eng.resolveMCPToolGroup(context.Background(), req); got != "" {
		t.Fatalf("expected empty, got %s", got)
	}
}
