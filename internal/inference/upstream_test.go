package inference

import "testing"

// TestIsUpstreamConnection mirrors 9router's UPSTREAM_CONNECTION_RE
// (src/app/api/v1/models/route.js:46,282-284): a connection name carrying a
// trailing UUID suffix marks an upstream connection whose live model fetch is
// skipped. The check is pure and deterministic (no I/O).
func TestIsUpstreamConnection(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		// UUID-suffixed names → upstream (skip live fetch).
		{"plain uuid suffix", "kiro-550e8400-e29b-41d4-a716-446655440000", true},
		{"dash separated uuid", "my-account-6ba7b810-9dad-11d1-80b4-00c04fd430c8", true},
		{"uppercase uuid suffix", "Acct-6BA7B811-9DAD-11D1-80B4-00C04FD430C8", true},
		{"bare uuid only", "550e8400-e29b-41d4-a716-446655440000", true},
		// Normal names → not upstream (live fetch runs).
		{"simple name", "kiro", false},
		{"hyphenated name", "my-account-1", false},
		{"empty", "", false},
		{"uuid-like but too short", "abc-123", false},
		{"uuid prefix not suffix", "550e8400-e29b-41d4-a716-446655440000-prod", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsUpstreamConnection(c.in); got != c.want {
				t.Errorf("IsUpstreamConnection(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}
