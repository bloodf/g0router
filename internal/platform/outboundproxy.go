package platform

import (
	"fmt"
	"net"
)

// IPResolver resolves a hostname to a set of IP addresses. It is an injectable
// seam so callers (and tests) control DNS resolution deterministically.
type IPResolver func(host string) ([]net.IP, error)

// defaultResolver resolves hostnames via the system resolver. It is used when
// IsBlockedTarget is called with a nil resolver and the host is not a literal IP.
func defaultResolver(host string) ([]net.IP, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", host, err)
	}
	return ips, nil
}

// IsBlockedIP reports whether ip falls in a range disallowed for outbound
// connections (SSRF mitigation, PAR-AUTH-020). It blocks loopback, private,
// link-local, unspecified, and multicast addresses (covering cloud-metadata
// 169.254.169.254/169.254.169.169 via link-local). Global-unicast public
// addresses are allowed. Pure and deterministic.
func IsBlockedIP(ip net.IP) (blocked bool, reason string) {
	switch {
	case ip == nil:
		return true, "unparseable IP"
	case ip.IsUnspecified():
		return true, "unspecified address"
	case ip.IsLoopback():
		return true, "loopback address"
	case ip.IsLinkLocalUnicast(), ip.IsLinkLocalMulticast():
		return true, "link-local address"
	case ip.IsPrivate():
		return true, "private address"
	case ip.IsMulticast():
		return true, "multicast address"
	default:
		return false, ""
	}
}

// IsBlockedTarget reports whether host is disallowed for outbound connections.
// host may be a bare host, a host:port pair, or a literal IP. A literal IP is
// evaluated directly; a hostname is resolved via resolver (or the system
// resolver when resolver is nil) and blocked if ANY resolved IP is blocked.
func IsBlockedTarget(host string, resolver IPResolver) (blocked bool, reason string, err error) {
	hostname := hostOnly(host)
	if hostname == "" {
		return true, "empty host", nil
	}

	if ip := net.ParseIP(hostname); ip != nil {
		b, r := IsBlockedIP(ip)
		return b, r, nil
	}

	if resolver == nil {
		resolver = defaultResolver
	}
	ips, err := resolver(hostname)
	if err != nil {
		return false, "", fmt.Errorf("resolve target %s: %w", hostname, err)
	}
	if len(ips) == 0 {
		return true, "host did not resolve to any IP", nil
	}
	for _, ip := range ips {
		if b, r := IsBlockedIP(ip); b {
			return true, fmt.Sprintf("%s resolves to a blocked %s", hostname, r), nil
		}
	}
	return false, "", nil
}

// hostOnly strips an optional :port suffix from host, returning the bare host.
// It tolerates bare hosts (no port) and bracketed IPv6 literals.
func hostOnly(host string) string {
	if host == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}
