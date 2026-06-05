package proxy

import (
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
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

// helpers for telemetry strategy tests.

var (
	stepGroq      = ComboStep{Provider: providers.ProviderGroq, Model: "llama-3.3-70b-versatile"}
	stepOpenAI    = ComboStep{Provider: providers.ProviderOpenAI, Model: "gpt-4o-mini"}
	stepAnthropic = ComboStep{Provider: providers.ProviderAnthropic, Model: "claude-sonnet-4"}
)

func makeStats(entries map[ComboStep]store.ModelStat) map[string]store.ModelStat {
	m := make(map[string]store.ModelStat, len(entries))
	for step, stat := range entries {
		m[telemetryKey(step)] = stat
	}
	return m
}

func TestTelemetryOrderFastestSortsByLatency(t *testing.T) {
	steps := []ComboStep{stepAnthropic, stepGroq, stepOpenAI}
	stats := makeStats(map[ComboStep]store.ModelStat{
		stepAnthropic: {AvgLatencyMS: 300, Requests: 10},
		stepGroq:      {AvgLatencyMS: 80, Requests: 10},
		stepOpenAI:    {AvgLatencyMS: 150, Requests: 10},
	})

	ordered := telemetryOrder(steps, stats, func(s store.ModelStat) float64 { return s.AvgLatencyMS })

	if ordered[0] != stepGroq {
		t.Fatalf("fastest[0] = %v, want groq (lowest latency 80ms)", ordered[0])
	}
	if ordered[1] != stepOpenAI {
		t.Fatalf("fastest[1] = %v, want openai (150ms)", ordered[1])
	}
	if ordered[2] != stepAnthropic {
		t.Fatalf("fastest[2] = %v, want anthropic (300ms)", ordered[2])
	}
}

func TestTelemetryOrderCheapestSortsByCost(t *testing.T) {
	steps := []ComboStep{stepAnthropic, stepGroq, stepOpenAI}
	stats := makeStats(map[ComboStep]store.ModelStat{
		stepAnthropic: {AvgCostUSD: 0.003, Requests: 5},
		stepGroq:      {AvgCostUSD: 0.0001, Requests: 5},
		stepOpenAI:    {AvgCostUSD: 0.001, Requests: 5},
	})

	ordered := telemetryOrder(steps, stats, func(s store.ModelStat) float64 { return s.AvgCostUSD })

	if ordered[0] != stepGroq {
		t.Fatalf("cheapest[0] = %v, want groq (lowest cost)", ordered[0])
	}
	if ordered[1] != stepOpenAI {
		t.Fatalf("cheapest[1] = %v, want openai", ordered[1])
	}
	if ordered[2] != stepAnthropic {
		t.Fatalf("cheapest[2] = %v, want anthropic (most expensive)", ordered[2])
	}
}

func TestTelemetryOrderUnknownStepsSortLast(t *testing.T) {
	steps := []ComboStep{stepAnthropic, stepGroq, stepOpenAI}
	// Only groq has telemetry; the other two are unknown.
	stats := makeStats(map[ComboStep]store.ModelStat{
		stepGroq: {AvgLatencyMS: 80, Requests: 3},
	})

	ordered := telemetryOrder(steps, stats, func(s store.ModelStat) float64 { return s.AvgLatencyMS })

	if ordered[0] != stepGroq {
		t.Fatalf("first should be groq (only known step), got %v", ordered[0])
	}
	// The two unknown steps follow in their original relative order (stable sort).
	if ordered[1] != stepAnthropic || ordered[2] != stepOpenAI {
		t.Fatalf("unknown steps should preserve original order: got %v %v", ordered[1], ordered[2])
	}
}

func TestTelemetryOrderAllUnknownPreservesOriginalOrder(t *testing.T) {
	steps := []ComboStep{stepAnthropic, stepGroq, stepOpenAI}
	ordered := telemetryOrder(steps, nil, func(s store.ModelStat) float64 { return s.AvgLatencyMS })

	for i, want := range steps {
		if ordered[i] != want {
			t.Fatalf("ordered[%d] = %v, want %v (original order preserved)", i, ordered[i], want)
		}
	}
}

func TestTelemetryOrderNilStatsFallsBackToStoredOrder(t *testing.T) {
	steps := []ComboStep{stepGroq, stepOpenAI, stepAnthropic}
	sel := &comboSelector{}

	ordered, _ := sel.orderedStepsWithStats(store.ComboStrategyFastest, steps, nil, nil)

	for i, want := range steps {
		if ordered[i] != want {
			t.Fatalf("nil stats fastest[%d] = %v, want %v", i, ordered[i], want)
		}
	}
}

func TestTelemetryOrderTiesPreserveOriginalOrder(t *testing.T) {
	steps := []ComboStep{stepAnthropic, stepGroq, stepOpenAI}
	// All identical latency — stable sort must preserve input order.
	stats := makeStats(map[ComboStep]store.ModelStat{
		stepAnthropic: {AvgLatencyMS: 100, Requests: 1},
		stepGroq:      {AvgLatencyMS: 100, Requests: 1},
		stepOpenAI:    {AvgLatencyMS: 100, Requests: 1},
	})

	ordered := telemetryOrder(steps, stats, func(s store.ModelStat) float64 { return s.AvgLatencyMS })

	for i, want := range steps {
		if ordered[i] != want {
			t.Fatalf("tie case: ordered[%d] = %v, want %v", i, ordered[i], want)
		}
	}
}
