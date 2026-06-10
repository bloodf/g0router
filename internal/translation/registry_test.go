package translation

import (
	"errors"
	"testing"
)

func TestRegistryRegisterLookup(t *testing.T) {
	reg := &Registry{
		request:  make(map[string]RequestTranslator),
		response: make(map[string]ResponseTranslator),
	}

	var called bool
	rt := func(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
		called = true
		return body, nil
	}
	reg.Register(FormatClaude, FormatOpenAI, rt, nil)

	fn := reg.RequestTranslatorFor(FormatClaude, FormatOpenAI)
	if fn == nil {
		t.Fatal("expected registered request translator")
	}
	if _, err := fn("", nil, false, nil); err != nil {
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

func TestNewRegistryWiresClaudeRequest(t *testing.T) {
	reg := NewRegistry()
	if reg.RequestTranslatorFor(FormatClaude, FormatOpenAI) == nil {
		t.Error("NewRegistry must wire claude->openai request translator")
	}
}

func TestNewRegistryWiresClaudeResponse(t *testing.T) {
	reg := NewRegistry()
	if reg.ResponseTranslatorFor(FormatOpenAI, FormatClaude) == nil {
		t.Error("NewRegistry must wire openai->claude response translator")
	}
}

func TestRegistryRequestTranslatorForMissing(t *testing.T) {
	reg := &Registry{request: make(map[string]RequestTranslator), response: make(map[string]ResponseTranslator)}
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
		func(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
			order = append(order, "claude->openai")
			body["via1"] = true
			return body, nil
		}, nil)
	reg.Register(FormatOpenAI, FormatGemini,
		func(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
			order = append(order, "openai->gemini")
			body["via2"] = true
			return body, nil
		}, nil)

	body := map[string]any{"x": 1}
	out, err := reg.TranslateRequest(FormatClaude, FormatGemini, "m", body, false, nil)
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
		func(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
			called = true
			return body, nil
		}, nil)

	body := map[string]any{"x": 1}
	out, err := reg.TranslateRequest(FormatOpenAI, FormatOpenAI, "m", body, false, nil)
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

func TestRegistryClaudeRoundTripViaPipeline(t *testing.T) {
	reg := NewRegistry()
	body := map[string]any{
		"model": "claude-3-opus",
		"messages": []any{
			map[string]any{"role": "system", "content": "You are helpful."},
			map[string]any{"role": "user", "content": "hi"},
		},
		"tools": []any{
			map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        "Read",
					"description": "read file",
					"parameters":  map[string]any{"type": "object", "properties": map[string]any{}},
				},
			},
		},
		"tool_choice": "auto",
	}
	out, err := reg.TranslateRequest(FormatOpenAI, FormatClaude, "claude-3-opus", body, false, nil)
	if err != nil {
		t.Fatalf("TranslateRequest: %v", err)
	}
	if _, ok := out["messages"]; !ok {
		t.Fatalf("expected messages in claude body: %v", out)
	}
	if _, ok := out["tools"]; !ok {
		t.Fatalf("expected tools in claude body: %v", out)
	}

	state := NewStreamState()
	events := []map[string]any{
		{"type": "message_start", "message": map[string]any{"id": "msg_1", "model": "claude-3-opus"}},
		{"type": "content_block_delta", "index": 0, "delta": map[string]any{"type": "text_delta", "text": "ok"}},
		{"type": "content_block_stop", "index": 0},
		{"type": "content_block_start", "index": 1, "content_block": map[string]any{"type": "tool_use", "id": "toolu_1", "name": "Read"}},
		{"type": "content_block_delta", "index": 1, "delta": map[string]any{"type": "input_json_delta", "partial_json": `{"file":`}},
		{"type": "content_block_delta", "index": 1, "delta": map[string]any{"type": "input_json_delta", "partial_json": `"a.txt"}`}},
		{"type": "content_block_stop", "index": 1},
		{"type": "message_delta", "delta": map[string]any{"stop_reason": "tool_use"}, "usage": map[string]any{"input_tokens": 1, "output_tokens": 1}},
		{"type": "message_stop"},
	}
	var last map[string]any
	toolArgs := ""
	sawToolCallStart := false
	for _, ev := range events {
		chunks, err := reg.TranslateResponse(FormatClaude, FormatOpenAI, ev, state)
		if err != nil {
			t.Fatalf("TranslateResponse: %v", err)
		}
		for _, chunk := range chunks {
			choices := chunk["choices"].([]any)
			delta := choices[0].(map[string]any)["delta"].(map[string]any)
			if tcs, ok := delta["tool_calls"].([]any); ok {
				tc := tcs[0].(map[string]any)
				fn := tc["function"].(map[string]any)
				if name, _ := fn["name"].(string); name == "Read" {
					sawToolCallStart = true
				}
				if args, _ := fn["arguments"].(string); args != "" {
					toolArgs += args
				}
			}
		}
		if len(chunks) > 0 {
			last = chunks[len(chunks)-1]
		}
	}
	if !sawToolCallStart {
		t.Error("expected a tool_calls chunk announcing tool Read")
	}
	if toolArgs != `{"file":"a.txt"}` {
		t.Errorf("accumulated tool arguments = %q, want %q", toolArgs, `{"file":"a.txt"}`)
	}
	if last == nil {
		t.Fatal("no response chunks")
	}
	choices := last["choices"].([]any)
	if fr := choices[0].(map[string]any)["finish_reason"]; fr != "tool_calls" {
		t.Fatalf("finish_reason = %v, want tool_calls", fr)
	}
}

