package translation

import (
	"errors"
	"testing"
)

func TestRegistryRegisterLookup(t *testing.T) {
	reg := NewRegistry()

	var called bool
	rt := func(model string, body map[string]any, stream bool) (map[string]any, error) {
		called = true
		return body, nil
	}
	reg.Register(FormatClaude, FormatOpenAI, rt, nil)

	fn := reg.RequestTranslatorFor(FormatClaude, FormatOpenAI)
	if fn == nil {
		t.Fatal("expected registered request translator")
	}
	if _, err := fn("", nil, false); err != nil {
		t.Fatalf("translator error: %v", err)
	}
	if !called {
		t.Error("request translator was not called")
	}

	if reg.ResponseTranslatorFor(FormatClaude, FormatOpenAI) != nil {
		t.Error("expected no response translator")
	}
}

func TestNeedsTranslation(t *testing.T) {
	reg := NewRegistry()
	if !reg.NeedsTranslation(FormatClaude, FormatOpenAI) {
		t.Error("different formats should need translation")
	}
	if reg.NeedsTranslation(FormatOpenAI, FormatOpenAI) {
		t.Error("same format should not need translation")
	}
}

func TestRegistryRequestTranslatorForMissing(t *testing.T) {
	reg := NewRegistry()
	if fn := reg.RequestTranslatorFor(FormatClaude, FormatOpenAI); fn != nil {
		t.Error("expected nil for unregistered translator")
	}
}

func TestRegistryResponseTranslatorReturnsError(t *testing.T) {
	reg := NewRegistry()
	wantErr := errors.New("boom")
	reg.Register(FormatOpenAI, FormatClaude, nil, func(chunk map[string]any, state *StreamState) ([]map[string]any, error) {
		return nil, wantErr
	})

	fn := reg.ResponseTranslatorFor(FormatOpenAI, FormatClaude)
	if fn == nil {
		t.Fatal("expected registered response translator")
	}
	_, err := fn(nil, nil)
	if err != wantErr {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}
