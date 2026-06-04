package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/store"
)

func TestProvidersListShowsKnownProviders(t *testing.T) {
	cmd := NewRootCommand("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"providers", "list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	output := out.String()
	for _, want := range []string{"alibaba", "anthropic", "azure", "bedrock", "cerebras", "cohere", "deepseek", "fireworks", "gemini", "github-copilot", "groq", "huggingface", "litellm", "lm-studio", "mistral", "minimax", "nebius", "nvidia", "ollama", "openai", "openrouter", "perplexity", "qianfan", "qwen", "together", "vercel-ai-gateway", "vertex", "vllm", "xai", "zhipu"} {
		if !strings.Contains(output, want+"\n") {
			t.Fatalf("output = %q, want provider %q", output, want)
		}
	}
	for _, wantAbsent := range []string{"cursor", "replicate"} {
		if strings.Contains(output, wantAbsent+"\n") {
			t.Fatalf("output = %q, should not list non-public provider %q", output, wantAbsent)
		}
	}
}

func TestProvidersTestRejectsUnknownProvider(t *testing.T) {
	cmd := NewRootCommand("test")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"providers", "test", "unknown"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute error is nil")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("error = %q, want unknown provider", err.Error())
	}
}

func TestProvidersTestRequiresActiveConnectionForCredentialProvider(t *testing.T) {
	dataDir := t.TempDir()
	cmd := NewRootCommand("test")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--data-dir", dataDir, "providers", "test", "perplexity"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute error is nil")
	}
	if !strings.Contains(err.Error(), "no active connection for provider: perplexity") {
		t.Fatalf("error = %q, want missing active connection", err.Error())
	}
}

func TestProvidersTestCanonicalizesCodexToOpenAI(t *testing.T) {
	dataDir := t.TempDir()
	s, err := store.NewStore(filepath.Join(dataDir, "g0router.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "codex",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	s.Close()

	cmd := NewRootCommand("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--data-dir", dataDir, "providers", "test", "codex"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "openai: active connection") {
		t.Fatalf("output = %q, want canonical openai active connection", out.String())
	}
}

func TestProvidersTestReportsAuthOnlyProvider(t *testing.T) {
	cmd := NewRootCommand("test")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"providers", "test", "cursor"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute error is nil")
	}
	if !strings.Contains(err.Error(), "cursor is auth_only") {
		t.Fatalf("error = %q, want auth-only provider status", err.Error())
	}
}
