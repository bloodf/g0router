package rtk

import (
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func TestInjectCavemanAddsSystemMessageForEachLevel(t *testing.T) {
	tests := []struct {
		name  string
		level CavemanLevel
		want  string
	}{
		{name: "lite", level: CavemanLite, want: "Respond tersely"},
		{name: "full", level: CavemanFull, want: "terse caveman"},
		{name: "ultra", level: CavemanUltra, want: "ultra-terse"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := providers.ChatRequest{
				Model: "gpt-4o",
				Messages: []providers.Message{
					{Role: "user", Content: "summarize this"},
				},
			}

			got := InjectCaveman(req, tt.level)

			if len(got.Messages) != 2 {
				t.Fatalf("messages len = %d, want 2", len(got.Messages))
			}
			if got.Messages[0].Role != "system" {
				t.Fatalf("first role = %q, want system", got.Messages[0].Role)
			}
			content, ok := got.Messages[0].Content.(string)
			if !ok {
				t.Fatalf("system content type = %T, want string", got.Messages[0].Content)
			}
			if !strings.Contains(content, tt.want) {
				t.Fatalf("system content %q does not contain %q", content, tt.want)
			}
			assertMessageEqual(t, got.Messages[1], req.Messages[0])
		})
	}
}

func TestInjectCavemanPrependsExistingSystemContent(t *testing.T) {
	req := providers.ChatRequest{
		Model: "gpt-4o",
		Messages: []providers.Message{
			{Role: "system", Content: "Existing rules."},
			{Role: "user", Content: "hello"},
		},
	}

	got := InjectCaveman(req, CavemanLite)

	if len(got.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(got.Messages))
	}
	content, ok := got.Messages[0].Content.(string)
	if !ok {
		t.Fatalf("system content type = %T, want string", got.Messages[0].Content)
	}
	if !strings.HasPrefix(content, CavemanLitePrompt+"\n\n") {
		t.Fatalf("system content = %q, want caveman prompt prefix", content)
	}
	if !strings.HasSuffix(content, "Existing rules.") {
		t.Fatalf("system content = %q, want existing rules preserved", content)
	}
}

func TestInjectCavemanPreservesNonStringSystemContent(t *testing.T) {
	req := providers.ChatRequest{
		Model: "gpt-4o",
		Messages: []providers.Message{
			{
				Role: "system",
				Content: []map[string]string{
					{"type": "text", "text": "Existing rules."},
				},
			},
			{Role: "user", Content: "hello"},
		},
	}

	got := InjectCaveman(req, CavemanUltra)

	if len(got.Messages) != 3 {
		t.Fatalf("messages len = %d, want 3", len(got.Messages))
	}
	if got.Messages[0].Role != "system" {
		t.Fatalf("first role = %q, want system", got.Messages[0].Role)
	}
	content, ok := got.Messages[0].Content.(string)
	if !ok {
		t.Fatalf("injected content type = %T, want string", got.Messages[0].Content)
	}
	if !strings.Contains(content, "ultra-terse") {
		t.Fatalf("injected content = %q, want ultra prompt", content)
	}
	if got.Messages[1].Content == nil {
		t.Fatal("existing system content should be preserved")
	}
}

func TestInjectCavemanDoesNotMutateCallerRequest(t *testing.T) {
	req := providers.ChatRequest{
		Model: "gpt-4o",
		Messages: []providers.Message{
			{Role: "system", Content: "Existing rules."},
			{Role: "user", Content: "hello"},
		},
	}
	originalFirst := req.Messages[0]
	originalLen := len(req.Messages)

	got := InjectCaveman(req, CavemanFull)

	if len(req.Messages) != originalLen {
		t.Fatalf("original messages len = %d, want %d", len(req.Messages), originalLen)
	}
	assertMessageEqual(t, req.Messages[0], originalFirst)
	if got.Messages[0].Content == req.Messages[0].Content {
		t.Fatal("returned system message should include caveman prompt")
	}
}

func TestInjectCavemanUnknownLevelLeavesRequestUnchanged(t *testing.T) {
	req := providers.ChatRequest{
		Model: "gpt-4o",
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	}

	got := InjectCaveman(req, CavemanLevel("mega"))

	if len(got.Messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(got.Messages))
	}
	assertMessageEqual(t, got.Messages[0], req.Messages[0])
}

func assertMessageEqual(t *testing.T, got providers.Message, want providers.Message) {
	t.Helper()

	if got.Role != want.Role {
		t.Fatalf("role = %q, want %q", got.Role, want.Role)
	}
	if got.Content != want.Content {
		t.Fatalf("content = %v, want %v", got.Content, want.Content)
	}
}
