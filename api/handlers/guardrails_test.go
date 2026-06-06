package handlers

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeGuardrailsStore struct {
	getCfg    store.GuardrailsConfig
	getErr    error
	updateErr error
}

func (f *fakeGuardrailsStore) GetGuardrailsConfig() (store.GuardrailsConfig, error) {
	return f.getCfg, f.getErr
}

func (f *fakeGuardrailsStore) UpdateGuardrailsConfig(cfg store.GuardrailsConfig) error {
	return f.updateErr
}

func TestGuardrailsGet(t *testing.T) {
	s := &fakeGuardrailsStore{
		getCfg: store.GuardrailsConfig{
			GuardrailsEnabled:   true,
			GuardrailsBlocklist: []string{"bad"},
			PIIRedactionEnabled: true,
			PIIRedactionTypes:   []string{"email"},
		},
	}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Guardrails(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp guardrailsConfigView
	decodeJSON(t, body, &resp)
	if !resp.GuardrailsEnabled {
		t.Error("expected guardrails_enabled true")
	}
	if len(resp.GuardrailsBlocklist) != 1 || resp.GuardrailsBlocklist[0] != "bad" {
		t.Errorf("blocklist = %v, want [bad]", resp.GuardrailsBlocklist)
	}
}

func TestGuardrailsPut(t *testing.T) {
	s := &fakeGuardrailsStore{}

	ctx, body := runHandler(t, fasthttp.MethodPut, `{"guardrails_enabled":true,"guardrails_blocklist":["bad"],"pii_redaction_enabled":true,"pii_redaction_types":["email"]}`, func(ctx *fasthttp.RequestCtx) {
		Guardrails(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp guardrailsConfigView
	decodeJSON(t, body, &resp)
	if !resp.GuardrailsEnabled {
		t.Error("expected guardrails_enabled true")
	}
}

func TestGuardrailsTestEndpointBlocked(t *testing.T) {
	s := &fakeGuardrailsStore{
		getCfg: store.GuardrailsConfig{
			GuardrailsEnabled:   true,
			GuardrailsBlocklist: []string{"badword"},
			PIIRedactionEnabled: true,
			PIIRedactionTypes:   []string{"email"},
		},
	}

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"prompt":"hello badword and a@b.com"}`, func(ctx *fasthttp.RequestCtx) {
		GuardrailsTest(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp guardrailsTestResponse
	decodeJSON(t, body, &resp)
	if !resp.Blocked {
		t.Error("expected blocked true")
	}
	if len(resp.Matches) != 1 || resp.Matches[0] != "badword" {
		t.Errorf("matches = %v, want [badword]", resp.Matches)
	}
	if resp.RedactedPrompt != "hello badword and [REDACTED:email]" {
		t.Errorf("redacted_prompt = %q, want hello badword and [REDACTED:email]", resp.RedactedPrompt)
	}
}

func TestGuardrailsTestEndpointClean(t *testing.T) {
	s := &fakeGuardrailsStore{
		getCfg: store.GuardrailsConfig{
			GuardrailsEnabled:   true,
			GuardrailsBlocklist: []string{"badword"},
			PIIRedactionEnabled: false,
		},
	}

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"prompt":"hello world"}`, func(ctx *fasthttp.RequestCtx) {
		GuardrailsTest(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp guardrailsTestResponse
	decodeJSON(t, body, &resp)
	if resp.Blocked {
		t.Error("expected blocked false")
	}
	if len(resp.Matches) != 0 {
		t.Errorf("matches = %v, want empty", resp.Matches)
	}
	if resp.RedactedPrompt != "hello world" {
		t.Errorf("redacted_prompt = %q, want hello world", resp.RedactedPrompt)
	}
}

func TestGuardrailsStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Guardrails(ctx, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestGuardrailsTestStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"prompt":"x"}`, func(ctx *fasthttp.RequestCtx) {
		GuardrailsTest(ctx, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestGuardrailsGetError(t *testing.T) {
	s := &fakeGuardrailsStore{getErr: errors.New("boom")}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Guardrails(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestGuardrailsPutError(t *testing.T) {
	s := &fakeGuardrailsStore{updateErr: errors.New("boom")}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"guardrails_enabled":true}`, func(ctx *fasthttp.RequestCtx) {
		Guardrails(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestGuardrailsPutInvalidJSON(t *testing.T) {
	s := &fakeGuardrailsStore{}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `not json`, func(ctx *fasthttp.RequestCtx) {
		Guardrails(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestGuardrailsTestInvalidJSON(t *testing.T) {
	s := &fakeGuardrailsStore{}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `not json`, func(ctx *fasthttp.RequestCtx) {
		GuardrailsTest(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestGuardrailsTestGetError(t *testing.T) {
	s := &fakeGuardrailsStore{getErr: errors.New("boom")}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"prompt":"x"}`, func(ctx *fasthttp.RequestCtx) {
		GuardrailsTest(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestGuardrailsMethodNotAllowed(t *testing.T) {
	s := &fakeGuardrailsStore{}
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		Guardrails(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}

func TestGuardrailsTestMethodNotAllowed(t *testing.T) {
	s := &fakeGuardrailsStore{}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		GuardrailsTest(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}
