package rtk

import (
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func TestCompressRequestAppliesDetectedFiltersToStringMessages(t *testing.T) {
	req := providers.ChatRequest{
		Model: "gpt-4o",
		Messages: []providers.Message{
			{
				Role: "user",
				Content: `On branch codex/wave-2c-task-73
Changes not staged for commit:
  modified:   internal/rtk/rtk.go
Untracked files:
  internal/rtk/rtk_test.go
`,
			},
			{
				Role: "assistant",
				Content: `internal/rtk/rtk.go:12:func CompressRequest(req providers.ChatRequest) providers.ChatRequest
internal/rtk/autodetect.go:10:func DetectFormat(input string) ContentFormat
`,
			},
		},
	}

	got := CompressRequest(req)

	assertStringContent(t, got.Messages[0], `branch codex/wave-2c-task-73
M internal/rtk/rtk.go
?? internal/rtk/rtk_test.go`)
	assertStringContent(t, got.Messages[1], `internal/rtk/rtk.go:12 func CompressRequest(req providers.ChatRequest) providers.ChatRequest
internal/rtk/autodetect.go:10 func DetectFormat(input string) ContentFormat`)
}

func TestCompressRequestDoesNotMutateCallerRequest(t *testing.T) {
	req := providers.ChatRequest{
		Model: "gpt-4o",
		Messages: []providers.Message{
			{
				Role: "user",
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

func TestCompressRequestSmartTruncatesPlainText(t *testing.T) {
	input := strings.Repeat("plain output ", 500)
	req := providers.ChatRequest{
		Model: "gpt-4o",
		Messages: []providers.Message{
			{Role: "user", Content: input},
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
