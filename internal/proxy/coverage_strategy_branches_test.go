package proxy

import (
	"fmt"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

// TestTelemetryOrderIUnknownJKnownReturnsJFirst exercises the `if !iOK { return false }`
// branch (line 97) in telemetryOrder: i is unknown, j is known → j sorts first.
// To trigger L97, the sort must compare (unknown, known) as (i, j) — this happens
// when the known step is at index 0 (sort's insertion pass compares index 1 vs 0,
// then index 2 vs 1; the (index1=unknown, index0=known) comparison hits L97).
func TestTelemetryOrderIUnknownJKnownBranch(t *testing.T) {
	// Put known step FIRST so the insertion sort compares (unknown, known) as (i,j).
	knownStep := ComboStep{Provider: providers.ProviderGroq, Model: "llama-3.3-70b-versatile"}
	unknownStep1 := ComboStep{Provider: providers.ProviderAnthropic, Model: "unknown-model-xyz"}
	unknownStep2 := ComboStep{Provider: providers.ProviderOpenAI, Model: "no-telemetry-model"}
	steps := []ComboStep{knownStep, unknownStep1, unknownStep2}

	stats := makeStats(map[ComboStep]store.ModelStat{
		knownStep: {AvgLatencyMS: 50, Requests: 5},
		// unknownStep1 and unknownStep2 have no entries
	})

	ordered := telemetryOrder(steps, stats, func(s store.ModelStat) float64 { return s.AvgLatencyMS })
	// Known step should remain first (or sort to front).
	if ordered[0] != knownStep {
		t.Fatalf("known step should sort first; got %v first", ordered[0])
	}
}

// TestRequestHasVisionDefaultBranchReturnsTrue exercises the default case
// in requestHasVision (line 243-244) when fmt.Sprintf of the content contains
// "image_url".
type imageURLStringer struct{}

func (imageURLStringer) String() string { return "image_url:http://x/y.png" }

func TestRequestHasVisionDefaultBranchTrueCase(t *testing.T) {
	// imageURLStringer.String() is not called by fmt.Sprintf("%v", ...) unless it
	// implements fmt.Stringer — which it does. So %v will produce "image_url:..."
	// which contains "image_url", triggering return true.
	req := &providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "user", Content: imageURLStringer{}},
		},
	}
	if !requestHasVision(req) {
		t.Error("Stringer content containing 'image_url' should trigger vision detection")
	}
}

// TestPartsHaveImageImageURLKey exercises the `m["image_url"]` key check (line 260)
// when the type field is absent/different but the "image_url" key exists.
func TestPartsHaveImageImageURLKey(t *testing.T) {
	parts := []any{
		map[string]any{
			"type":      "text", // not "image_url" or "image"
			"image_url": map[string]any{"url": "http://x/y.png"}, // but has image_url key
		},
	}
	if !partsHaveImage(parts) {
		t.Error("part with image_url key should trigger partsHaveImage")
	}
}

// TestClassifyRequestTaskDefaultContent exercises the default branch in
// requestText/classifyRequestTask for non-string non-slice content.
func TestClassifyRequestTaskFmtSprintf(t *testing.T) {
	// Use a custom type that fmt.Sprintf renders non-trivially.
	req := &providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "user", Content: fmt.Errorf("some error content")},
		},
	}
	// Should not panic; classification may be taskSimple.
	task := classifyRequestTask(req)
	if task == "" {
		t.Fatal("classifyRequestTask should return a non-empty task")
	}
}
