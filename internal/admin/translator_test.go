package admin

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

func TestTranslatorLoadContainsGPT4o(t *testing.T) {
	env := newTestEnv(t)
	token := loginToken(t, env)

	status, envl := call(t, env.handlers.RequireSession(env.handlers.TranslatorLoad),
		"GET", "/api/translator/load", "", nil,
		map[string]string{"Authorization": "Bearer " + token})
	if status != fasthttp.StatusOK {
		t.Fatalf("load status = %d, err = %q", status, errMessage(t, envl))
	}
	data := dataField[map[string]any](t, envl)
	payload, _ := data["payload"].(string)
	if !strings.Contains(payload, "gpt-4o") {
		t.Fatalf("load payload missing gpt-4o: %q", payload)
	}
	if _, ok := data["file"]; !ok {
		t.Fatalf("load data missing file: %v", data)
	}
}

func TestTranslatorTranslateMarkedTranslated(t *testing.T) {
	env := newTestEnv(t)
	token := loginToken(t, env)

	in := map[string]any{
		"model":    "gpt-4o",
		"messages": []any{map[string]any{"role": "user", "content": "hi"}},
		"stream":   false,
	}
	inJSON, _ := json.Marshal(in)
	body, _ := json.Marshal(map[string]any{
		"from":    "openai",
		"to":      "claude",
		"model":   "gpt-4o",
		"payload": string(inJSON),
	})

	status, envl := call(t, env.handlers.RequireSession(env.handlers.TranslatorTranslate),
		"POST", "/api/translator/translate", string(body), nil,
		map[string]string{"Authorization": "Bearer " + token})
	if status != fasthttp.StatusOK {
		t.Fatalf("translate status = %d, err = %q", status, errMessage(t, envl))
	}
	data := dataField[map[string]any](t, envl)
	if data["translated"] != true {
		t.Fatalf("translate data missing translated marker: %v", data)
	}
	payload, _ := data["payload"].(string)
	if payload == "" {
		t.Fatalf("translate payload empty: %v", data)
	}
	// The openai->claude transform adds max_tokens; assert the transform changed
	// the body (it is no longer the verbatim input).
	if payload == string(inJSON) {
		t.Fatalf("translate payload unchanged from input: %q", payload)
	}
	if !strings.Contains(payload, "max_tokens") {
		t.Fatalf("translate payload not claude-shaped (no max_tokens): %q", payload)
	}
}

func TestTranslatorTranslateBadPayload400(t *testing.T) {
	env := newTestEnv(t)
	token := loginToken(t, env)

	body, _ := json.Marshal(map[string]any{
		"from":    "openai",
		"to":      "claude",
		"payload": "{not valid json",
	})
	status, _ := call(t, env.handlers.RequireSession(env.handlers.TranslatorTranslate),
		"POST", "/api/translator/translate", string(body), nil,
		map[string]string{"Authorization": "Bearer " + token})
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("bad payload status = %d, want 400", status)
	}
}
