package governance

import (
	"regexp"
	"strings"

	"github.com/bloodf/g0router/internal/store"
)

// GuardrailEngine evaluates prompts against the stored guardrails config and
// reads/writes that config through the store.
type GuardrailEngine struct {
	store *store.Store
}

// NewGuardrailEngine creates a guardrail engine backed by the store. The store
// may be nil when only the pure Evaluate method is used (e.g. in unit tests).
func NewGuardrailEngine(st *store.Store) *GuardrailEngine {
	return &GuardrailEngine{store: st}
}

// Config returns the current guardrails config.
func (e *GuardrailEngine) Config() (*store.Guardrails, error) {
	return e.store.GetGuardrails()
}

// Save persists the guardrails config.
func (e *GuardrailEngine) Save(g *store.Guardrails) error {
	return e.store.SetGuardrails(g)
}

// piiPatterns are deterministic, dependency-free PII matchers keyed by type.
var piiPatterns = map[string]*regexp.Regexp{
	"email": regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
	"phone": regexp.MustCompile(`\b\d{3}[-.\s]?\d{3}[-.\s]?\d{4}\b`),
	"ssn":   regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
}

// Evaluate is a pure, dependency-free guardrail evaluation. It mirrors the
// e2e mock: blocked is true when guardrails are enabled and a blocklist word is
// a case-insensitive substring of the prompt; matches lists the matching words
// in blocklist order; redacted is the prompt with PII redacted when PII
// redaction is enabled, otherwise the prompt verbatim.
func (e *GuardrailEngine) Evaluate(cfg *store.Guardrails, prompt string) (blocked bool, redacted string, matches []string) {
	matches = []string{}
	if cfg != nil && cfg.Enabled {
		lowerPrompt := strings.ToLower(prompt)
		for _, w := range cfg.Blocklist {
			if w == "" {
				continue
			}
			if strings.Contains(lowerPrompt, strings.ToLower(w)) {
				matches = append(matches, w)
			}
		}
	}
	blocked = len(matches) > 0

	redacted = prompt
	if cfg != nil && cfg.PIIRedactionEnabled {
		for _, typ := range cfg.PIIRedactionTypes {
			if re, ok := piiPatterns[typ]; ok {
				redacted = re.ReplaceAllString(redacted, "[REDACTED]")
			}
		}
	}
	return blocked, redacted, matches
}
