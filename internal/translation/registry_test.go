package translation

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestRegistryRegisterLookupMechanics(t *testing.T) {
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

func TestRegistryRegisterLookup(t *testing.T) {
	reg := NewRegistry()

	var called bool
	override := func(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
		called = true
		return body, nil
	}
	reg.Register(FormatClaude, FormatOpenAI, override, nil)

	fn := reg.RequestTranslatorFor(FormatClaude, FormatOpenAI)
	if fn == nil {
		t.Fatal("expected registered request translator")
	}
	if _, err := fn("", nil, false, nil); err != nil {
		t.Fatalf("translator error: %v", err)
	}
	if !called {
		t.Error("override request translator was not called")
	}

	if reg.ResponseTranslatorFor(FormatClaude, FormatGemini) != nil {
		t.Error("expected no response translator for unwired pair")
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
	// Claude body shape: system extracted from the system message, user
	// message preserved, tools converted to {name, input_schema}, tool_choice
	// mapped to the Claude object form.
	sys, ok := out["system"].([]any)
	if !ok || len(sys) == 0 {
		t.Fatalf("expected extracted system blocks: %v", out["system"])
	}
	sysTexts := ""
	for _, b := range sys {
		if bm, ok := b.(map[string]any); ok {
			if txt, _ := bm["text"].(string); txt != "" {
				sysTexts += txt + "\n"
			}
		}
	}
	if !strings.Contains(sysTexts, "You are helpful.") {
		t.Errorf("system blocks missing extracted system message: %q", sysTexts)
	}
	msgs, ok := out["messages"].([]any)
	if !ok || len(msgs) != 1 {
		t.Fatalf("expected single user message in claude body: %v", out["messages"])
	}
	userMsg := msgs[0].(map[string]any)
	if userMsg["role"] != "user" {
		t.Errorf("messages[0].role = %v, want user", userMsg["role"])
	}
	tools, ok := out["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("expected one converted tool: %v", out["tools"])
	}
	tool0 := tools[0].(map[string]any)
	if tool0["name"] != "Read" {
		t.Errorf("tools[0].name = %v, want Read", tool0["name"])
	}
	if _, ok := tool0["input_schema"]; !ok {
		t.Errorf("tools[0] missing input_schema: %v", tool0)
	}
	tcOut, ok := out["tool_choice"].(map[string]any)
	if !ok || tcOut["type"] != "auto" {
		t.Errorf("tool_choice = %v, want {type:auto}", out["tool_choice"])
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

func TestResponseAliasesUseGeminiTranslator(t *testing.T) {
	// PAR-TRANS-043: gemini-cli, vertex and antigravity responses are aliases
	// of the gemini→openai translator — identity, not just non-nil wiring.
	reg := NewRegistry()
	want := reflect.ValueOf(ResponseTranslator(geminiToOpenAIResponse)).Pointer()
	for _, from := range []Format{FormatGeminiCLI, FormatVertex, FormatAntigravity} {
		got := reg.ResponseTranslatorFor(from, FormatOpenAI)
		if got == nil {
			t.Errorf("%s->openai response translator not wired", from)
			continue
		}
		if reflect.ValueOf(got).Pointer() != want {
			t.Errorf("%s->openai response translator is not geminiToOpenAIResponse", from)
		}
	}
}

func TestNewRegistryWiresResponsesPair(t *testing.T) {
	reg := NewRegistry()

	// All four lookups must be non-nil
	if reg.RequestTranslatorFor(FormatOpenAIResponses, FormatOpenAI) == nil {
		t.Error("NewRegistry must wire responses->openai request translator")
	}
	if reg.ResponseTranslatorFor(FormatOpenAIResponses, FormatOpenAI) == nil {
		t.Error("NewRegistry must wire responses->openai response translator")
	}
	if reg.RequestTranslatorFor(FormatOpenAI, FormatOpenAIResponses) == nil {
		t.Error("NewRegistry must wire openai->responses request translator")
	}
	if reg.ResponseTranslatorFor(FormatOpenAI, FormatOpenAIResponses) == nil {
		t.Error("NewRegistry must wire openai->responses response translator")
	}

	// Identity checks (same technique as TestResponseAliasesUseGeminiTranslator)
	wantReq1 := reflect.ValueOf(RequestTranslator(responsesToOpenAIRequest)).Pointer()
	gotReq1 := reflect.ValueOf(reg.RequestTranslatorFor(FormatOpenAIResponses, FormatOpenAI)).Pointer()
	if gotReq1 != wantReq1 {
		t.Error("responses->openai request translator is not responsesToOpenAIRequest")
	}

	wantResp1 := reflect.ValueOf(ResponseTranslator(responsesToOpenAIResponse)).Pointer()
	gotResp1 := reflect.ValueOf(reg.ResponseTranslatorFor(FormatOpenAIResponses, FormatOpenAI)).Pointer()
	if gotResp1 != wantResp1 {
		t.Error("responses->openai response translator is not responsesToOpenAIResponse")
	}

	wantReq2 := reflect.ValueOf(RequestTranslator(openaiToResponsesRequest)).Pointer()
	gotReq2 := reflect.ValueOf(reg.RequestTranslatorFor(FormatOpenAI, FormatOpenAIResponses)).Pointer()
	if gotReq2 != wantReq2 {
		t.Error("openai->responses request translator is not openaiToResponsesRequest")
	}

	wantResp2 := reflect.ValueOf(ResponseTranslator(openaiToResponsesResponse)).Pointer()
	gotResp2 := reflect.ValueOf(reg.ResponseTranslatorFor(FormatOpenAI, FormatOpenAIResponses)).Pointer()
	if gotResp2 != wantResp2 {
		t.Error("openai->responses response translator is not openaiToResponsesResponse")
	}
}

func TestNewRegistryWiresOllamaPair(t *testing.T) {
	reg := NewRegistry()

	if reg.RequestTranslatorFor(FormatOpenAI, FormatOllama) == nil {
		t.Error("NewRegistry must wire openai->ollama request translator")
	}
	if reg.ResponseTranslatorFor(FormatOllama, FormatOpenAI) == nil {
		t.Error("NewRegistry must wire ollama->openai response translator")
	}

	wantReq := reflect.ValueOf(RequestTranslator(openaiToOllamaRequest)).Pointer()
	gotReq := reflect.ValueOf(reg.RequestTranslatorFor(FormatOpenAI, FormatOllama)).Pointer()
	if gotReq != wantReq {
		t.Error("openai->ollama request translator is not openaiToOllamaRequest")
	}

	wantResp := reflect.ValueOf(ResponseTranslator(ollamaToOpenAIResponse)).Pointer()
	gotResp := reflect.ValueOf(reg.ResponseTranslatorFor(FormatOllama, FormatOpenAI)).Pointer()
	if gotResp != wantResp {
		t.Error("ollama->openai response translator is not ollamaToOpenAIResponse")
	}
}

func TestNewRegistryWiresCommandCodePair(t *testing.T) {
	reg := NewRegistry()

	if reg.RequestTranslatorFor(FormatOpenAI, FormatCommandCode) == nil {
		t.Error("NewRegistry must wire openai->commandcode request translator")
	}
	if reg.ResponseTranslatorFor(FormatCommandCode, FormatOpenAI) == nil {
		t.Error("NewRegistry must wire commandcode->openai response translator")
	}

	wantReq := reflect.ValueOf(RequestTranslator(openaiToCommandCodeRequest)).Pointer()
	gotReq := reflect.ValueOf(reg.RequestTranslatorFor(FormatOpenAI, FormatCommandCode)).Pointer()
	if gotReq != wantReq {
		t.Error("openai->commandcode request translator is not openaiToCommandCodeRequest")
	}

	wantResp := reflect.ValueOf(ResponseTranslator(commandcodeToOpenAIResponse)).Pointer()
	gotResp := reflect.ValueOf(reg.ResponseTranslatorFor(FormatCommandCode, FormatOpenAI)).Pointer()
	if gotResp != wantResp {
		t.Error("commandcode->openai response translator is not commandcodeToOpenAIResponse")
	}
}

func TestNewRegistryWiresKiroPair(t *testing.T) {
	reg := NewRegistry()

	if reg.RequestTranslatorFor(FormatOpenAI, FormatKiro) == nil {
		t.Error("NewRegistry must wire openai->kiro request translator")
	}
	if reg.ResponseTranslatorFor(FormatKiro, FormatOpenAI) == nil {
		t.Error("NewRegistry must wire kiro->openai response translator")
	}

	wantReq := reflect.ValueOf(RequestTranslator(buildKiroPayload)).Pointer()
	gotReq := reflect.ValueOf(reg.RequestTranslatorFor(FormatOpenAI, FormatKiro)).Pointer()
	if gotReq != wantReq {
		t.Error("openai->kiro request translator is not buildKiroPayload")
	}

	wantResp := reflect.ValueOf(ResponseTranslator(kiroToOpenAIResponse)).Pointer()
	gotResp := reflect.ValueOf(reg.ResponseTranslatorFor(FormatKiro, FormatOpenAI)).Pointer()
	if gotResp != wantResp {
		t.Error("kiro->openai response translator is not kiroToOpenAIResponse")
	}
}

func TestNewRegistryWiresCursorPair(t *testing.T) {
	reg := NewRegistry()

	if reg.RequestTranslatorFor(FormatOpenAI, FormatCursor) == nil {
		t.Error("NewRegistry must wire openai->cursor request translator")
	}
	if reg.ResponseTranslatorFor(FormatCursor, FormatOpenAI) == nil {
		t.Error("NewRegistry must wire cursor->openai response translator")
	}

	wantReq := reflect.ValueOf(RequestTranslator(buildCursorRequest)).Pointer()
	gotReq := reflect.ValueOf(reg.RequestTranslatorFor(FormatOpenAI, FormatCursor)).Pointer()
	if gotReq != wantReq {
		t.Error("openai->cursor request translator is not buildCursorRequest")
	}

	wantResp := reflect.ValueOf(ResponseTranslator(cursorToOpenAIResponse)).Pointer()
	gotResp := reflect.ValueOf(reg.ResponseTranslatorFor(FormatCursor, FormatOpenAI)).Pointer()
	if gotResp != wantResp {
		t.Error("cursor->openai response translator is not cursorToOpenAIResponse")
	}
}

func TestRegistryWiresGeminiClientRequest(t *testing.T) {
	reg := NewRegistry()

	req1 := reg.RequestTranslatorFor(FormatGemini, FormatOpenAI)
	if req1 == nil {
		t.Fatal("expected gemini->openai request translator")
	}
	want1 := reflect.ValueOf(RequestTranslator(geminiToOpenAIRequest)).Pointer()
	got1 := reflect.ValueOf(req1).Pointer()
	if got1 != want1 {
		t.Error("gemini->openai request translator is not geminiToOpenAIRequest")
	}

	req2 := reg.RequestTranslatorFor(FormatGeminiCLI, FormatOpenAI)
	if req2 == nil {
		t.Fatal("expected gemini-cli->openai request translator")
	}
	want2 := reflect.ValueOf(RequestTranslator(geminiToOpenAIRequest)).Pointer()
	got2 := reflect.ValueOf(req2).Pointer()
	if got2 != want2 {
		t.Error("gemini-cli->openai request translator is not geminiToOpenAIRequest")
	}

	// Response translators on those pairs must remain unchanged.
	if reg.ResponseTranslatorFor(FormatGemini, FormatOpenAI) == nil {
		t.Error("gemini->openai response translator should still be wired")
	}
	if reg.ResponseTranslatorFor(FormatGeminiCLI, FormatOpenAI) == nil {
		t.Error("gemini-cli->openai response translator should still be wired")
	}
}
