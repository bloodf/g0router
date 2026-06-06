package mcp

import (
	"context"
	"testing"
)

func TestInjectAllowedToolsEmpty(t *testing.T) {
	ctx := context.Background()
	if got := InjectAllowedTools(ctx, nil); got != ctx {
		t.Fatal("expected same context for empty allowed")
	}
	if got := InjectAllowedTools(ctx, []string{}); got != ctx {
		t.Fatal("expected same context for empty slice")
	}
}

func TestInjectAllowedToolsNonEmpty(t *testing.T) {
	ctx := context.Background()
	got := InjectAllowedTools(ctx, []string{"tool1", "tool2"})
	if got == ctx {
		t.Fatal("expected new context when tools are injected")
	}
	allowed, ok := allowedToolsFromContext(got)
	if !ok {
		t.Fatal("expected allowed tools in context")
	}
	if len(allowed) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(allowed))
	}
}
