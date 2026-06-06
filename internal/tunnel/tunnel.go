package tunnel

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

var (
	portPattern       = regexp.MustCompile(`^[0-9]+$`)
	tunnelNamePattern = regexp.MustCompile(`^[a-z0-9\-]{1,63}$`)
)

func validatePort(port string) error {
	if !portPattern.MatchString(port) {
		return fmt.Errorf("port must be numeric")
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}
	if p < 1 || p > 65535 {
		return fmt.Errorf("port out of range")
	}
	return nil
}

func validateTunnelName(name string) error {
	if !tunnelNamePattern.MatchString(name) {
		return fmt.Errorf("invalid tunnel name")
	}
	return nil
}

type tunnelStore interface {
	ListTunnelConfigs() ([]store.TunnelConfig, error)
	UpsertTunnelConfig(cfg store.TunnelConfig) error
	UpdateTunnelStatus(tunnelType, status, lastError string) error
}

// Manager orchestrates tunnel binaries and persists status to the store.
type Manager struct {
	store      tunnelStore
	dataDir    string
	supervisor *Supervisor
}

// NewManager creates a new tunnel manager.
func NewManager(store tunnelStore, dataDir string) *Manager {
	return &Manager{
		store:   store,
		dataDir: dataDir,
	}
}

// StartCloudflare ensures the cloudflared binary is present, starts it, waits
// for the public URL, and persists the active status.
func (m *Manager) StartCloudflare(port string) (string, error) {
	if err := validatePort(port); err != nil {
		return "", fmt.Errorf("invalid port: %w", err)
	}

	binPath := filepath.Join(m.dataDir, "bin", "cloudflared")
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		var derr error
		binPath, derr = DownloadCloudflared(m.dataDir)
		if derr != nil {
			_ = m.store.UpdateTunnelStatus("cloudflare", "error", derr.Error())
			return "", fmt.Errorf("download cloudflared: %w", derr)
		}
	}

	absPath, err := filepath.Abs(binPath)
	if err != nil {
		return "", fmt.Errorf("resolve binary path: %w", err)
	}

	m.supervisor = &Supervisor{}
	args := []string{"tunnel", "--url", "http://localhost:" + port}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := m.supervisor.Start(ctx, absPath, args); err != nil {
		m.supervisor = nil
		_ = m.store.UpdateTunnelStatus("cloudflare", "error", err.Error())
		return "", fmt.Errorf("start cloudflared: %w", err)
	}

	url, err := m.waitForURL()
	if err != nil {
		_ = m.supervisor.Stop()
		m.supervisor = nil
		_ = m.store.UpdateTunnelStatus("cloudflare", "error", err.Error())
		return "", err
	}

	_ = m.store.UpdateTunnelStatus("cloudflare", "active", "")
	_ = m.store.UpsertTunnelConfig(store.TunnelConfig{
		Type:      "cloudflare",
		URL:       url,
		Status:    "active",
		IsEnabled: true,
	})

	return url, nil
}

// StopCloudflare stops the cloudflare tunnel and marks it inactive.
func (m *Manager) StopCloudflare() error {
	if m.supervisor == nil {
		return nil
	}
	if err := m.supervisor.Stop(); err != nil {
		_ = m.store.UpdateTunnelStatus("cloudflare", "error", err.Error())
		return fmt.Errorf("stop cloudflared: %w", err)
	}
	_ = m.store.UpdateTunnelStatus("cloudflare", "inactive", "")
	m.supervisor = nil
	return nil
}

// StartTailscale checks that tailscale is on PATH, starts it, waits for the
// public URL, and persists the active status.
func (m *Manager) StartTailscale(port string) (string, error) {
	if err := validatePort(port); err != nil {
		return "", fmt.Errorf("invalid port: %w", err)
	}

	binPath, err := exec.LookPath("tailscale")
	if err != nil {
		_ = m.store.UpdateTunnelStatus("tailscale", "error", "tailscale not found on PATH")
		return "", fmt.Errorf("tailscale not found on PATH")
	}

	absPath, err := filepath.Abs(binPath)
	if err != nil {
		return "", fmt.Errorf("resolve binary path: %w", err)
	}

	m.supervisor = &Supervisor{}
	args := []string{"funnel", port}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := m.supervisor.Start(ctx, absPath, args); err != nil {
		m.supervisor = nil
		_ = m.store.UpdateTunnelStatus("tailscale", "error", err.Error())
		return "", fmt.Errorf("start tailscale: %w", err)
	}

	url, err := m.waitForURL()
	if err != nil {
		_ = m.supervisor.Stop()
		m.supervisor = nil
		_ = m.store.UpdateTunnelStatus("tailscale", "error", err.Error())
		return "", err
	}

	_ = m.store.UpdateTunnelStatus("tailscale", "active", "")
	_ = m.store.UpsertTunnelConfig(store.TunnelConfig{
		Type:      "tailscale",
		URL:       url,
		Status:    "active",
		IsEnabled: true,
	})

	return url, nil
}

// StopTailscale stops the tailscale funnel and marks it inactive.
func (m *Manager) StopTailscale() error {
	if m.supervisor == nil {
		return nil
	}
	if err := m.supervisor.Stop(); err != nil {
		_ = m.store.UpdateTunnelStatus("tailscale", "error", err.Error())
		return fmt.Errorf("stop tailscale: %w", err)
	}
	_ = m.store.UpdateTunnelStatus("tailscale", "inactive", "")
	m.supervisor = nil
	return nil
}

func (m *Manager) waitForURL() (string, error) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			if url := m.supervisor.URL(); url != "" {
				return url, nil
			}
		case <-timer.C:
			return "", fmt.Errorf("timeout waiting for tunnel URL")
		}
	}
}
