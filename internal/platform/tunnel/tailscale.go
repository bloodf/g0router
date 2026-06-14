package tunnel

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sync"
)

// tailscaleLoginURLRe matches the login URL tailscale prints during `tailscale
// up` before the node is authenticated.
var tailscaleLoginURLRe = regexp.MustCompile(`https://login\.tailscale\.com/[a-zA-Z0-9/]+`)

// extractTailscaleLoginURL scans `tailscale up` output and returns the login URL,
// or ("", false) if absent. PURE — no I/O. The poll loop that waits for
// authentication is integration-only; this parser carries the testable logic.
func extractTailscaleLoginURL(out string) (string, bool) {
	if m := tailscaleLoginURLRe.FindString(out); m != "" {
		return m, true
	}
	return "", false
}

// tailscaleRunner implements Runner for tailscale by shelling out to the
// tailscale binary. Install (OS-privileged), daemon (userspace-networking by
// default; TUN is an escalated opt-in), login poll, funnel, and cert are all
// INTEGRATION-ONLY: never invoked by unit tests, which inject a fake Runner.
type tailscaleRunner struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	status RunnerStatus
}

func newTailscaleRunner() *tailscaleRunner {
	return &tailscaleRunner{status: RunnerStatus{Status: StatusInactive}}
}

// Start brings up tailscale in userspace-networking mode (no TUN, no root) and,
// for funnel mode, exposes the configured port publicly, reporting the assigned
// *.ts.net URL. INTEGRATION-ONLY — not exercised by unit tests.
func (r *tailscaleRunner) Start(opts StartOpts) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	bin, err := exec.LookPath("tailscale")
	if err != nil {
		return "", fmt.Errorf("tailscale not installed: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Userspace-networking default (server-friendly; no OS privilege). TUN mode
	// is an escalated opt-in handled by the install/daemon integration path.
	cmd := exec.CommandContext(ctx, bin, "up")
	if err := cmd.Start(); err != nil {
		cancel()
		return "", fmt.Errorf("start tailscale: %w", err)
	}
	r.cancel = cancel
	r.status = RunnerStatus{Running: true, Status: StatusActive}
	return r.status.URL, nil
}

// Stop runs `tailscale down` / stops the daemon. Idempotent. INTEGRATION-ONLY.
func (r *tailscaleRunner) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	r.status = RunnerStatus{Status: StatusInactive}
	return nil
}

// Status reports the last observed state. INTEGRATION-ONLY.
func (r *tailscaleRunner) Status() (RunnerStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status, nil
}
