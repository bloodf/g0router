package governance

import (
	"reflect"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/store"
)

func TestGuardrailEngineEvaluate(t *testing.T) {
	eng := NewGuardrailEngine(nil)

	cases := []struct {
		name        string
		cfg         *store.Guardrails
		prompt      string
		wantBlocked bool
		wantMatches []string
	}{
		{
			name: "enabled blocklist substring match (the spec case)",
			cfg: &store.Guardrails{
				Enabled:   true,
				Blocklist: []string{"password", "secret", "badword1"},
			},
			prompt:      "my secret password",
			wantBlocked: true,
			wantMatches: []string{"password", "secret"},
		},
		{
			name: "disabled never blocks",
			cfg: &store.Guardrails{
				Enabled:   false,
				Blocklist: []string{"password", "secret"},
			},
			prompt:      "my secret password",
			wantBlocked: false,
			wantMatches: []string{},
		},
		{
			name: "no match",
			cfg: &store.Guardrails{
				Enabled:   true,
				Blocklist: []string{"password", "secret"},
			},
			prompt:      "hello world",
			wantBlocked: false,
			wantMatches: []string{},
		},
		{
			name: "case-insensitive substring",
			cfg: &store.Guardrails{
				Enabled:   true,
				Blocklist: []string{"secret"},
			},
			prompt:      "This is SECRET",
			wantBlocked: true,
			wantMatches: []string{"secret"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			blocked, _, matches := eng.Evaluate(tc.cfg, tc.prompt)
			if blocked != tc.wantBlocked {
				t.Fatalf("blocked = %v, want %v", blocked, tc.wantBlocked)
			}
			if !reflect.DeepEqual(matches, tc.wantMatches) {
				t.Fatalf("matches = %v, want %v", matches, tc.wantMatches)
			}
		})
	}
}

func TestGuardrailEngineRedactedPromptEchoWhenPIIOff(t *testing.T) {
	eng := NewGuardrailEngine(nil)
	cfg := &store.Guardrails{Enabled: true, Blocklist: []string{"x"}, PIIRedactionEnabled: false}
	_, redacted, _ := eng.Evaluate(cfg, "contact me at a@b.com")
	if redacted != "contact me at a@b.com" {
		t.Fatalf("redacted = %q, want verbatim prompt", redacted)
	}
}

func TestGuardrailEngineRedactsPIIWhenEnabled(t *testing.T) {
	eng := NewGuardrailEngine(nil)
	cfg := &store.Guardrails{
		Enabled:             false,
		PIIRedactionEnabled: true,
		PIIRedactionTypes:   []string{"email", "phone", "ssn"},
	}
	_, redacted, _ := eng.Evaluate(cfg, "email a@b.com phone 555-123-4567 ssn 123-45-6789")
	if strings.Contains(redacted, "a@b.com") {
		t.Fatalf("email not redacted: %q", redacted)
	}
	if strings.Contains(redacted, "555-123-4567") {
		t.Fatalf("phone not redacted: %q", redacted)
	}
	if strings.Contains(redacted, "123-45-6789") {
		t.Fatalf("ssn not redacted: %q", redacted)
	}
	if !strings.Contains(redacted, "[REDACTED]") {
		t.Fatalf("expected [REDACTED] markers: %q", redacted)
	}
}
