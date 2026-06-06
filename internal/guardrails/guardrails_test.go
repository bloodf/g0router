package guardrails

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func TestCheckBlocklist(t *testing.T) {
	cfg := Config{Enabled: true, Blocklist: []string{"badword", "FORBIDDEN"}}

	tests := []struct {
		name    string
		prompt  string
		want    bool
		matches []string
	}{
		{"no match", "hello world", false, nil},
		{"case insensitive", "This has BADWORD in it", true, []string{"badword"}},
		{"multiple matches", "badword and forbidden text", true, []string{"badword", "forbidden"}},
		{"partial word", "badwords are here", true, []string{"badword"}},
		{"disabled", "badword", false, nil},
		{"empty blocklist", "badword", false, nil},
		{"empty prompt", "", false, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "disabled" {
				c := Config{Enabled: false, Blocklist: cfg.Blocklist}
				blocked, matches := c.CheckBlocklist(tc.prompt)
				if blocked {
					t.Errorf("blocked = %v, want false", blocked)
				}
				if len(matches) != 0 {
					t.Errorf("matches = %v, want empty", matches)
				}
				return
			}
			if tc.name == "empty blocklist" {
				c := Config{Enabled: true, Blocklist: []string{}}
				blocked, matches := c.CheckBlocklist(tc.prompt)
				if blocked {
					t.Errorf("blocked = %v, want false", blocked)
				}
				if len(matches) != 0 {
					t.Errorf("matches = %v, want empty", matches)
				}
				return
			}
			blocked, matches := cfg.CheckBlocklist(tc.prompt)
			if blocked != tc.want {
				t.Errorf("blocked = %v, want %v", blocked, tc.want)
			}
			if tc.matches != nil {
				if len(matches) != len(tc.matches) {
					t.Errorf("matches = %v, want %v", matches, tc.matches)
				}
				for i := range tc.matches {
					if matches[i] != tc.matches[i] {
						t.Errorf("matches[%d] = %q, want %q", i, matches[i], tc.matches[i])
					}
				}
			}
		})
	}
}

func TestCheckRequest(t *testing.T) {
	cfg := Config{Enabled: true, Blocklist: []string{"badword"}}

	req := &providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "user", Content: "hello badword world"},
		},
	}

	blocked, matches, err := CheckRequest(cfg, req)
	if !blocked {
		t.Errorf("blocked = %v, want true", blocked)
	}
	if len(matches) != 1 || matches[0] != "badword" {
		t.Errorf("matches = %v, want [badword]", matches)
	}
	if err == nil {
		t.Error("expected error for blocklist match")
	}

	// Not blocked
	req2 := &providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "user", Content: "hello world"},
		},
	}
	blocked, matches, err = CheckRequest(cfg, req2)
	if blocked {
		t.Errorf("blocked = %v, want false", blocked)
	}
	if len(matches) != 0 {
		t.Errorf("matches = %v, want empty", matches)
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Disabled
	cfg2 := Config{Enabled: false}
	blocked, matches, err = CheckRequest(cfg2, req)
	if blocked {
		t.Errorf("blocked = %v, want false", blocked)
	}
	if len(matches) != 0 {
		t.Errorf("matches = %v, want empty", matches)
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckRequestWithContentBlocks(t *testing.T) {
	cfg := Config{Enabled: true, Blocklist: []string{"badword"}}

	req := &providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "user", Content: []map[string]any{
				{"type": "text", "text": "hello badword world"},
				{"type": "image_url", "image_url": map[string]any{"url": "http://example.com"}},
			}},
		},
	}

	blocked, matches, err := CheckRequest(cfg, req)
	if !blocked {
		t.Errorf("blocked = %v, want true", blocked)
	}
	if len(matches) != 1 || matches[0] != "badword" {
		t.Errorf("matches = %v, want [badword]", matches)
	}
	if err == nil {
		t.Error("expected error for blocklist match")
	}
}

func TestCheckRequestNil(t *testing.T) {
	blocked, matches, err := CheckRequest(Config{Enabled: true}, nil)
	if blocked {
		t.Errorf("blocked = %v, want false", blocked)
	}
	if len(matches) != 0 {
		t.Errorf("matches = %v, want empty", matches)
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRedactPII(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Types: []string{"email", "phone", "ssn", "credit_card", "ip_address"}}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"email", "Contact me at user@example.com please", "Contact me at [REDACTED:email] please"},
		{"phone us", "Call 555-123-4567 now", "Call [REDACTED:phone] now"},
		{"ssn", "My ssn is 123-45-6789", "My ssn is [REDACTED:ssn]"},
		{"credit card", "Card: 4111-1111-1111-1111", "Card: [REDACTED:credit_card]"},
		{"ip address", "Server at 192.168.1.1", "Server at [REDACTED:ip_address]"},
		{"multiple", "Email: a@b.com and phone: 555-123-4567", "Email: [REDACTED:email] and phone: [REDACTED:phone]"},
		{"no pii", "hello world", "hello world"},
		{"disabled", "a@b.com", "a@b.com"},
		{"empty", "", ""},
		{"unknown type ignored", "hello", "hello"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "disabled" {
				c := PIIConfig{Enabled: false, Types: cfg.Types}
				result := c.Redact(tc.input)
				if result != tc.expected {
					t.Errorf("Redact() = %q, want %q", result, tc.expected)
				}
				return
			}
			result := cfg.Redact(tc.input)
			if result != tc.expected {
				t.Errorf("Redact() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestRedactPIISelectiveTypes(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Types: []string{"email"}}

	result := cfg.Redact("Email: a@b.com, phone: 555-123-4567")
	if result != "Email: [REDACTED:email], phone: 555-123-4567" {
		t.Errorf("Redact() = %q, want selective redaction", result)
	}
}

func TestRedactRequest(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Types: []string{"email"}}

	req := &providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "user", Content: "Email: user@example.com"},
			{Role: "assistant", Content: "Got it"},
		},
	}

	redacted := RedactRequest(cfg, req)
	if redacted.Messages[0].Content != "Email: [REDACTED:email]" {
		t.Errorf("msg[0] content = %q, want redacted email", redacted.Messages[0].Content)
	}
	if redacted.Messages[1].Content != "Got it" {
		t.Errorf("msg[1] content = %q, want unchanged", redacted.Messages[1].Content)
	}

	// Ensure original is not mutated
	if req.Messages[0].Content != "Email: user@example.com" {
		t.Error("original request was mutated")
	}
}

