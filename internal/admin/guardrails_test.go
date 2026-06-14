package admin

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestGuardrailsGetDefaultConfig(t *testing.T) {
	env := newTestEnv(t)

	status, envl := call(t, env.handlers.GetGuardrails, "GET", "/api/guardrails", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get status = %d err = %q", status, errMessage(t, envl))
	}
	cfg := dataField[map[string]any](t, envl)
	if cfg["guardrails_enabled"] != false {
		t.Fatalf("default guardrails_enabled = %v, want false", cfg["guardrails_enabled"])
	}
	if _, ok := cfg["guardrails_blocklist"]; !ok {
		t.Fatalf("missing guardrails_blocklist: %v", cfg)
	}
	if _, ok := cfg["pii_redaction_enabled"]; !ok {
		t.Fatalf("missing pii_redaction_enabled: %v", cfg)
	}
	if _, ok := cfg["pii_redaction_types"]; !ok {
		t.Fatalf("missing pii_redaction_types: %v", cfg)
	}
}

func TestGuardrailsUpdateAndReflect(t *testing.T) {
	env := newTestEnv(t)
	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}

	body := `{"guardrails_enabled":true,"guardrails_blocklist":["password","secret"],"pii_redaction_enabled":false,"pii_redaction_types":["email"]}`
	status, envl := call(t, env.handlers.UpdateGuardrails, "PUT", "/api/guardrails", body,
		map[string]any{userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("update status = %d err = %q", status, errMessage(t, envl))
	}
	updated := dataField[map[string]any](t, envl)
	if updated["guardrails_enabled"] != true {
		t.Fatalf("updated guardrails_enabled = %v, want true", updated["guardrails_enabled"])
	}
	bl, _ := updated["guardrails_blocklist"].([]any)
	if len(bl) != 2 {
		t.Fatalf("blocklist = %v", updated["guardrails_blocklist"])
	}

	// GET reflects the update.
	status, envl = call(t, env.handlers.GetGuardrails, "GET", "/api/guardrails", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get-after-update status = %d", status)
	}
	cfg := dataField[map[string]any](t, envl)
	if cfg["guardrails_enabled"] != true {
		t.Fatalf("get-after-update enabled = %v, want true", cfg["guardrails_enabled"])
	}

	// Audit entry on update.
	status, envl = call(t, env.handlers.GetAudit, "GET", "/api/audit", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("audit status = %d", status)
	}
	page := dataField[struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}](t, envl)
	if page.Total < 1 {
		t.Fatalf("expected an audit entry after guardrails update, total=%d", page.Total)
	}
	if page.Items[0]["actor"] != "admin" {
		t.Fatalf("audit actor = %v", page.Items[0]["actor"])
	}
}

func TestGuardrailsTestBlockedForSeedConfig(t *testing.T) {
	env := newTestEnv(t)
	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}

	// Configure the seed-equivalent state: enabled + blocklist password/secret.
	cfgBody := `{"guardrails_enabled":true,"guardrails_blocklist":["password","secret","badword1"],"pii_redaction_enabled":false,"pii_redaction_types":["email","phone","ssn"]}`
	status, _ := call(t, env.handlers.UpdateGuardrails, "PUT", "/api/guardrails", cfgBody,
		map[string]any{userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("config status = %d", status)
	}

	// The spec-binding case: "my secret password" → blocked:true.
	status, envl := call(t, env.handlers.TestGuardrails, "POST", "/api/guardrails/test",
		`{"prompt":"my secret password"}`, nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("test status = %d err = %q", status, errMessage(t, envl))
	}
	out := dataField[map[string]any](t, envl)
	if out["blocked"] != true {
		t.Fatalf("blocked = %v, want true", out["blocked"])
	}
	if _, ok := out["redacted_prompt"]; !ok {
		t.Fatalf("missing redacted_prompt: %v", out)
	}
	matches, _ := out["matches"].([]any)
	if len(matches) != 2 {
		t.Fatalf("matches = %v, want 2 (password, secret)", out["matches"])
	}
}

func TestGuardrailsTestNotBlockedWhenDisabled(t *testing.T) {
	env := newTestEnv(t)
	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}

	cfgBody := `{"guardrails_enabled":false,"guardrails_blocklist":["password","secret"],"pii_redaction_enabled":false,"pii_redaction_types":[]}`
	status, _ := call(t, env.handlers.UpdateGuardrails, "PUT", "/api/guardrails", cfgBody,
		map[string]any{userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("config status = %d", status)
	}

	status, envl := call(t, env.handlers.TestGuardrails, "POST", "/api/guardrails/test",
		`{"prompt":"my secret password"}`, nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("test status = %d", status)
	}
	out := dataField[map[string]any](t, envl)
	if out["blocked"] != false {
		t.Fatalf("blocked = %v, want false (disabled)", out["blocked"])
	}
}
