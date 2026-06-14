package platform

import "net"

// IPResolver resolves a hostname to a set of IP addresses. It is an injectable
// seam so callers (and tests) control DNS resolution deterministically.
type IPResolver func(host string) ([]net.IP, error)

// IsBlockedIP reports whether ip falls in a range disallowed for outbound
// connections (SSRF mitigation). Stub: implementation lands in T-ssrf STEP(b).
func IsBlockedIP(ip net.IP) (blocked bool, reason string) {
	return false, ""
}

// IsBlockedTarget reports whether host (optionally host:port, a literal IP, or a
// hostname resolved via resolver) is disallowed for outbound connections.
// Stub: implementation lands in T-ssrf STEP(b).
func IsBlockedTarget(host string, resolver IPResolver) (blocked bool, reason string, err error) {
	return false, "", nil
}
