package tunnel

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

var urlPattern = regexp.MustCompile(`https://[a-zA-Z0-9._\-]+\.[a-zA-Z]{2,}`)

// Supervisor manages the lifecycle of a tunnel binary and captures its public URL.
type Supervisor struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
	mu     sync.Mutex
	url    string
	done   chan struct{}
}

// Start runs the tunnel binary and captures its public URL from stdout.
// binPath must be an absolute path. args is a fixed slice — no shell interpolation.
func (s *Supervisor) Start(ctx context.Context, binPath string, args []string) error {
	if !filepath.IsAbs(binPath) {
		return fmt.Errorf("binary path must be absolute: %s", binPath)
	}

	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.done = make(chan struct{})

	cmd := exec.CommandContext(ctx, binPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("start process: %w", err)
	}

	s.cmd = cmd
	go s.captureOutput(stdout)
	go s.discardOutput(stderr)

	go func() {
		_ = cmd.Wait()
		close(s.done)
	}()

	return nil
}

func (s *Supervisor) captureOutput(r io.ReadCloser) {
	defer r.Close()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if match := urlPattern.FindString(line); match != "" {
			s.mu.Lock()
			s.url = match
			s.mu.Unlock()
		}
	}
}

func (s *Supervisor) discardOutput(r io.ReadCloser) {
	defer r.Close()
	_, _ = io.Copy(io.Discard, r)
}

// Stop terminates the supervised process.
func (s *Supervisor) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	if s.done != nil {
		select {
		case <-s.done:
		case <-time.After(5 * time.Second):
			return fmt.Errorf("timeout waiting for process to exit")
		}
	}
	return nil
}

// URL returns the captured tunnel URL.
func (s *Supervisor) URL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.url
}
