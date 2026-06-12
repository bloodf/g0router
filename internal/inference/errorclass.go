package inference

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ErrorClass is the ordered verdict produced by Classify.
type ErrorClass int

const (
	ClassUnknown ErrorClass = iota
	ClassRateLimit
	ClassAuthError
	ClassTransient
	ClassPermanent
	ClassUnsupportedParam
)

// Classification holds the verdict for an upstream response.
type Classification struct {
	Class     ErrorClass
	ResetsAt  time.Time
	Retryable bool
}

// maxRateLimitCooldown is the hard cap on provider-reported rate-limit cooldown.
// Mirrors MAX_RATE_LIMIT_COOLDOWN_MS in open-sse/config/errorConfig.js:42.
const maxRateLimitCooldown = 30 * time.Minute

// unixTimestampMsThreshold is the boundary used to decide whether a numeric
// timestamp is in seconds or milliseconds. Matches parseResetTime in
// open-sse/services/usage.js:114.
const unixTimestampMsThreshold int64 = 1e12

// errorRule is a single classification rule. Text rules are evaluated before
// status rules, in the order declared here, matching open-sse/config/errorConfig.js.
type errorRule struct {
	text   string
	status int
	class  ErrorClass
}

// classificationRules returns the ordered rule table ported from ERROR_RULES in
// open-sse/config/errorConfig.js. Returning a fresh slice prevents callers from
// mutating the package-level rule table.
func classificationRules() []errorRule {
	return []errorRule{
		// Text-based rules from errorConfig.js (checked first, order = priority).
		{text: "no credentials", class: ClassAuthError},
		{text: "request not allowed", class: ClassAuthError},
		{text: "improperly formed request", class: ClassPermanent},
		{text: "unsupported", class: ClassUnsupportedParam},
		{text: "rate limit", class: ClassRateLimit},
		{text: "too many requests", class: ClassRateLimit},
		{text: "quota exceeded", class: ClassRateLimit},
		{text: "capacity", class: ClassRateLimit},
		{text: "overloaded", class: ClassRateLimit},

		// Status-based rules from errorConfig.js (fallback when text doesn't match).
		{status: 401, class: ClassAuthError},
		{status: 402, class: ClassAuthError},
		{status: 403, class: ClassAuthError},
		{status: 404, class: ClassPermanent},
		{status: 429, class: ClassRateLimit},
	}
}

// Classify inspects the HTTP status code and response body and returns a
// Classification. Text rules are checked before status rules, per PAR-ROUTE-044.
func Classify(statusCode int, body []byte) Classification {
	message := extractMessage(body)
	lower := strings.ToLower(message)

	c := Classification{Class: ClassUnknown, Retryable: true}

	for _, rule := range classificationRules() {
		if rule.text != "" {
			if strings.Contains(lower, rule.text) {
				c.Class = rule.class
				break
			}
			continue
		}
		if rule.status != 0 && statusCode == rule.status {
			c.Class = rule.class
			break
		}
	}

	// Server errors without an explicit rule are treated as transient/retriable.
	if c.Class == ClassUnknown && statusCode >= 500 {
		c.Class = ClassTransient
	}
	// 400 and 406 map to invalid_request_error in ERROR_TYPES and are permanent.
	if c.Class == ClassUnknown && (statusCode == 400 || statusCode == 406) {
		c.Class = ClassPermanent
	}

	switch c.Class {
	case ClassRateLimit:
		c.ResetsAt = extractResetsAt(body)
		c.Retryable = true
	case ClassTransient:
		c.Retryable = true
	case ClassAuthError, ClassPermanent, ClassUnsupportedParam:
		c.Retryable = false
	}

	return c
}

// extractMessage pulls the human-readable message out of an upstream error body.
func extractMessage(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return string(body)
	}

	// OpenAI-shaped error: { error: { message: "..." } }
	if errObj, ok := payload["error"].(map[string]any); ok {
		if msg, ok := errObj["message"].(string); ok {
			return msg
		}
		if msg, ok := errObj["message"].(fmt.Stringer); ok {
			return msg.String()
		}
	}

	// Fallback fields used by some providers.
	for _, key := range []string{"message", "error", "detail"} {
		switch v := payload[key].(type) {
		case string:
			return v
		case map[string]any:
			if msg, ok := v["message"].(string); ok {
				return msg
			}
		}
	}

	return string(body)
}

// extractResetsAt parses provider-specific reset fields from the error body and
// caps the resulting cooldown at maxRateLimitCooldown (PAR-ROUTE-045).
func extractResetsAt(body []byte) time.Time {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return time.Time{}
	}

	errObj, _ := payload["error"].(map[string]any)
	if errObj == nil {
		errObj = payload
	}

	now := time.Now()
	var resetsAt time.Time

	if v, ok := errObj["resets_at"]; ok {
		resetsAt = parseTimestamp(v, now)
	}
	if resetsAt.IsZero() {
		if v, ok := errObj["resets_in_seconds"]; ok {
			if sec, ok := toFloat64(v); ok && sec > 0 {
				resetsAt = now.Add(time.Duration(sec) * time.Second)
			}
		}
	}

	if resetsAt.IsZero() {
		return time.Time{}
	}

	// Cap at maxRateLimitCooldown from now.
	maxAllowed := now.Add(maxRateLimitCooldown)
	if resetsAt.After(maxAllowed) {
		resetsAt = maxAllowed
	}
	if resetsAt.Before(now) {
		return time.Time{}
	}
	return resetsAt
}

// parseTimestamp handles both Unix seconds and Unix milliseconds, matching
// open-sse/services/usage.js:114 (PAR-ROUTE-048).
func parseTimestamp(v any, now time.Time) time.Time {
	val, ok := toFloat64(v)
	if !ok || val <= 0 {
		return time.Time{}
	}

	ms := int64(val)
	if ms < unixTimestampMsThreshold {
		ms *= 1000
	}
	t := time.Unix(0, ms*int64(time.Millisecond))
	if t.Before(now) {
		return time.Time{}
	}
	return t
}

// toFloat64 coerces JSON numbers (float64) and numeric strings to float64.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case string:
		var f float64
		if _, err := fmt.Sscanf(n, "%f", &f); err == nil {
			return f, true
		}
	}
	return 0, false
}
