package inference

import (
	"strconv"
	"testing"
	"time"
)

func TestErrorRulesOrderTextFirst(t *testing.T) {
	// 200 with a rate-limit text body must classify by TEXT rule, not status.
	c200 := Classify(200, []byte(`{"error":{"message":"You have hit a rate limit"}}`))
	if c200.Class != ClassRateLimit {
		t.Errorf("200+rate limit text: class=%v, want ClassRateLimit", c200.Class)
	}
	if !c200.Retryable {
		t.Errorf("200+rate limit text: Retryable=false, want true")
	}

	// 429 with the same text must also be ClassRateLimit — proves same verdict
	// regardless of whether text or status matched first.
	c429 := Classify(429, []byte(`{"error":{"message":"You have hit a rate limit"}}`))
	if c429.Class != ClassRateLimit {
		t.Errorf("429+rate limit text: class=%v, want ClassRateLimit", c429.Class)
	}
	if !c429.Retryable {
		t.Errorf("429+rate limit text: Retryable=false, want true")
	}

	// A status that would otherwise be auth_error is overridden by text match.
	c401 := Classify(401, []byte(`{"error":{"message":"rate limit exceeded"}}`))
	if c401.Class != ClassRateLimit {
		t.Errorf("401+rate limit text: class=%v, want ClassRateLimit", c401.Class)
	}
}

func TestResetsAtCap30Min(t *testing.T) {
	now := time.Now()
	// resets_at in seconds, 2 hours in the future. Must be capped at 30 min.
	futureSec := now.Unix() + 2*60*60
	body := []byte(`{"error":{"type":"usage_limit_reached","resets_at":` + itoa64(futureSec) + `}}`)

	c := Classify(429, body)
	if c.Class != ClassRateLimit {
		t.Fatalf("class=%v, want ClassRateLimit", c.Class)
	}

	want := now.Add(30 * time.Minute)
	diff := c.ResetsAt.Sub(want)
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Second {
		t.Errorf("ResetsAt cap diff=%v, want within 1s of %v, got %v", diff, want, c.ResetsAt)
	}

	// It must NOT be the raw 2-hour value.
	if c.ResetsAt.After(now.Add(31 * time.Minute)) {
		t.Errorf("ResetsAt not capped: %v", c.ResetsAt)
	}
}

func TestQuotaWindowSecMs(t *testing.T) {
	now := time.Now()
	base := now.Unix() + 120 // 2 minutes in the future

	// Value in seconds (< 1e12) must be normalized to milliseconds.
	secBody := []byte(`{"error":{"resets_at":` + itoa64(base) + `}}`)
	cSec := Classify(429, secBody)

	// Value already in milliseconds (> 1e12) must be used as-is.
	msBody := []byte(`{"error":{"resets_at":` + itoa64(base*1000) + `}}`)
	cMs := Classify(429, msBody)

	want := time.Unix(base, 0)
	for _, tc := range []struct {
		name string
		got  time.Time
	}{
		{"seconds-input", cSec.ResetsAt},
		{"milliseconds-input", cMs.ResetsAt},
	} {
		diff := tc.got.Sub(want)
		if diff < 0 {
			diff = -diff
		}
		if diff > time.Second {
			t.Errorf("%s: ResetsAt=%v, want within 1s of %v", tc.name, tc.got, want)
		}
	}
}

func TestErrorClassFixture(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		body      string
		wantClass ErrorClass
	}{
		// Text-based rules in exact implementation order (errorConfig.js:59-68 + g0router
		// PR-1626 "unsupported" rule inserted at position 4, before "rate limit").
		{"no_credentials", 200, `{"error":"no credentials"}`, ClassAuthError},
		{"request_not_allowed", 200, `{"error":"request not allowed"}`, ClassAuthError},
		{"improperly_formed_request", 200, `{"error":"improperly formed request"}`, ClassPermanent},
		{"unsupported_param", 200, `{"error":"unsupported parameter: max_tokens"}`, ClassUnsupportedParam},
		{"rate_limit", 200, `{"error":"rate limit"}`, ClassRateLimit},
		{"too_many_requests", 200, `{"error":"too many requests"}`, ClassRateLimit},
		{"quota_exceeded", 200, `{"error":"quota exceeded"}`, ClassRateLimit},
		{"capacity", 200, `{"error":"capacity"}`, ClassRateLimit},
		{"overloaded", 200, `{"error":"overloaded"}`, ClassRateLimit},

		// Status-based rules from errorConfig.js:70-76 (fallback when text doesn't match).
		{"status_401", 401, `{}`, ClassAuthError},
		{"status_402", 402, `{}`, ClassAuthError},
		{"status_403", 403, `{}`, ClassAuthError},
		{"status_404", 404, `{}`, ClassPermanent},
		{"status_429", 429, `{}`, ClassRateLimit},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Classify(tc.status, []byte(tc.body))
			if got.Class != tc.wantClass {
				t.Errorf("Classify(%d, %q).Class = %v, want %v", tc.status, tc.body, got.Class, tc.wantClass)
			}
		})
	}
}

// TestErrorClassRuleOrder pins the exact sequence of rules returned by classificationRules,
// so any reordering of the rule table is caught immediately.
func TestErrorClassRuleOrder(t *testing.T) {
	wantOrder := []struct {
		text   string
		status int
		class  ErrorClass
	}{
		{"no credentials", 0, ClassAuthError},
		{"request not allowed", 0, ClassAuthError},
		{"improperly formed request", 0, ClassPermanent},
		{"unsupported", 0, ClassUnsupportedParam},
		{"rate limit", 0, ClassRateLimit},
		{"too many requests", 0, ClassRateLimit},
		{"quota exceeded", 0, ClassRateLimit},
		{"capacity", 0, ClassRateLimit},
		{"overloaded", 0, ClassRateLimit},
		{"", 401, ClassAuthError},
		{"", 402, ClassAuthError},
		{"", 403, ClassAuthError},
		{"", 404, ClassPermanent},
		{"", 429, ClassRateLimit},
	}
	rules := classificationRules()
	if len(rules) != len(wantOrder) {
		t.Fatalf("classificationRules() has %d rules, want %d", len(rules), len(wantOrder))
	}
	for i, want := range wantOrder {
		got := rules[i]
		if got.text != want.text || got.status != want.status || got.class != want.class {
			t.Errorf("rule[%d]: got {text:%q status:%d class:%v}, want {text:%q status:%d class:%v}",
				i, got.text, got.status, got.class, want.text, want.status, want.class)
		}
	}
}

// TestErrorClassCaseInsensitive ensures text matching is case-insensitive.
func TestErrorClassCaseInsensitive(t *testing.T) {
	c := Classify(200, []byte(`{"error":{"message":"RATE LIMIT HIT"}}`))
	if c.Class != ClassRateLimit {
		t.Errorf("uppercase text: class=%v, want ClassRateLimit", c.Class)
	}
}

// TestErrorClassResetsInSeconds supports the codex-style resets_in_seconds field.
func TestErrorClassResetsInSeconds(t *testing.T) {
	now := time.Now()
	body := []byte(`{"error":{"type":"usage_limit_reached","resets_in_seconds":90}}`)
	c := Classify(429, body)

	want := now.Add(90 * time.Second)
	diff := c.ResetsAt.Sub(want)
	if diff < 0 {
		diff = -diff
	}
	if diff > 2*time.Second {
		t.Errorf("ResetsAt=%v, want within 2s of %v", c.ResetsAt, want)
	}
}

func itoa64(v int64) string {
	return strconv.FormatInt(v, 10)
}
