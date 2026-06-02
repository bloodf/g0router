package streaming

import "testing"

func TestResponsesAccumulatorTextDeltas(t *testing.T) {
	acc := NewResponsesAccumulator()

	acc.AddEvent(ResponseEvent{Type: "response.output_text.delta", Delta: "Hel"})
	acc.AddEvent(ResponseEvent{Type: "response.output_text.delta", Delta: "lo"})
	acc.AddEvent(ResponseEvent{
		Type: "response.completed",
		Response: &Response{
			ID:        "resp_123",
			Object:    "response",
			CreatedAt: 1700000000,
			Model:     "gpt-4o-mini",
			Status:    "completed",
		},
	})

	got := acc.Response()
	if got.ID != "resp_123" || got.Model != "gpt-4o-mini" || got.OutputText != "Hello" {
		t.Fatalf("response = %+v", got)
	}
}

func TestResponsesAccumulatorDoneTextReplacesDeltas(t *testing.T) {
	acc := NewResponsesAccumulator()

	acc.AddEvent(ResponseEvent{Type: "response.output_text.delta", Delta: "draft"})
	acc.AddEvent(ResponseEvent{Type: "response.output_text.done", Text: "final"})

	got := acc.Response()
	if got.OutputText != "final" {
		t.Fatalf("output text = %q", got.OutputText)
	}
}
