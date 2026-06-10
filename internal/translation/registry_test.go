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

func TestTranslateRequestPipeline(t *testing.T) {
	reg := NewRegistry()

	var order []string
	reg.Register(FormatClaude, FormatOpenAI,
		func(model string, body map[string]any, stream bool) (map[string]any, error) {
			order = append(order, "claude->openai")
			body["via1"] = true
			return body, nil
		}, nil)
	reg.Register(FormatOpenAI, FormatGemini,
		func(model string, body map[string]any, stream bool) (map[string]any, error) {
			order = append(order, "openai->gemini")
			body["via2"] = true
			return body, nil
		}, nil)

	body := map[string]any{"x": 1}
	out, err := reg.TranslateRequest(FormatClaude, FormatGemini, "m", body, false)
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	if len(order) != 2 || order[0] != "claude->openai" || order[1] != "openai->gemini" {
		t.Errorf("order = %v", order)
	}
	if out["via1"] != true || out["via2"] != true {
		t.Errorf("pipeline did not mutate body: %v", out)
	}
}

func TestTranslateRequestSameFormatSkips(t *testing.T) {
	reg := NewRegistry()

	called := false
	reg.Register(FormatOpenAI, FormatClaude,
		func(model string, body map[string]any, stream bool) (map[string]any, error) {
			called = true
			return body, nil
		}, nil)

	body := map[string]any{"x": 1}
	out, err := reg.TranslateRequest(FormatOpenAI, FormatOpenAI, "m", body, false)
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	if called {
		t.Error("same-format translation should not invoke any translator")
	}
	if out["x"] != 1 {
		t.Errorf("body mutated: %v", out)
	}
}

func TestTranslateResponseFanOut(t *testing.T) {
	reg := NewRegistry()

	reg.Register(FormatOpenAI, FormatClaude, nil,
		func(chunk map[string]any, state *StreamState) ([]map[string]any, error) {
			return []map[string]any{
				{"from": chunk["id"], "n": 1},
				{"from": chunk["id"], "n": 2},
			}, nil
		})

	chunks, err := reg.TranslateResponse(FormatOpenAI, FormatClaude, map[string]any{"id": "a"}, nil)
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	if len(chunks) != 2 {
		t.Fatalf("len(chunks) = %d, want 2", len(chunks))
	}
	if chunks[0]["n"] != 1 || chunks[1]["n"] != 2 {
		t.Errorf("chunks = %v", chunks)
	}
}
