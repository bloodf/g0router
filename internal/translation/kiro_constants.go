package translation

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	kiroAgenticSuffix       = "-agentic"
	kiroThinkingSuffix      = "-thinking"
	kiroThinkingBudgetDefault = 16000
)

// kiroAgenticSystemPrompt is the chunked-write system prompt injected when the
// model carries the "-agentic" suffix. Byte-exact port of
// open-sse/config/kiroConstants.js:23-71 (.trim()-ed template literal).
const kiroAgenticSystemPrompt = `# CRITICAL: CHUNKED WRITE PROTOCOL (MANDATORY)

You MUST follow these rules for ALL file operations. Violation causes server timeouts and task failure.

## ABSOLUTE LIMITS
- **MAXIMUM 350 LINES** per single write/edit operation - NO EXCEPTIONS
- **RECOMMENDED 300 LINES** or less for optimal performance
- **NEVER** write entire files in one operation if >300 lines

## MANDATORY CHUNKED WRITE STRATEGY

### For NEW FILES (>300 lines total):
1. FIRST: Write initial chunk (first 250-300 lines) using write_to_file/fsWrite
2. THEN: Append remaining content in 250-300 line chunks using file append operations
3. REPEAT: Continue appending until complete

### For EDITING EXISTING FILES:
1. Use surgical edits (apply_diff/targeted edits) - change ONLY what's needed
2. NEVER rewrite entire files - use incremental modifications
3. Split large refactors into multiple small, focused edits

### For LARGE CODE GENERATION:
1. Generate in logical sections (imports, types, functions separately)
2. Write each section as a separate operation
3. Use append operations for subsequent sections

## EXAMPLES OF CORRECT BEHAVIOR

CORRECT: Writing a 600-line file
- Operation 1: Write lines 1-300 (initial file creation)
- Operation 2: Append lines 301-600

CORRECT: Editing multiple functions
- Operation 1: Edit function A
- Operation 2: Edit function B
- Operation 3: Edit function C

WRONG: Writing 500 lines in single operation -> TIMEOUT
WRONG: Rewriting entire file to change 5 lines -> TIMEOUT
WRONG: Generating massive code blocks without chunking -> TIMEOUT

## WHY THIS MATTERS
- Server has 2-3 minute timeout for operations
- Large writes exceed timeout and FAIL completely
- Chunked writes are FASTER and more RELIABLE
- Failed writes waste time and require retry

REMEMBER: When in doubt, write LESS per operation. Multiple small operations > one large operation.`

func isAgenticModel(model string) bool {
	return strings.HasSuffix(model, kiroAgenticSuffix)
}

func stripAgenticSuffix(model string) string {
	if !isAgenticModel(model) {
		return model
	}
	return model[:len(model)-len(kiroAgenticSuffix)]
}

func isThinkingModel(model string) bool {
	return strings.HasSuffix(model, kiroThinkingSuffix)
}

func stripThinkingSuffix(model string) string {
	if !isThinkingModel(model) {
		return model
	}
	return model[:len(model)-len(kiroThinkingSuffix)]
}

// ResolveKiroModel resolves a 9router model id to the real upstream Kiro model
// id, plus flags describing which behaviours the suffixes implied.
//
// Agentic suffix is stripped FIRST, then thinking.
//
//	ResolveKiroModel("claude-sonnet-4.5-thinking-agentic")
//	  => upstream="claude-sonnet-4.5", agentic=true, thinking=true
//	ResolveKiroModel("claude-sonnet-4.5-thinking")
//	  => upstream="claude-sonnet-4.5", agentic=false, thinking=true
//	ResolveKiroModel("claude-sonnet-4.5-agentic")
//	  => upstream="claude-sonnet-4.5", agentic=true, thinking=false
//	ResolveKiroModel("claude-sonnet-4.5")
//	  => upstream="claude-sonnet-4.5", agentic=false, thinking=false
func ResolveKiroModel(model string) (upstream string, agentic bool, thinking bool) {
	upstream = model
	if isAgenticModel(upstream) {
		agentic = true
		upstream = stripAgenticSuffix(upstream)
	}
	if isThinkingModel(upstream) {
		thinking = true
		upstream = stripThinkingSuffix(upstream)
	}
	return upstream, agentic, thinking
}

