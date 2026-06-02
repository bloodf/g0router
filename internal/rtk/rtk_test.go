package rtk

import (
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func TestCompressRequestOnlyCompressesToolStringMessages(t *testing.T) {
	toolOutput := `On branch codex/wave-2c-task-73
Changes not staged for commit:
  modified:   internal/rtk/rtk.go
Untracked files:
  internal/rtk/rtk_test.go
`
	req := providers.ChatRequest{
		Model: "gpt-4o",
		Messages: []providers.Message{
			{
				Role: "user",
				Content: toolOutput,
			},
			{
				Role: "assistant",
				Content: `internal/rtk/rtk.go:12:func CompressRequest(req providers.ChatRequest) providers.ChatRequest
internal/rtk/autodetect.go:10:func DetectFormat(input string) ContentFormat
`,
			},
			{
				Role:    "system",
				Content: toolOutput,
			},
			{
				Role:    "tool",
				Content: toolOutput,
			},
		},
	}

	got := CompressRequest(req)

	assertStringContent(t, got.Messages[0], toolOutput)
	assertStringContent(t, got.Messages[1], `internal/rtk/rtk.go:12:func CompressRequest(req providers.ChatRequest) providers.ChatRequest
internal/rtk/autodetect.go:10:func DetectFormat(input string) ContentFormat
`)
	assertStringContent(t, got.Messages[2], toolOutput)
	assertStringContent(t, got.Messages[3], `branch codex/wave-2c-task-73
M internal/rtk/rtk.go
?? internal/rtk/rtk_test.go`)
}

func TestCompressRequestDoesNotMutateCallerRequest(t *testing.T) {
	req := providers.ChatRequest{
		Model: "gpt-4o",
		Messages: []providers.Message{
			{
				Role: "tool",
				Content: `     1	package rtk
     2
     3	func DetectFormat(input string) ContentFormat {
`,
			},
		},
	}
	originalMessage := req.Messages[0]

	got := CompressRequest(req)

	assertMessageEqual(t, req.Messages[0], originalMessage)
	if got.Messages[0].Content == req.Messages[0].Content {
		t.Fatal("returned message should contain compressed content")
	}
}

func TestCompressRequestLeavesNonStringContentUntouched(t *testing.T) {
	content := []map[string]string{
		{"type": "text", "text": "plain multimodal content"},
	}
	req := providers.ChatRequest{
		Model: "gpt-4o",
		Messages: []providers.Message{
			{Role: "user", Content: content},
		},
	}

	got := CompressRequest(req)

	if len(got.Messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(got.Messages))
	}
	if got.Messages[0].Content == nil {
		t.Fatal("non-string content should be preserved")
	}
	if len(req.Messages[0].Content.([]map[string]string)) != 1 {
		t.Fatal("original non-string content should remain unchanged")
	}
}

func TestCompressRequestCompressesToolResultBlocks(t *testing.T) {
	req := providers.ChatRequest{
		Model: "claude-3-5-sonnet",
		Messages: []providers.Message{
			{
				Role: "user",
				Content: []map[string]any{
					{"type": "text", "text": "please summarize"},
					{
						"type": "tool_result",
						"content": `internal/rtk/rtk.go:12:func CompressRequest(req providers.ChatRequest) providers.ChatRequest
internal/rtk/autodetect.go:10:func DetectFormat(input string) ContentFormat
`,
					},
				},
			},
		},
	}

	got := CompressRequest(req)

	originalBlocks := req.Messages[0].Content.([]map[string]any)
	if originalBlocks[1]["content"] == gotToolResultContent(t, got.Messages[0]) {
		t.Fatal("returned tool_result block should contain compressed content")
	}
	if originalBlocks[1]["content"].(string) != `internal/rtk/rtk.go:12:func CompressRequest(req providers.ChatRequest) providers.ChatRequest
internal/rtk/autodetect.go:10:func DetectFormat(input string) ContentFormat
` {
		t.Fatal("original tool_result block should remain unchanged")
	}
	if gotToolResultContent(t, got.Messages[0]) != `internal/rtk/rtk.go:12 func CompressRequest(req providers.ChatRequest) providers.ChatRequest
internal/rtk/autodetect.go:10 func DetectFormat(input string) ContentFormat` {
		t.Fatalf("unexpected compressed tool_result content: %q", gotToolResultContent(t, got.Messages[0]))
	}
}

func TestCompressRequestSmartTruncatesToolPlainText(t *testing.T) {
	input := strings.Repeat("plain output ", 500)
	req := providers.ChatRequest{
		Model: "gpt-4o",
		Messages: []providers.Message{
			{Role: "tool", Content: input},
		},
	}

	got := CompressRequest(req)
	content := stringContent(t, got.Messages[0])

	if len(content) >= len(input) {
		t.Fatalf("compressed length = %d, want less than %d", len(content), len(input))
	}
	if !strings.Contains(content, "\n... truncated ") {
		t.Fatalf("compressed content missing truncation marker: %q", content)
	}
}

func gotToolResultContent(t *testing.T, message providers.Message) string {
	t.Helper()

	blocks, ok := message.Content.([]map[string]any)
	if !ok {
		t.Fatalf("content type = %T, want []map[string]any", message.Content)
	}
	content, ok := blocks[1]["content"].(string)
	if !ok {
		t.Fatalf("tool_result content type = %T, want string", blocks[1]["content"])
	}
	return content
}

func assertStringContent(t *testing.T, message providers.Message, want string) {
	t.Helper()

	got := stringContent(t, message)
	if got != want {
		t.Fatalf("content =\n%s\nwant\n%s", got, want)
	}
}

func stringContent(t *testing.T, message providers.Message) string {
	t.Helper()

	content, ok := message.Content.(string)
	if !ok {
		t.Fatalf("content type = %T, want string", message.Content)
	}
	return content
}