func TestRedactRequestWithBlocks(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Types: []string{"email"}}

	req := &providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "user", Content: []map[string]any{
				{"type": "text", "text": "Email: user@example.com"},
				{"type": "image_url", "image_url": map[string]any{"url": "http://example.com"}},
			}},
		},
	}

	redacted := RedactRequest(cfg, req)

	blocks, ok := redacted.Messages[0].Content.([]map[string]any)
	if !ok {
		t.Fatalf("content type = %T, want []map[string]any", redacted.Messages[0].Content)
	}
	if blocks[0]["text"] != "Email: [REDACTED:email]" {
		t.Errorf("block text = %q, want redacted", blocks[0]["text"])
	}
	if blocks[1]["type"] != "image_url" {
		t.Error("non-text block was mutated")
	}

	// Original unchanged
	origBlocks := req.Messages[0].Content.([]map[string]any)
	if origBlocks[0]["text"] != "Email: user@example.com" {
		t.Error("original was mutated")
	}
}

func TestRedactRequestDisabled(t *testing.T) {
	cfg := PIIConfig{Enabled: false, Types: []string{"email"}}

	req := &providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "user", Content: "Email: user@example.com"},
		},
	}

	redacted := RedactRequest(cfg, req)
	if redacted.Messages[0].Content != "Email: user@example.com" {
		t.Errorf("content = %q, want unchanged", redacted.Messages[0].Content)
	}
}

func TestRedactRequestNil(t *testing.T) {
	result := RedactRequest(PIIConfig{Enabled: true}, nil)
	if result != nil {
		t.Error("expected nil for nil request")
	}
}

func TestPIIRegexCoverage(t *testing.T) {
	tests := []struct {
		typ     string
		input   string
		matched bool
	}{
		{"email", "test@example.com", true},
		{"email", "not-an-email", false},
		{"phone", "123-456-7890", true},
		{"phone", "abc", false},
		{"ssn", "123-45-6789", true},
		{"ssn", "123-45-678", false},
		{"credit_card", "4111-1111-1111-1111", true},
		{"credit_card", "1234", false},
		{"ip_address", "192.168.1.1", true},
		{"ip_address", "999.999.999.999", false},
	}

	for _, tc := range tests {
		t.Run(tc.typ+"_"+tc.input, func(t *testing.T) {
			re := piiRegexes[tc.typ]
			matched := re.MatchString(tc.input)
			if matched != tc.matched {
				t.Errorf("MatchString(%q) = %v, want %v", tc.input, matched, tc.matched)
			}
		})
	}
}

func TestErrBlocklistMatch(t *testing.T) {
	err := ErrBlocklistMatch
	if !errors.Is(err, ErrBlocklistMatch) {
		t.Error("ErrBlocklistMatch should be its own sentinel")
	}
}

func TestCheckBlocklistEmptyTerm(t *testing.T) {
	cfg := Config{Enabled: true, Blocklist: []string{"", "badword"}}
	blocked, matches := cfg.CheckBlocklist("badword here")
	if !blocked {
		t.Errorf("blocked = %v, want true", blocked)
	}
	if len(matches) != 1 || matches[0] != "badword" {
		t.Errorf("matches = %v, want [badword]", matches)
	}
}

func TestMessageTextWithAnySlice(t *testing.T) {
	content := []any{
		map[string]any{"type": "text", "text": "hello"},
		map[string]any{"type": "image_url", "image_url": "http://example.com"},
	}
	text := messageText(content)
	if text != "hello" {
		t.Errorf("messageText = %q, want %q", text, "hello")
	}
}

func TestMessageTextDefault(t *testing.T) {
	content := map[string]any{"foo": "bar"}
	text := messageText(content)
	if text != `{"foo":"bar"}` {
		t.Errorf("messageText = %q, want %q", text, `{"foo":"bar"}`)
	}
}

func TestRedactContentWithAnySlice(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Types: []string{"email"}}
	content := []any{
		map[string]any{"type": "text", "text": "Email: user@example.com"},
		map[string]any{"type": "image_url", "image_url": "http://example.com"},
	}
	result := redactContent(cfg, content)
	blocks, ok := result.([]any)
	if !ok {
		t.Fatalf("type = %T, want []any", result)
	}
	m0 := blocks[0].(map[string]any)
	if m0["text"] != "Email: [REDACTED:email]" {
		t.Errorf("text = %q, want redacted", m0["text"])
	}
	m1 := blocks[1].(map[string]any)
	if m1["type"] != "image_url" {
		t.Error("non-text block was mutated")
	}
}

func TestRedactContentDefault(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Types: []string{"email"}}
	content := 12345
	result := redactContent(cfg, content)
	if result != 12345 {
		t.Errorf("redactContent = %v, want 12345", result)
	}
}
