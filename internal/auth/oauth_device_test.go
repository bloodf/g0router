package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// noWaitFlow returns a flow whose device-poll clock never actually sleeps, so the
// poll loop runs hermetically with no real time elapsed.
func noWaitFlow(cfg OAuthConfig, st *store.Store, client *http.Client) *OAuthFlow {
	f := NewOAuthFlow(cfg, st, client)
	f.afterFunc = func(time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}
	return f
}

func TestStartDeviceQwen(t *testing.T) {
	st := newTestStore(t)
	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotForm = r.PostForm
		json.NewEncoder(w).Encode(map[string]any{
			"device_code":      "dev-123",
			"user_code":        "WXYZ-1234",
			"verification_uri": "https://chat.qwen.ai/device",
			"interval":         5,
			"expires_in":       300,
		})
	}))
	defer srv.Close()

	cfg := QwenOAuth()
	cfg.DeviceCodeURL = srv.URL
	cfg.TokenURL = srv.URL
	flow := NewOAuthFlow(cfg, st, srv.Client())

	resp, err := flow.StartDevice()
	if err != nil {
		t.Fatalf("StartDevice: %v", err)
	}
	if resp.DeviceCode != "dev-123" {
		t.Errorf("DeviceCode = %q", resp.DeviceCode)
	}
	if resp.UserCode != "WXYZ-1234" {
		t.Errorf("UserCode = %q", resp.UserCode)
	}
	if resp.VerificationURI != "https://chat.qwen.ai/device" {
		t.Errorf("VerificationURI = %q", resp.VerificationURI)
	}
	if resp.Interval != 5 {
		t.Errorf("Interval = %d", resp.Interval)
	}
	// qwen device-code request is PKCE: code_challenge + S256 must be sent.
	if gotForm.Get("code_challenge") == "" {
		t.Error("device-code request missing code_challenge (PKCE)")
	}
	if gotForm.Get("code_challenge_method") != "S256" {
		t.Errorf("code_challenge_method = %q", gotForm.Get("code_challenge_method"))
	}
	if gotForm.Get("client_id") != cfg.ClientID {
		t.Errorf("client_id = %q", gotForm.Get("client_id"))
	}
}

func TestPollDevicePendingThenSuccess(t *testing.T) {
	st := newTestStore(t)
	var calls int32
	var lastForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		lastForm = r.PostForm
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"error": "authorization_pending"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "qwen-at",
			"refresh_token": "qwen-rt",
			"expires_in":    3600,
		})
	}))
	defer srv.Close()

	cfg := QwenOAuth()
	cfg.DeviceCodeURL = srv.URL
	cfg.TokenURL = srv.URL
	flow := noWaitFlow(cfg, st, srv.Client())

	tok, err := flow.PollDevice(context.Background(), "dev-xyz", "verifier-abc", 1)
	if err != nil {
		t.Fatalf("PollDevice: %v", err)
	}
	if tok.AccessToken != "qwen-at" {
		t.Errorf("AccessToken = %q", tok.AccessToken)
	}
	if tok.RefreshToken != "qwen-rt" {
		t.Errorf("RefreshToken = %q", tok.RefreshToken)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
	// qwen device-code poll sends grant_type=device_code + code_verifier.
	if lastForm.Get("grant_type") != "urn:ietf:params:oauth:grant-type:device_code" {
		t.Errorf("grant_type = %q", lastForm.Get("grant_type"))
	}
	if lastForm.Get("code_verifier") != "verifier-abc" {
		t.Errorf("code_verifier = %q", lastForm.Get("code_verifier"))
	}
	if lastForm.Get("device_code") != "dev-xyz" {
		t.Errorf("device_code = %q", lastForm.Get("device_code"))
	}
}

func TestPollDeviceSlowDownBumpsInterval(t *testing.T) {
	st := newTestStore(t)
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"error": "slow_down"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "expires_in": 3600})
	}))
	defer srv.Close()

	cfg := QwenOAuth()
	cfg.DeviceCodeURL = srv.URL
	cfg.TokenURL = srv.URL

	var waits []time.Duration
	flow := NewOAuthFlow(cfg, st, srv.Client())
	flow.afterFunc = func(d time.Duration) <-chan time.Time {
		waits = append(waits, d)
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}

	if _, err := flow.PollDevice(context.Background(), "dev", "ver", 5); err != nil {
		t.Fatalf("PollDevice: %v", err)
	}
	if len(waits) < 2 {
		t.Fatalf("expected >=2 waits, got %d", len(waits))
	}
	// After slow_down the interval must increase by 5s (5s -> 10s).
	if waits[1] <= waits[0] {
		t.Errorf("interval did not bump after slow_down: %v then %v", waits[0], waits[1])
	}
}

