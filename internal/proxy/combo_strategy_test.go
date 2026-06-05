package proxy

import (
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func msgReq(content any) *providers.ChatRequest {
	return &providers.ChatRequest{
		Messages: []providers.Message{{Role: "user", Content: content}},
	}
}

func TestClassifyRequestTask(t *testing.T) {
	imagePart := []any{
		map[string]any{"type": "text", "text": "what is this"},
		map[string]any{"type": "image_url", "image_url": map[string]any{"url": "https://x/y.png"}},
	}

	big := strings.Repeat("x", autoHeavyContextThreshold+1)

	tests := []struct {
		name string
		req  *providers.ChatRequest
		want string
	}{
		{"nil request", nil, taskSimple},
		{"empty messages", &providers.ChatRequest{}, taskSimple},
		{"short plain chat", msgReq("hello there"), taskSimple},
		{"plain sentence with one semicolon", msgReq("I went home; then I slept."), taskSimple},
		{"vision multimodal part", msgReq(imagePart), taskVision},
		{"vision embedded data url string", msgReq("look at data:image/png;base64,AAAA"), taskVision},
		{"vision image_url marker string", msgReq(`{"image_url": "http://x"}`), taskVision},
		{
			"tools",
			&providers.ChatRequest{
				Messages: []providers.Message{{Role: "user", Content: "hi"}},
				Tools:    []providers.Tool{{Type: "function"}},
			},
			taskTools,
		},
		{"code fence", msgReq("here:\n```go\nfunc main() {}\n```"), taskCode},
		{"code density", msgReq("func a() {}\nfunc b() {}\nimport x => y;\n"), taskCode},
		{"large context", msgReq(big), taskLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyRequestTask(tt.req); got != tt.want {
				t.Fatalf("classifyRequestTask = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyRequestTaskPriority(t *testing.T) {
	// Vision wins over tools when both present.
	req := &providers.ChatRequest{
		Messages: []providers.Message{{Role: "user", Content: []any{
			map[string]any{"type": "image_url", "image_url": map[string]any{"url": "u"}},
		}}},
		Tools: []providers.Tool{{Type: "function"}},
	}
	if got := classifyRequestTask(req); got != taskVision {
		t.Fatalf("classifyRequestTask = %q, want %q", got, taskVision)
	}
}

func TestSelectAutoStepIndexRouting(t *testing.T) {
	steps := []ComboStep{
		{Provider: providers.ProviderAnthropic, Model: "claude-sonnet-4"},
		{Provider: providers.ProviderOpenAI, Model: "gpt-4o-mini"},
		{Provider: providers.ProviderGroq, Model: "llama-3.3-70b-versatile"},
	}
	last := len(steps) - 1

	big := strings.Repeat("x", autoHeavyContextThreshold+1)
	imageReq := msgReq([]any{map[string]any{"type": "image_url", "image_url": map[string]any{"url": "u"}}})

	tests := []struct {
		name  string
		steps []ComboStep
		req   *providers.ChatRequest
		want  int
	}{
		{"empty steps", nil, msgReq("hi"), 0},
		{"single step", steps[:1], msgReq("hi"), 0},
		{"short plain -> last", steps, msgReq("hi"), last},
		{"nil request -> last", steps, nil, last},
		{"vision -> first", steps, imageReq, 0},
		{
			"tools -> first",
			steps,
			&providers.ChatRequest{
				Messages: []providers.Message{{Role: "user", Content: "hi"}},
				Tools:    []providers.Tool{{Type: "function"}},
			},
			0,
		},
		{"code fence -> first", steps, msgReq("```\ncode\n```"), 0},
		{"large -> first", steps, msgReq(big), 0},
		{"plain semicolon not code -> last", steps, msgReq("go home; sleep."), last},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := selectAutoStepIndex(tt.steps, tt.req); got != tt.want {
				t.Fatalf("selectAutoStepIndex = %d, want %d", got, tt.want)
			}
		})
	}
}
