package inference

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers/catalog"
)

func TestRetryPerStatusAttempts(t *testing.T) {
	// Default retry config gives 429 zero attempts, so a provider that would
	// eventually succeed is not retried and the first 429 is returned.
	calls := 0
	call := func() (int, []byte, error) {
		calls++
		if calls < 3 {
			return 429, []byte(`{"error":{"message":"rate limit"}}`), nil
		}
		return 200, []byte(`{"ok":true}`), nil
	}

	provider := catalog.ProviderConfig{}
	status, body, err := WithRetry(context.Background(), provider, call, newDefaultRetryConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 429 {
		t.Errorf("status=%d, want 429", status)
	}
	if calls != 1 {
		t.Errorf("calls=%d, want 1 (no retry for 429 by default)", calls)
	}
	_ = body
}

func TestProviderRetryOverride(t *testing.T) {
	// Use kiro's actual catalog retry data instead of a hardcoded map.
	provider, ok := catalog.Lookup("kiro")
	if !ok {
		t.Fatal("catalog.Lookup(\"kiro\") returned ok=false")
	}
	if provider.RetryOverride()[429] != 2 {
		t.Fatalf("kiro retry override for 429=%v, want 2", provider.RetryOverride()[429])
	}

	calls := 0
	call := func() (int, []byte, error) {
		calls++
		if calls < 3 {
			return 429, []byte(`{"error":{"message":"rate limit"}}`), nil
		}
		return 200, []byte(`{"ok":true}`), nil
	}

	status, _, err := WithRetry(context.Background(), provider, call, newDefaultRetryConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 200 {
		t.Errorf("status=%d, want 200", status)
	}
	if calls != 3 {
		t.Errorf("calls=%d, want 3 (2 retries + 1 success)", calls)
	}
}

func TestProviderRetryOverrideFewerAttempts(t *testing.T) {
	// Override with fewer attempts than default; use a tiny delay to keep the test fast.
	calls := 0
	call := func() (int, []byte, error) {
		calls++
		return 502, []byte(`{"error":{"message":"bad gateway"}}`), nil
	}

	provider := catalog.ProviderConfig{Retry: map[int]int{502: 1}}
	cfg := map[int]RetryEntry{502: {Attempts: 3, DelayMs: 1}}
	status, _, err := WithRetry(context.Background(), provider, call, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 502 {
		t.Errorf("status=%d, want 502", status)
	}
	// 1 attempt allowed means 1 initial call + 1 retry = 2 calls.
	if calls != 2 {
		t.Errorf("calls=%d, want 2", calls)
	}
}

func TestConnectTimeout502NotRetriedAsClientAbort(t *testing.T) {
	// Connect timeout is a net.Error with Timeout() true and Temporary() false.
	calls := 0
	call := func() (int, []byte, error) {
		calls++
		return 0, nil, &fakeNetError{timeout: true, temporary: false}
	}

	provider := catalog.ProviderConfig{}
	status, _, err := WithRetry(context.Background(), provider, call, newDefaultRetryConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 502 {
		t.Errorf("status=%d, want 502 for connect timeout", status)
	}
	if calls != 1 {
		t.Errorf("calls=%d, want 1 (no retry for connect timeout)", calls)
	}

	// Client mid-stream abort (context.Canceled) is a different path and must
	// NOT be converted to a 502.
	clientCalls := 0
	clientCall := func() (int, []byte, error) {
		clientCalls++
		return 0, nil, context.Canceled
	}
	_, _, err = WithRetry(context.Background(), provider, clientCall, newDefaultRetryConfig())
	if !errors.Is(err, context.Canceled) {
		t.Errorf("client abort err=%v, want context.Canceled", err)
	}
	if clientCalls != 1 {
		t.Errorf("client abort calls=%d, want 1", clientCalls)
	}
}

func TestNoRetryOnPermanentClass(t *testing.T) {
	calls := 0
	call := func() (int, []byte, error) {
		calls++
		return 400, []byte(`{"error":{"message":"invalid request"}}`), nil
	}

	provider := catalog.ProviderConfig{}
	status, _, err := WithRetry(context.Background(), provider, call, newDefaultRetryConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 400 {
		t.Errorf("status=%d, want 400", status)
	}
	if calls != 1 {
		t.Errorf("calls=%d, want 1 (no retry for ClassPermanent)", calls)
	}
}

func TestRetryRespectsResetsAt(t *testing.T) {
	// Rate limit with a short resets_at should be respected (capped behavior is
	// tested in errorclass tests; here we just ensure it does not blow up).
	calls := 0
	call := func() (int, []byte, error) {
		calls++
		if calls == 1 {
			return 429, []byte(`{"error":{"message":"rate limit","resets_in_seconds":1}}`), nil
		}
		return 200, []byte(`{"ok":true}`), nil
	}

	provider := catalog.ProviderConfig{Retry: map[int]int{429: 1}}
	start := time.Now()
	status, _, err := WithRetry(context.Background(), provider, call, newDefaultRetryConfig())
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 200 {
		t.Errorf("status=%d, want 200", status)
	}
	if calls != 2 {
		t.Errorf("calls=%d, want 2", calls)
	}
	// Should have slept at least the 1-second resets_at window.
	if elapsed < 800*time.Millisecond {
		t.Errorf("elapsed=%v, expected at least ~1s sleep for resets_at", elapsed)
	}
}

// fakeNetError implements net.Error for test control.
type fakeNetError struct {
	timeout   bool
	temporary bool
}

func (f *fakeNetError) Error() string   { return "fake network error" }
func (f *fakeNetError) Timeout() bool   { return f.timeout }
func (f *fakeNetError) Temporary() bool { return f.temporary }

var _ net.Error = (*fakeNetError)(nil)

func TestTokenParamAutoLearn(t *testing.T) {
	settings := &fakeSettings{data: make(map[string]string)}
	providerID := "test-provider"
	modelID := "test-model"

	body := map[string]any{
		"model":      modelID,
		"messages":   []any{map[string]any{"role": "user", "content": "hi"}},
		"max_tokens": 100,
	}

	// First request: max_tokens is rejected, max_completion_tokens succeeds.
	firstCalls := 0
	firstCall := func(b map[string]any) (int, []byte, error) {
		firstCalls++
		if _, ok := b["max_tokens"]; ok {
			return 400, []byte(`{"error":{"message":"unsupported parameter: max_tokens"}}`), nil
		}
		if _, ok := b["max_completion_tokens"]; ok {
			return 200, []byte(`{"ok":true}`), nil
		}
		return 500, nil, fmt.Errorf("no token param in body")
	}

	status, _, err := AutoLearnTokenParam(context.Background(), providerID, modelID, body, settings, firstCall)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 200 {
		t.Errorf("status=%d, want 200", status)
	}
	if firstCalls != 2 {
		t.Errorf("firstCalls=%d, want 2", firstCalls)
	}

	key := learnedTokenParamKey(providerID, modelID)
	if got, want := settings.data[key], TokenParamMaxCompletionTokens; got != want {
		t.Errorf("learned pref=%q, want %q", got, want)
	}

	// Second request for the same provider+model should immediately use the
	// learned parameter without trying max_tokens first.
	secondBody := map[string]any{
		"model":      modelID,
		"messages":   []any{map[string]any{"role": "user", "content": "hi"}},
		"max_tokens": 100,
	}
	secondCalls := 0
	secondCall := func(b map[string]any) (int, []byte, error) {
		secondCalls++
		if _, ok := b["max_tokens"]; ok {
			t.Errorf("second request used max_tokens before learned preference")
			return 400, []byte(`{"error":{"message":"unsupported parameter: max_tokens"}}`), nil
		}
		if _, ok := b["max_completion_tokens"]; ok {
			return 200, []byte(`{"ok":true}`), nil
		}
		return 500, nil, fmt.Errorf("no token param in body")
	}

	status, _, err = AutoLearnTokenParam(context.Background(), providerID, modelID, secondBody, settings, secondCall)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 200 {
		t.Errorf("status=%d, want 200", status)
	}
	if secondCalls != 1 {
		t.Errorf("secondCalls=%d, want 1", secondCalls)
	}
}

func TestTokenParamAutoLearnNoMismatch(t *testing.T) {
	// A normal permanent error that does NOT mention the token param must not
	// trigger the auto-learn retry.
	settings := &fakeSettings{data: make(map[string]string)}
	body := map[string]any{"max_tokens": 100}

	calls := 0
	call := func(b map[string]any) (int, []byte, error) {
		calls++
		return 400, []byte(`{"error":{"message":"invalid request"}}`), nil
	}

	status, _, err := AutoLearnTokenParam(context.Background(), "p", "m", body, settings, call)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 400 {
		t.Errorf("status=%d, want 400", status)
	}
	if calls != 1 {
		t.Errorf("calls=%d, want 1", calls)
	}
	if len(settings.data) != 0 {
		t.Errorf("unexpected learned prefs: %v", settings.data)
	}
}

type fakeSettings struct {
	data      map[string]string
	setErr    error
	setCalled int
}

func (f *fakeSettings) GetSetting(key string) (string, error) {
	v, ok := f.data[key]
	if !ok {
		return "", fmt.Errorf("missing key %s", key)
	}
	return v, nil
}

func (f *fakeSettings) SetSetting(key, value string) error {
	f.setCalled++
	if f.setErr != nil {
		return f.setErr
	}
	f.data[key] = value
	return nil
}

func TestTokenParamAutoLearnPersistError(t *testing.T) {
	settings := &fakeSettings{data: make(map[string]string), setErr: errors.New("disk full")}
	body := map[string]any{"max_tokens": 100}

	call := func(b map[string]any) (int, []byte, error) {
		if _, ok := b["max_tokens"]; ok {
			return 400, []byte(`{"error":{"message":"unsupported parameter: max_tokens"}}`), nil
		}
		return 200, []byte(`{"ok":true}`), nil
	}

	status, _, err := AutoLearnTokenParam(context.Background(), "p", "m", body, settings, call)
	if err == nil {
		t.Fatal("expected persistence error, got nil")
	}
	if status != 200 {
		t.Errorf("status=%d, want 200 (upstream succeeded)", status)
	}
	if !errors.Is(err, settings.setErr) {
		t.Errorf("err=%v, want wrapped disk full", err)
	}
}
