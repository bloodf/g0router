package mcp

import (
	"context"
	"fmt"
	"sort"
)

type LaunchResult struct {
	Transport   Transport
	SessionID   string
	Diagnostics string
}

type Launcher struct {
	runner ProcessRunner
	http   *HTTPTransport
}

func NewLauncher(runner ProcessRunner, client HTTPDoer) *Launcher {
	return &Launcher{
		runner: runner,
		http:   NewHTTPTransport(client),
	}
}

func BuildLaunchSpec(cfg InstanceConfig) (ProcessSpec, error) {
	switch cfg.LaunchType {
	case LaunchCommand:
		if cfg.Command == "" {
			return ProcessSpec{}, ErrInvalidInstanceConfig
		}
		return ProcessSpec{
			Command: cfg.Command,
			Args:    append([]string(nil), cfg.Args...),
			Env:     redactNothing(cfg.Env),
			CWD:     cfg.CWD,
		}, nil
	case LaunchNPX:
		if cfg.Command == "" {
			return ProcessSpec{}, ErrInvalidInstanceConfig
		}
		args := append([]string{"--yes", cfg.Command}, cfg.Args...)
		return ProcessSpec{Command: "npx", Args: args, Env: redactNothing(cfg.Env), CWD: cfg.CWD}, nil
	case LaunchDocker:
		if cfg.Command == "" {
			return ProcessSpec{}, ErrInvalidInstanceConfig
		}
		args := []string{"run", "--rm", "-i"}
		for _, key := range sortedKeys(cfg.Env) {
			args = append(args, "-e", key)
		}
		args = append(args, cfg.Command)
		args = append(args, cfg.Args...)
		return ProcessSpec{Command: "docker", Args: args, Env: redactNothing(cfg.Env), CWD: cfg.CWD}, nil
	default:
		return ProcessSpec{}, ErrInvalidInstanceConfig
	}
}

func (l *Launcher) Launch(ctx context.Context, cfg InstanceConfig) (LaunchResult, error) {
	switch cfg.LaunchType {
	case LaunchCommand, LaunchNPX, LaunchDocker:
		return l.launchProcess(ctx, cfg)
	case LaunchHTTP:
		return l.launchHTTP(ctx, cfg)
	default:
		return LaunchResult{}, ErrInvalidInstanceConfig
	}
}

func (l *Launcher) launchProcess(ctx context.Context, cfg InstanceConfig) (LaunchResult, error) {
	if l.runner == nil {
		return LaunchResult{}, fmt.Errorf("mcp launcher: process runner unavailable")
	}
	spec, err := BuildLaunchSpec(cfg)
	if err != nil {
		return LaunchResult{}, err
	}
	process, err := l.runner.Start(ctx, spec)
	if err != nil {
		return LaunchResult{}, fmt.Errorf("start mcp process: %w", err)
	}
	diagnostics := ""
	if stderr := process.Stderr(); stderr != nil {
		diagnostics = stderr.String()
	}
	return LaunchResult{Transport: TransportStdio, Diagnostics: diagnostics}, nil
}

func (l *Launcher) launchHTTP(ctx context.Context, cfg InstanceConfig) (LaunchResult, error) {
	if cfg.URL == "" {
		return LaunchResult{}, ErrInvalidInstanceConfig
	}
	session, status, err := l.http.InitializeStreamable(ctx, cfg.URL, cfg.Headers)
	if err == nil {
		return LaunchResult{Transport: TransportStreamableHTTP, SessionID: session}, nil
	}
	if !shouldFallbackToSSE(status) {
		return LaunchResult{}, err
	}
	if err := l.http.InitializeSSE(ctx, cfg.URL, cfg.Headers); err != nil {
		return LaunchResult{}, err
	}
	return LaunchResult{Transport: TransportSSE}, nil
}

func shouldFallbackToSSE(status int) bool {
	switch status {
	case 400, 404, 405, 406, 415:
		return true
	default:
		return false
	}
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func redactNothing(values map[string]string) map[string]string {
	return copyMap(values)
}

func copyMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}
