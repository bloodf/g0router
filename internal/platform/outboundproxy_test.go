package platform

import (
	"net"
	"testing"
)

func TestIsBlockedIP(t *testing.T) {
	cases := []struct {
		ip      string
		blocked bool
	}{
		// Loopback.
		{"127.0.0.1", true},
		{"127.1.2.3", true},
		{"::1", true},
		// Private IPv4.
		{"10.1.2.3", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"192.168.1.1", true},
		// Private IPv6 (fc00::/7).
		{"fc00::1", true},
		{"fd12:3456::1", true},
		// Link-local.
		{"169.254.0.1", true},
		{"169.254.169.254", true}, // cloud metadata
		{"169.254.169.169", true}, // cloud metadata
		{"fe80::1", true},
		// Unspecified.
		{"0.0.0.0", true},
		{"::", true},
		// Multicast.
		{"224.0.0.1", true},
		{"ff02::1", true},
		// Public — allowed.
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"93.184.216.34", false},
		{"2606:2800:220:1:248:1893:25c8:1946", false},
		// Public IPv4 just outside private ranges.
		{"172.15.0.1", false},
		{"172.32.0.1", false},
		{"11.0.0.1", false},
	}
	for _, c := range cases {
		ip := net.ParseIP(c.ip)
		if ip == nil {
			t.Fatalf("ParseIP(%q) returned nil", c.ip)
		}
		blocked, reason := IsBlockedIP(ip)
		if blocked != c.blocked {
			t.Errorf("IsBlockedIP(%s) = %v (reason %q); want %v", c.ip, blocked, reason, c.blocked)
		}
		if blocked && reason == "" {
			t.Errorf("IsBlockedIP(%s) blocked but reason empty", c.ip)
		}
	}
}

func TestIsBlockedTargetLiteralIP(t *testing.T) {
	// A literal blocked IP is refused without any resolver call.
	blocked, reason, err := IsBlockedTarget("127.0.0.1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !blocked || reason == "" {
		t.Errorf("literal 127.0.0.1: blocked=%v reason=%q; want blocked with reason", blocked, reason)
	}

	// A literal public IP is allowed.
	blocked, _, err = IsBlockedTarget("8.8.8.8", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if blocked {
		t.Errorf("literal 8.8.8.8: blocked=true; want allowed")
	}

	// host:port form is parsed.
	blocked, _, err = IsBlockedTarget("10.0.0.5:8080", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !blocked {
		t.Errorf("10.0.0.5:8080: blocked=false; want blocked")
	}
}

func TestIsBlockedTargetHostnameResolver(t *testing.T) {
	// Hostname resolving to a private IP is blocked.
	resolver := func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("10.0.0.9")}, nil
	}
	blocked, reason, err := IsBlockedTarget("internal.example.com", resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !blocked || reason == "" {
		t.Errorf("internal hostname: blocked=%v reason=%q; want blocked", blocked, reason)
	}

	// Hostname resolving to a public IP is allowed.
	resolver = func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	}
	blocked, _, err = IsBlockedTarget("example.com", resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if blocked {
		t.Errorf("public hostname: blocked=true; want allowed")
	}

	// Any blocked IP among the resolved set blocks the whole target.
	resolver = func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("93.184.216.34"), net.ParseIP("127.0.0.1")}, nil
	}
	blocked, _, err = IsBlockedTarget("mixed.example.com", resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !blocked {
		t.Errorf("mixed hostname: blocked=false; want blocked (one private IP)")
	}
}

func TestIsBlockedTargetResolverError(t *testing.T) {
	resolver := func(host string) ([]net.IP, error) {
		return nil, net.UnknownNetworkError("boom")
	}
	_, _, err := IsBlockedTarget("nope.example.com", resolver)
	if err == nil {
		t.Fatalf("expected resolver error to propagate")
	}
}