func TestPollDeviceExpiredAndDeniedTerminal(t *testing.T) {
	cases := []struct{ errCode string }{{"expired_token"}, {"access_denied"}}
	for _, tc := range cases {
		t.Run(tc.errCode, func(t *testing.T) {
			st := newTestStore(t)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]any{"error": tc.errCode})
			}))
			defer srv.Close()

			cfg := QwenOAuth()
			cfg.DeviceCodeURL = srv.URL
			cfg.TokenURL = srv.URL
			flow := noWaitFlow(cfg, st, srv.Client())

			if _, err := flow.PollDevice(context.Background(), "dev", "ver", 1); err == nil {
				t.Fatalf("PollDevice(%s) returned nil error, want terminal", tc.errCode)
			}
		})
	}
}

func TestPollDeviceDeadlineTimeout(t *testing.T) {
	st := newTestStore(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"error": "authorization_pending"})
	}))
	defer srv.Close()

	cfg := QwenOAuth()
	cfg.DeviceCodeURL = srv.URL
	cfg.TokenURL = srv.URL

	// Fake clock that advances well past the device timeout on each tick so the
	// loop terminates via the deadline with no real sleep.
	fake := time.Now()
	flow := NewOAuthFlow(cfg, st, srv.Client())
	flow.nowFunc = func() time.Time { return fake }
	flow.afterFunc = func(time.Duration) <-chan time.Time {
		fake = fake.Add(10 * time.Minute)
		ch := make(chan time.Time, 1)
		ch <- fake
		return ch
	}

	if _, err := flow.PollDevice(context.Background(), "dev", "ver", 1); err == nil {
		t.Fatal("PollDevice returned nil error past deadline, want timeout")
	} else if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("error = %v, want timeout", err)
	}
}

func TestPollDeviceKilocodeStatusCodes(t *testing.T) {
	// kilocode: GET pollUrlBase/{code}; 202=pending, 403=denied, 410=expired,
	// 2xx + status=approved + token => success.
	t.Run("approved", func(t *testing.T) {
		st := newTestStore(t)
		var calls int32
		var gotMethod string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			n := atomic.AddInt32(&calls, 1)
			if n < 2 {
				w.WriteHeader(http.StatusAccepted) // 202 pending
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"status": "approved", "token": "kc-token"})
		}))
		defer srv.Close()

		cfg := KilocodeOAuth()
		cfg.DeviceCodeURL = srv.URL
		cfg.TokenURL = srv.URL
		flow := noWaitFlow(cfg, st, srv.Client())

		tok, err := flow.PollDevice(context.Background(), "code-abc", "", 1)
		if err != nil {
			t.Fatalf("PollDevice: %v", err)
		}
		if tok.AccessToken != "kc-token" {
			t.Errorf("AccessToken = %q", tok.AccessToken)
		}
		if gotMethod != http.MethodGet {
			t.Errorf("method = %q, want GET", gotMethod)
		}
	})

	for _, tc := range []struct {
		name string
		code int
	}{{"denied", http.StatusForbidden}, {"expired", http.StatusGone}} {
		t.Run(tc.name, func(t *testing.T) {
			st := newTestStore(t)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
			}))
			defer srv.Close()

			cfg := KilocodeOAuth()
			cfg.DeviceCodeURL = srv.URL
			cfg.TokenURL = srv.URL
			flow := noWaitFlow(cfg, st, srv.Client())

			if _, err := flow.PollDevice(context.Background(), "code-abc", "", 1); err == nil {
				t.Fatalf("PollDevice(%s) returned nil error, want terminal", tc.name)
			}
		})
	}
}

func TestStartDeviceKilocodeInitiate(t *testing.T) {
	st := newTestStore(t)
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		json.NewEncoder(w).Encode(map[string]any{
			"code":            "kc-dev-code",
			"verificationUrl": "https://kilo.ai/verify",
			"expiresIn":       300,
		})
	}))
	defer srv.Close()

	cfg := KilocodeOAuth()
	cfg.DeviceCodeURL = srv.URL
	cfg.TokenURL = srv.URL
	flow := NewOAuthFlow(cfg, st, srv.Client())

	resp, err := flow.StartDevice()
	if err != nil {
		t.Fatalf("StartDevice: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if resp.DeviceCode != "kc-dev-code" {
		t.Errorf("DeviceCode = %q", resp.DeviceCode)
	}
	if resp.VerificationURI != "https://kilo.ai/verify" {
		t.Errorf("VerificationURI = %q", resp.VerificationURI)
	}
}
