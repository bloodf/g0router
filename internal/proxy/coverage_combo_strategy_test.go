package proxy

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

// TestPartsTextExtractsTextFields exercises partsText (0% covered).
func TestPartsTextExtractsTextFields(t *testing.T) {
	parts := []any{
		map[string]any{"type": "text", "text": "hello"},
		map[string]any{"type": "image_url", "image_url": map[string]any{"url": "u"}},
		map[string]any{"type": "text", "text": "world"},
		"not a map",
		map[string]any{"type": "text"},           // no text field
		map[string]any{"type": "text", "text": 42}, // non-string text
	}
	got := partsText(parts)
	if !strings.Contains(got, "hello") {
		t.Errorf("partsText missing 'hello', got %q", got)
	}
	if !strings.Contains(got, "world") {
		t.Errorf("partsText missing 'world', got %q", got)
	}
	// Non-text part should not contribute.
	if strings.Contains(got, "image_url") {
		t.Errorf("partsText should not include image_url part, got %q", got)
	}
}

// TestPartsTextEmpty exercises the empty slice path.
func TestPartsTextEmpty(t *testing.T) {
	got := partsText([]any{})
	if got != "" {
		t.Errorf("partsText empty = %q, want empty string", got)
	}
}

// TestRequestTextWithPartsContent exercises the []any branch in requestText (75%).
func TestRequestTextWithPartsContent(t *testing.T) {
	req := &providers.ChatRequest{
		Messages: []providers.Message{
			{
				Role: "user",
				Content: []any{
					map[string]any{"type": "text", "text": "describe this image"},
					map[string]any{"type": "image_url", "image_url": map[string]any{"url": "http://x/y.png"}},
				},
			},
		},
	}
	got := requestText(req)
	if !strings.Contains(got, "describe this image") {
		t.Errorf("requestText from parts = %q, want to contain 'describe this image'", got)
	}
}

// TestRequestTextWithDefaultContent exercises the default branch in requestText.
func TestRequestTextWithDefaultContent(t *testing.T) {
	req := &providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "user", Content: 12345},
		},
	}
	got := requestText(req)
	if got == "" {
		t.Error("requestText with non-string non-slice content should produce non-empty string")
	}
}

// TestRequestHasVisionDefaultBranch exercises the default Content branch in requestHasVision.
func TestRequestHasVisionDefaultBranch(t *testing.T) {
	// Content is an integer — triggers default fmt.Sprintf path. No image marker present.
	req := &providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "user", Content: 999},
		},
	}
	if requestHasVision(req) {
		t.Error("non-image non-string content should not trigger vision")
	}

	// Content that stringifies to something containing "image_url".
	type weirdContent struct{ image_url string }
	req2 := &providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "user", Content: weirdContent{image_url: "http://x/y.png"}},
		},
	}
	// fmt.Sprintf of the struct will contain "image_url" in the field repr.
	// Just check it doesn't panic; the result may or may not be true depending on fmt output.
	_ = requestHasVision(req2)
}

// TestPartsHaveImageNoMatch exercises the partsHaveImage path where no image found.
func TestPartsHaveImageNoMatch(t *testing.T) {
	parts := []any{
		"not a map",
		map[string]any{"type": "text", "text": "hello"},
		42,
	}
	if partsHaveImage(parts) {
		t.Error("partsHaveImage should return false for non-image parts")
	}
}

// TestPartsHaveImageMatchesImageType exercises the "image" type variant.
func TestPartsHaveImageMatchesImageType(t *testing.T) {
	parts := []any{
		map[string]any{"type": "image", "source": map[string]any{"url": "u"}},
	}
	if !partsHaveImage(parts) {
		t.Error("partsHaveImage should return true for 'image' type")
	}
}

// TestOrderedStepsDefaultBranch exercises the default strategy path in orderedSteps.
func TestOrderedStepsDefaultBranch(t *testing.T) {
	sel := &comboSelector{}
	steps := []ComboStep{
		{Provider: providers.ProviderAnthropic, Model: "claude-sonnet-4"},
		{Provider: providers.ProviderOpenAI, Model: "gpt-4o"},
	}
	ordered, idx := sel.orderedSteps("unknown_strategy", steps, msgReq("hi"))
	if len(ordered) != len(steps) {
		t.Errorf("orderedSteps default len = %d, want %d", len(ordered), len(steps))
	}
	if idx != -1 {
		t.Errorf("orderedSteps default selectIdx = %d, want -1", idx)
	}
}

// TestSanitizeRefreshReasonNilErr exercises the nil error branch in sanitizeRefreshReason.
func TestSanitizeRefreshReasonNilErr(t *testing.T) {
	got := sanitizeRefreshReason(nil)
	if got != "" {
		t.Errorf("sanitizeRefreshReason(nil) = %q, want empty", got)
	}
}

// TestSanitizeRefreshReasonTruncatesLong exercises the >200 rune truncation.
func TestSanitizeRefreshReasonTruncatesLong(t *testing.T) {
	long := strings.Repeat("e", 300)
	got := sanitizeRefreshReason(fmt.Errorf("%s", long))
	if len([]rune(got)) > 200 {
		t.Errorf("sanitizeRefreshReason did not truncate: got len %d", len([]rune(got)))
	}
}