// isThinkingEnabled detects whether an inbound request is asking for reasoning /
// thinking output. Port of open-sse/config/kiroConstants.js:91-130.
func isThinkingEnabled(body map[string]any, headers map[string]string, model string) bool {
	if headers != nil {
		for k, v := range headers {
			if strings.EqualFold(k, "anthropic-beta") {
				if strings.Contains(strings.ToLower(v), "interleaved-thinking") {
					return true
				}
			}
		}
	}

	if body != nil {
		if thinking, ok := body["thinking"].(map[string]any); ok && thinking != nil {
			if t, _ := thinking["type"].(string); t == "enabled" {
				budget := math.NaN()
				switch bv := thinking["budget_tokens"].(type) {
				case float64:
					budget = bv
				case int:
					budget = float64(bv)
				case string:
					if f, err := strconv.ParseFloat(bv, 64); err == nil {
						budget = f
					}
				}
				if math.IsNaN(budget) || math.IsInf(budget, 1) || math.IsInf(budget, -1) || budget > 0 {
					return true
				}
			}
		}

		var effort string
		if e, ok := body["reasoning_effort"].(string); ok && e != "" {
			effort = e
		} else if reasoning, ok := body["reasoning"].(map[string]any); ok && reasoning != nil {
			if e, ok := reasoning["effort"].(string); ok && e != "" {
				effort = e
			}
		}
		if effort != "" {
			v := strings.ToLower(effort)
			if v != "none" && (v == "low" || v == "medium" || v == "high" || v == "auto") {
				return true
			}
		}

		if containsThinkingModeTag(body) {
			return true
		}
	}

	if model != "" {
		m := strings.ToLower(model)
		if strings.Contains(m, "thinking") || strings.Contains(m, "-reason") {
			return true
		}
	}

	return false
}

func containsThinkingModeTag(body map[string]any) bool {
	msgs, _ := body["messages"].([]any)
	for _, msg := range msgs {
		m, ok := msg.(map[string]any)
		if !ok {
			continue
		}
		role, _ := m["role"].(string)
		if role != "system" && role != "user" {
			continue
		}
		content := m["content"]
		if text, ok := content.(string); ok {
			if containsTagInText(text) {
				return true
			}
		} else if parts, ok := content.([]any); ok {
			for _, part := range parts {
				pm, ok := part.(map[string]any)
				if !ok {
					continue
				}
				if text, ok := pm["text"].(string); ok && containsTagInText(text) {
					return true
				}
			}
		}
	}
	if sys, ok := body["system"].(string); ok && containsTagInText(sys) {
		return true
	}
	return false
}

func containsTagInText(text string) bool {
	if text == "" {
		return false
	}
	if !strings.Contains(text, "<thinking_mode>") {
		return false
	}
	return strings.Contains(text, "<thinking_mode>enabled</thinking_mode>") ||
		strings.Contains(text, "<thinking_mode>interleaved</thinking_mode>")
}

// buildThinkingSystemPrefix builds the magic system-prompt prefix that turns
// Kiro reasoning on. Port of open-sse/config/kiroConstants.js:219-222.
func buildThinkingSystemPrefix(budget int) string {
	safe := kiroThinkingBudgetDefault
	if budget != 0 {
		safe = budget
	}
	if safe < 1 {
		safe = 1
	}
	if safe > 32000 {
		safe = 32000
	}
	return fmt.Sprintf("<thinking_mode>enabled</thinking_mode>\n<max_thinking_length>%d</max_thinking_length>", safe)
}
