package handlers

import (
	"errors"
	"testing"
)

func TestTranslateAnthropicMessagesRequestErrorPropagation(t *testing.T) {
	// Happy path: tools + tool_choice translate into the internal request.
	ok := `{"model":"m","messages":[{"role":"user","content":"hi"}],` +
		`"tools":[{"name":"f","input_schema":{"type":"object"}}],"tool_choice":{"type":"auto"}}`
	req, err := translateAnthropicMessagesRequest([]byte(ok))
	if err != nil {
		t.Fatalf("happy path: %v", err)
	}
	if len(req.Tools) != 1 || req.ToolChoice != "auto" {
		t.Fatalf("req tools=%v tool_choice=%v", req.Tools, req.ToolChoice)
	}

	// Unrepresentable tool -> error propagates from translateAnthropicTools.
	serverTool := `{"model":"m","messages":[],"tools":[{"type":"web_search","name":"ws"}]}`
	if _, err := translateAnthropicMessagesRequest([]byte(serverTool)); !errors.Is(err, errAnthropicTranslate) {
		t.Fatalf("server tool err = %v, want errAnthropicTranslate", err)
	}

	// Unknown tool_choice type -> error propagates from translateAnthropicToolChoice.
	badChoice := `{"model":"m","messages":[],"tool_choice":{"type":"banana"}}`
	if _, err := translateAnthropicMessagesRequest([]byte(badChoice)); !errors.Is(err, errAnthropicTranslate) {
		t.Fatalf("bad tool_choice err = %v, want errAnthropicTranslate", err)
	}

	// Malformed body -> json error.
	if _, err := translateAnthropicMessagesRequest([]byte(`{bad`)); err == nil {
		t.Fatalf("want json error for malformed body")
	}
}
