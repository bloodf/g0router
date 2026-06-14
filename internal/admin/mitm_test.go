package admin

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

// fakeMitmProxy is a deterministic mitm.MitmProxy for the admin handler tests:
// no port bind, no TLS handshake.
type fakeMitmProxy struct {
	started bool
}

func (f *fakeMitmProxy) Start(addr string) error { f.started = true; return nil }
func (f *fakeMitmProxy) Stop() error             { f.started = false; return nil }
func (f *fakeMitmProxy) Running() bool           { return f.started }

func mitmActor(t *testing.T, env *testEnv) map[string]any {
	t.Helper()
	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	return map[string]any{userKey: admin}
}

func TestMitmStatusReturnsEnabledAndTwoTools(t *testing.T) {
	env := newTestEnv(t)
	env.handlers.SetMitmProxy(&fakeMitmProxy{})

	status, envl := call(t, env.handlers.MitmStatus, "GET", "/api/mitm/status", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("status = %d err = %q", status, errMessage(t, envl))
	}
	body := dataField[struct {
		Enabled bool             `json:"enabled"`
		Tools   []map[string]any `json:"tools"`
	}](t, envl)
	if body.Enabled {
		t.Fatalf("default enabled = true, want false")
	}
	if len(body.Tools) != 2 {
		t.Fatalf("tools len = %d, want 2: %v", len(body.Tools), body.Tools)
	}
	// Canonical 5-field DTO; never any key material.
	for _, tool := range body.Tools {
		for _, f := range []string{"id", "name", "enabled", "dns_override", "status"} {
			if _, ok := tool[f]; !ok {
				t.Fatalf("tool missing field %q: %v", f, tool)
			}
		}
		if _, ok := tool["key"]; ok {
			t.Fatalf("tool leaked a key field: %v", tool)
		}
	}
}

func TestMitmToggleFlipsAndAudits(t *testing.T) {
	env := newTestEnv(t)
	env.handlers.SetMitmProxy(&fakeMitmProxy{})

	status, envl := call(t, env.handlers.MitmToggle, "POST", "/api/mitm/toggle", "",
		mitmActor(t, env), nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("toggle status = %d err = %q", status, errMessage(t, envl))
	}
	res := dataField[struct {
		Enabled bool `json:"enabled"`
	}](t, envl)
	if !res.Enabled {
		t.Fatalf("toggle enabled = false, want true")
	}
	if got, _ := env.store.GetMitmEnabled(); !got {
		t.Fatalf("global flag not persisted")
	}

	entries, err := env.store.ListAuditEntries(10)
	if err != nil {
		t.Fatalf("ListAuditEntries: %v", err)
	}
	found := false
	for _, e := range entries {
		if strings.Contains(e.Action, "mitm") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no mitm audit entry recorded: %+v", entries)
	}
}

func TestMitmToolToggleFlipsAndDerivesStatus(t *testing.T) {
	env := newTestEnv(t)
	env.handlers.SetMitmProxy(&fakeMitmProxy{})

	// mitm-2 starts disabled; toggling enables it (status active).
	uv := mitmActor(t, env)
	uv["id"] = "mitm-2"
	status, envl := call(t, env.handlers.MitmToolToggle, "POST", "/api/mitm/tools/mitm-2", "",
		uv, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("tool toggle status = %d err = %q", status, errMessage(t, envl))
	}
	tool := dataField[map[string]any](t, envl)
	if tool["enabled"] != true {
		t.Fatalf("toggled tool enabled = %v, want true", tool["enabled"])
	}
	if tool["status"] != "active" {
		t.Fatalf("toggled tool status = %v, want active", tool["status"])
	}
}

func TestMitmToolToggleUnknownIDReturns404(t *testing.T) {
	env := newTestEnv(t)
	env.handlers.SetMitmProxy(&fakeMitmProxy{})

	uv := mitmActor(t, env)
	uv["id"] = "nope"
	status, _ := call(t, env.handlers.MitmToolToggle, "POST", "/api/mitm/tools/nope", "",
		uv, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("unknown tool toggle status = %d, want 404", status)
	}
}

func TestMitmCACertServesRawPEM(t *testing.T) {
	env := newTestEnv(t)
	env.handlers.SetMitmProxy(&fakeMitmProxy{})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/api/mitm/ca-cert")
	env.handlers.MitmCACert(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("ca-cert status = %d", ctx.Response.StatusCode())
	}
	ct := string(ctx.Response.Header.ContentType())
	if ct != "application/x-pem-file" {
		t.Fatalf("ca-cert content-type = %q, want application/x-pem-file", ct)
	}
	body := ctx.Response.Body()
	if !bytes.HasPrefix(body, []byte("-----BEGIN CERTIFICATE-----")) {
		t.Fatalf("ca-cert body is not a CERTIFICATE PEM block:\n%s", body)
	}
	if bytes.Contains(body, []byte("PRIVATE KEY")) {
		t.Fatalf("ca-cert body leaked a PRIVATE KEY block")
	}
	block, _ := pem.Decode(body)
	if block == nil {
		t.Fatalf("ca-cert body does not decode as PEM")
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		t.Fatalf("ca-cert body does not parse as a certificate: %v", err)
	}
}

func TestMitmResponsesNeverLeakKeyMaterial(t *testing.T) {
	env := newTestEnv(t)
	env.handlers.SetMitmProxy(&fakeMitmProxy{})

	checks := []struct {
		name    string
		handler fasthttp.RequestHandler
		method  string
		uri     string
		uv      map[string]any
	}{
		{"status", env.handlers.MitmStatus, "GET", "/api/mitm/status", nil},
		{"toggle", env.handlers.MitmToggle, "POST", "/api/mitm/toggle", mitmActor(t, env)},
		{"tool", env.handlers.MitmToolToggle, "POST", "/api/mitm/tools/mitm-1", func() map[string]any {
			uv := mitmActor(t, env)
			uv["id"] = "mitm-1"
			return uv
		}()},
	}
	for _, c := range checks {
		var ctx fasthttp.RequestCtx
		ctx.Request.Header.SetMethod(c.method)
		ctx.Request.SetRequestURI(c.uri)
		for k, v := range c.uv {
			ctx.SetUserValue(k, v)
		}
		c.handler(&ctx)
		body := ctx.Response.Body()
		if bytes.Contains(body, []byte("PRIVATE KEY")) {
			t.Fatalf("%s response leaked PRIVATE KEY: %s", c.name, body)
		}
		for _, tok := range []string{"\"key\"", "\"private_key\"", "\"ca_key\""} {
			if bytes.Contains(body, []byte(tok)) {
				t.Fatalf("%s response leaked key field %s: %s", c.name, tok, body)
			}
		}
	}
}
