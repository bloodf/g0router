package admin

import (
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

func TestPostLocaleSetsCookie(t *testing.T) {
	env := newTestEnv(t)

	status, envl := call(t, env.handlers.PostLocale, "POST", "/api/locale", `{"locale":"pt-BR"}`, nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}

	data := dataField[map[string]any](t, envl)
	if data["locale"] != "pt-BR" {
		t.Fatalf("data.locale = %v, want pt-BR", data["locale"])
	}
	if string(envl["error"]) != "null" {
		t.Fatalf("error = %s, want null", envl["error"])
	}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetRequestURI("/api/locale")
	ctx.Request.SetBody([]byte(`{"locale":"pt-BR"}`))
	env.handlers.PostLocale(&ctx)

	setCookie := string(ctx.Response.Header.Peek("Set-Cookie"))
	if !strings.Contains(setCookie, "locale=pt-BR") {
		t.Fatalf("Set-Cookie missing locale=pt-BR: %q", setCookie)
	}
	if !strings.Contains(setCookie, "Path=/") {
		t.Fatalf("Set-Cookie missing Path=/: %q", setCookie)
	}
	if !strings.Contains(setCookie, "SameSite=Lax") {
		t.Fatalf("Set-Cookie missing SameSite=Lax: %q", setCookie)
	}
	if strings.Contains(setCookie, "HttpOnly") {
		t.Fatalf("Set-Cookie must NOT be HttpOnly: %q", setCookie)
	}
}

func TestPostLocaleRejectsUnknown(t *testing.T) {
	env := newTestEnv(t)

	status, envl := call(t, env.handlers.PostLocale, "POST", "/api/locale", `{"locale":"xx-XX"}`, nil, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", status)
	}
	if string(envl["data"]) != "null" {
		t.Fatalf("data = %s, want null", envl["data"])
	}
	msg := errMessage(t, envl)
	if !strings.Contains(msg, "xx-XX") {
		t.Fatalf("error message = %q, want it to name xx-XX", msg)
	}

	status, _ = call(t, env.handlers.PostLocale, "POST", "/api/locale", ``, nil, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("empty body status = %d, want 400", status)
	}

	status, _ = call(t, env.handlers.PostLocale, "POST", "/api/locale", `not-json`, nil, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("malformed body status = %d, want 400", status)
	}
}
