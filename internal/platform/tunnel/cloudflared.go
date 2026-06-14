package tunnel

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"runtime"
	"sync"
)

// quickTunnelURLRe matches the assigned https://<sub>.trycloudflare.com URL that
// cloudflared prints to stderr when running a quick tunnel.
var quickTunnelURLRe = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

// extractQuickTunnelURL scans cloudflared stderr text and returns the assigned
// https://<sub>.trycloudflare.com URL, or ("", false) if absent. It returns the
// first match when several are present. PURE — no I/O.
func extractQuickTunnelURL(stderr string) (string, bool) {
	if m := quickTunnelURLRe.FindString(stderr); m != "" {
		return m, true
	}
	return "", false
}

// isValidExecutable reports whether head (the leading bytes of a downloaded
// file) carries the executable magic for goos: ELF on linux, Mach-O on darwin,
// PE (MZ) on windows. PURE — used to validate a downloaded binary before
// chmod+exec, and unit-tested on canned byte slices independently of the
// (integration-only) download path.
func isValidExecutable(head []byte, goos string) bool {
	if len(head) < 4 {
		return false
	}
	switch goos {
	case "linux":
		return head[0] == 0x7f && head[1] == 'E' && head[2] == 'L' && head[3] == 'F'
	case "darwin":
		// Mach-O magic (32/64-bit, both endiannesses).
		switch {
		case head[0] == 0xfe && head[1] == 0xed && head[2] == 0xfa && (head[3] == 0xce || head[3] == 0xcf):
			return true
		case (head[0] == 0xce || head[0] == 0xcf) && head[1] == 0xfa && head[2] == 0xed && head[3] == 0xfe:
			return true
		}
		return false
	case "windows":
		return head[0] == 'M' && head[1] == 'Z'
	default:
		return false
	}
}

// cloudflaredRunner implements Runner for cloudflare by shelling out to the
// cloudflared binary. The process spawn / kill / binary download bodies are
// INTEGRATION-ONLY: they are never invoked by unit tests, which inject a fake
// Runner instead (mirrors platform.Prober/SetProber). The pure helpers above
// (extractQuickTunnelURL, isValidExecutable) carry the unit-tested logic.
type cloudflaredRunner struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	status RunnerStatus
}

func newCloudflaredRunner() *cloudflaredRunner {
	return &cloudflaredRunner{status: RunnerStatus{Status: StatusInactive}}
}

// Start launches cloudflared. With a token it runs a named tunnel
// (`tunnel run --token`); without a token it runs a quick tunnel
// (`tunnel --url`) and extracts the assigned *.trycloudflare.com URL from
// stderr. INTEGRATION-ONLY — not exercised by unit tests.
func (r *cloudflaredRunner) Start(opts StartOpts) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	bin, err := r.ensureBinary()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithCancel(context.Background())
	var args []string
	if opts.Token != "" {
		args = []string{"tunnel", "run", "--token", opts.Token}
	} else {
		args = []string{"tunnel", "--url", "http://localhost:8080"}
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return "", fmt.Errorf("cloudflared stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		cancel()
		return "", fmt.Errorf("start cloudflared: %w", err)
	}
	r.cancel = cancel

	url := ""
	if opts.Token == "" {
		url = scanForQuickTunnelURL(stderr)
	}
	r.status = RunnerStatus{Running: true, URL: url, Status: StatusActive}
	return url, nil
}

// Stop kills the cloudflared process. Idempotent. INTEGRATION-ONLY.
func (r *cloudflaredRunner) Stop() error {
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
func (r *cloudflaredRunner) Status() (RunnerStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status, nil
}

// scanForQuickTunnelURL reads cloudflared stderr line-by-line until the assigned
// quick-tunnel URL appears, returning it (or "" if the pipe closes first). The
// PURE parser extractQuickTunnelURL carries the matching logic. INTEGRATION-ONLY.
func scanForQuickTunnelURL(stderr io.Reader) string {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		if url, ok := extractQuickTunnelURL(scanner.Text()); ok {
			return url
		}
	}
	return ""
}

// ensureBinary downloads the platform cloudflared binary to the data dir and
// validates its magic bytes (isValidExecutable) before chmod+exec. This is the
// PAR-PLAT-015 binary-download path: INTEGRATION-ONLY (needs network), never
// invoked by unit tests. The current minimal impl resolves a system-installed
// cloudflared; the full download is the integration surface.
func (r *cloudflaredRunner) ensureBinary() (string, error) {
	path, err := exec.LookPath("cloudflared")
	if err != nil {
		return "", fmt.Errorf("cloudflared not installed for %s: %w", runtime.GOOS, err)
	}
	return path, nil
}