func TestRegistryGeminiRoundTripViaPipeline(t *testing.T) {
	reg := NewRegistry()

	// Request direction: openai → gemini.
	body := map[string]any{
		"model": "gemini-pro",
		"messages": []any{
			map[string]any{"role": "system", "content": "You are helpful."},
			map[string]any{
				"role": "assistant",
				"content": "",
				"reasoning_content": "thinking...",
				"tool_calls": []any{
					map[string]any{
						"id":   "call-abc-123",
						"type": "function",
						"function": map[string]any{
							"name":      "get_weather",
							"arguments": `{"location":"NYC"}`,
						},
					},
				},
			},
			map[string]any{
				"role":         "tool",
				"tool_call_id": "call-abc-123",
				"content":      `{"temp":72}`,
			},
		},
		"tools": []any{
			map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        "get_weather",
					"description": "get weather",
					"parameters":  map[string]any{"type": "object", "properties": map[string]any{}},
				},
			},
		},
		"tool_choice": "auto",
	}
	reqOut, err := reg.TranslateRequest(FormatOpenAI, FormatGemini, "gemini-pro", body, false, nil)
	if err != nil {
		t.Fatalf("TranslateRequest: %v", err)
	}

	// Assert gemini body shape.
	if _, ok := reqOut["systemInstruction"]; !ok {
		t.Fatal("expected systemInstruction in gemini body")
	}
	contents, ok := reqOut["contents"].([]any)
	if !ok || len(contents) == 0 {
		t.Fatalf("expected contents in gemini body: %v", reqOut["contents"])
	}
	tools, ok := reqOut["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatalf("expected tools in gemini body: %v", reqOut["tools"])
	}
	if _, ok := reqOut["toolConfig"]; ok {
		t.Error("expected no toolConfig in gemini body")
	}

	// Response direction: gemini → openai.
	state := NewStreamState()
	events := []map[string]any{
		{
			"responseId":   "resp_1",
			"modelVersion": "gemini-1.5-pro",
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{"thought": true, "thoughtSignature": "sig", "text": "thinking..."},
						},
					},
				},
			},
		},
		{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "ok"}},
					},
				},
			},
		},
		{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{"functionCall": map[string]any{"name": "get_weather", "args": map[string]any{"location": "NYC"}}},
						},
					},
				},
			},
		},
		{
			"candidates": []any{
				map[string]any{
					"content":      map[string]any{"parts": []any{}},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(10),
				"candidatesTokenCount": float64(5),
				"totalTokenCount":      float64(15),
			},
		},
	}

	var last map[string]any
	for _, ev := range events {
		chunks, err := reg.TranslateResponse(FormatGemini, FormatOpenAI, ev, state)
		if err != nil {
			t.Fatalf("TranslateResponse: %v", err)
		}
		if len(chunks) > 0 {
			last = chunks[len(chunks)-1]
		}
	}
	if last == nil {
		t.Fatal("no response chunks")
	}
	choices := last["choices"].([]any)
	finishReason := choices[0].(map[string]any)["finish_reason"]
	if finishReason != "tool_calls" {
		t.Fatalf("expected finish_reason=tool_calls, got %v", finishReason)
	}
	if last["usage"] == nil {
		t.Fatal("expected usage on final chunk")
	}
}

func TestTranslateRequestForwardsCredentials(t *testing.T) {
	reg := &Registry{
		request:  make(map[string]RequestTranslator),
		response: make(map[string]ResponseTranslator),
	}

	var received map[string]any
	reg.Register(FormatClaude, FormatOpenAI, func(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
		received = credentials
		return body, nil
	}, nil)

	creds := map[string]any{"connectionId": "conn-123"}
	body := map[string]any{"x": 1}
	_, err := reg.TranslateRequest(FormatClaude, FormatOpenAI, "m", body, false, creds)
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	if received == nil {
		t.Fatal("translator did not receive credentials")
	}
	if received["connectionId"] != "conn-123" {
		t.Errorf("connectionId = %v, want conn-123", received["connectionId"])
	}
}

func TestRegistryGeminiCLIResponseUsesGeminiOpenAI(t *testing.T) {
	reg := NewRegistry()
	if reg.ResponseTranslatorFor(FormatGeminiCLI, FormatOpenAI) == nil {
		t.Error("NewRegistry must wire gemini-cli->openai response translator")
	}
}

func TestRegistryVertexResponseUsesGeminiOpenAI(t *testing.T) {
	reg := NewRegistry()
	if reg.ResponseTranslatorFor(FormatVertex, FormatOpenAI) == nil {
		t.Error("NewRegistry must wire vertex->openai response translator")
	}
}

func TestRegistryAntigravityResponseUsesGeminiOpenAI(t *testing.T) {
	reg := NewRegistry()
	if reg.ResponseTranslatorFor(FormatAntigravity, FormatOpenAI) == nil {
		t.Error("NewRegistry must wire antigravity->openai response translator")
	}
}
