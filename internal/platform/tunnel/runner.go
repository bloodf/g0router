// Package tunnel implements the cloudflared + tailscale tunnel subsystem: an
// injectable Runner abstraction over the external tunnel binaries, a state
// machine that enables/disables/reports tunnel status, and the persistence of
// tunnel state. The Runner seam mirrors platform.Prober (proxypools.go:18): the
// real impls shell out to cloudflared/tailscale, while tests inject a
// deterministic fake so the state machine + admin API are unit-tested WITHOUT
// spawning any process, downloading any binary, or touching the network.
package tunnel

// Runner abstracts the lifecycle of a single tunnel process (cloudflared or
// tailscale). The REAL impl shells out to the external binary; the TEST impl is
// a deterministic fake returning canned status/URL — so the admin handlers + the
// enable/disable/status/health state machine are unit-tested WITHOUT spawning
// any process or touching the network. Mirrors platform.Prober (proxypools.go:18).
type Runner interface {
	// Start enables the tunnel; for cloudflared-quick it returns the extracted
	// *.trycloudflare.com URL. Returns the resolved public URL (may be "").
	Start(opts StartOpts) (url string, err error)
	// Stop disables/kills the tunnel process. Idempotent.
	Stop() error
	// Status reports the live state without side effects.
	Status() (RunnerStatus, error)
}

// StartOpts configures a Runner.Start call.
type StartOpts struct {
	Type  string // "cloudflare" | "tailscale"
	Token string // cloudflared named-tunnel token (from token_enc); "" → quick tunnel
	Mode  string // "named"|"quick" (cloudflare) | "funnel"|"serve" (tailscale)
}

// RunnerStatus is the side-effect-free view of a Runner's current state.
type RunnerStatus struct {
	Running bool
	URL     string
	Status  string // "inactive"|"starting"|"active"|"error"
	LastErr string // human-readable; NO secrets
}

// Tunnel status constants — the four states the state machine cycles through.
const (
	StatusInactive = "inactive"
	StatusStarting = "starting"
	StatusActive   = "active"
	StatusError    = "error"
)

// TypeCloudflare and TypeTailscale are the two fixed tunnel types.
const (
	TypeCloudflare = "cloudflare"
	TypeTailscale  = "tailscale"
)
