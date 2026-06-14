package inference

import "regexp"

// upstreamConnectionRE matches a connection name ending in a canonical UUID
// (8-4-4-4-12 hex groups). It mirrors 9router's UPSTREAM_CONNECTION_RE
// (src/app/api/v1/models/route.js:46): connections whose name carries a
// trailing UUID suffix are "upstream" connections whose live model fetch is
// skipped (route.js:282-284). The match is anchored to the end so a UUID in
// the middle of a name does not flag the connection.
var upstreamConnectionRE = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// IsUpstreamConnection reports whether a connection name marks an upstream
// connection (UUID suffix), for which the live model-catalog fetch is skipped
// (PAR-ROUTE-060). It is a pure, deterministic guard with no I/O.
func IsUpstreamConnection(name string) bool {
	return upstreamConnectionRE.MatchString(name)
}
