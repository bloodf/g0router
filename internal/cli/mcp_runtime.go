package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"sync"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/store"
)

type defaultMCPRuntime struct {
	clients   *mcp.ClientManager
	tools     *mcp.ToolManager
	connector *mcpLauncherConnector
}

func newDefaultMCPRuntime() *defaultMCPRuntime {
	launcher := mcp.NewLauncher(commandProcessRunner{}, http.DefaultClient)
	connector := &mcpLauncherConnector{launcher: launcher}
	return &defaultMCPRuntime{
		clients:   mcp.NewClientManager(connector),
		tools:     mcp.NewToolManager(),
		connector: connector,
	}
}

type mcpLauncherConnector struct {
	mu              sync.RWMutex
	launcher        *mcp.Launcher
	instanceConfigs map[string]mcp.InstanceConfig
}

func (c *mcpLauncherConnector) RememberInstanceConfig(cfg mcp.InstanceConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.instanceConfigs == nil {
		c.instanceConfigs = make(map[string]mcp.InstanceConfig)
	}
	c.instanceConfigs[cfg.ID] = cfg
}

func (c *mcpLauncherConnector) ForgetInstanceConfig(instanceID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.instanceConfigs, instanceID)
}

func (c *mcpLauncherConnector) Connect(ctx context.Context, cfg mcp.ClientConfig) (mcp.Client, error) {
	instanceCfg, err := c.instanceConfig(cfg)
	if err != nil {
		return nil, err
	}
	result, err := c.launcher.Launch(ctx, instanceCfg)
	if err != nil {
		return nil, err
	}
	if result.Transport == mcp.TransportStdio {
		return mcp.NewStdioClient(result.Process), nil
	}
	if result.Transport == mcp.TransportStreamableHTTP {
		return mcp.NewStreamableHTTPClient(c.launcher.HTTPClient(), instanceCfg.URL, instanceCfg.Headers, result.SessionID, true), nil
	}
	if result.Transport == mcp.TransportSSE {
		return mcp.NewSSEClient(c.launcher.HTTPClient(), instanceCfg.URL, instanceCfg.Headers), nil
	}
	return &launchedMCPClient{process: result.Process}, nil
}

func (c *mcpLauncherConnector) instanceConfig(cfg mcp.ClientConfig) (mcp.InstanceConfig, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.instanceConfigs != nil {
		if instanceCfg, ok := c.instanceConfigs[cfg.ID]; ok {
			return instanceCfg, nil
		}
	}
	return clientInstanceConfig(cfg)
}

func (r *defaultMCPRuntime) RegisterInstance(ctx context.Context, instance *store.MCPInstance) (mcp.Manifest, error) {
	if r == nil || r.clients == nil || r.tools == nil || r.connector == nil {
		return mcp.Manifest{}, mcp.ErrInvalidDiscovery
	}
	cfg := instance.Config()
	r.connector.RememberInstanceConfig(cfg)
	manifest, err := r.clients.Register(ctx, mcp.ClientConfig{
		ID:        instance.ID,
		Name:      instance.Name,
		Transport: instance.Transport,
	})
	if err != nil {
		return mcp.Manifest{}, err
	}
	if err := r.tools.RegisterManifest(manifest); err != nil {
		_ = r.clients.Close(instance.ID)
		return mcp.Manifest{}, err
	}
	registered, ok := r.clients.Client(instance.ID)
	if !ok {
		return mcp.Manifest{}, mcp.ErrClientNotFound
	}
	r.tools.RegisterClient(instance.ID, registered)
	return manifest, nil
}

func (r *defaultMCPRuntime) CloseInstance(instanceID string) error {
	if r == nil || r.clients == nil || r.tools == nil || r.connector == nil {
		return mcp.ErrInvalidDiscovery
	}
	r.tools.UnregisterClient(instanceID)
	r.connector.ForgetInstanceConfig(instanceID)
	if err := r.clients.Close(instanceID); err != nil {
		return err
	}
	return nil
}

func (r *defaultMCPRuntime) ReapplyInstanceCredentials(ctx context.Context, s *store.Store, instanceID string) (mcp.Manifest, error) {
	if s == nil {
		return mcp.Manifest{}, store.ErrNotFound
	}
	instance, err := s.GetMCPInstance(instanceID)
	if err != nil {
		return mcp.Manifest{}, err
	}
	runtimeInstance, err := mcpInstanceForRuntime(ctx, s, instance)
	if err != nil {
		return mcp.Manifest{}, err
	}
	if err := r.CloseInstance(instanceID); err != nil && !errors.Is(err, mcp.ErrClientNotFound) {
		return mcp.Manifest{}, err
	}
	return r.RegisterInstance(ctx, runtimeInstance)
}

func clientInstanceConfig(cfg mcp.ClientConfig) (mcp.InstanceConfig, error) {
	switch cfg.Transport {
	case mcp.TransportStdio:
		if cfg.Command == "" {
			return mcp.InstanceConfig{}, mcp.ErrInvalidClientConfig
		}
		return mcp.InstanceConfig{
			ID:         cfg.ID,
			Name:       cfg.Name,
			ServerKey:  cfg.ID,
			LaunchType: mcp.LaunchCommand,
			Transport:  cfg.Transport,
			Command:    cfg.Command,
			Args:       append([]string(nil), cfg.Args...),
			Env:        copyEnvMap(cfg.Env),
		}, nil
	case mcp.TransportStreamableHTTP, mcp.TransportSSE:
		if cfg.URL == "" {
			return mcp.InstanceConfig{}, mcp.ErrInvalidClientConfig
		}
		return mcp.InstanceConfig{
			ID:         cfg.ID,
			Name:       cfg.Name,
			ServerKey:  cfg.ID,
			LaunchType: mcp.LaunchHTTP,
			Transport:  cfg.Transport,
			URL:        cfg.URL,
			Env:        copyEnvMap(cfg.Env),
		}, nil
	default:
		return mcp.InstanceConfig{}, mcp.ErrInvalidClientConfig
	}
}

type launchedMCPClient struct {
	process mcp.Process
}

func (c *launchedMCPClient) ListTools(context.Context) ([]mcp.Tool, error) {
	return nil, nil
}

func (c *launchedMCPClient) CallTool(context.Context, mcp.CallRequest) (mcp.CallResult, error) {
	return mcp.CallResult{}, mcp.ErrToolNotFound
}

func (c *launchedMCPClient) Close() error {
	if c.process == nil {
		return nil
	}
	return c.process.Close()
}

type commandProcessRunner struct{}

func (commandProcessRunner) Start(ctx context.Context, spec mcp.ProcessSpec) (mcp.Process, error) {
	if spec.Command == "" {
		return nil, mcp.ErrInvalidInstanceConfig
	}
	cmd := exec.CommandContext(ctx, spec.Command, spec.Args...)
	cmd.Env = processEnv(spec.Env)
	cmd.Dir = spec.CWD
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open command %q stdin: %w", spec.Command, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open command %q stdout: %w", spec.Command, err)
	}

	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start command %q: %w", spec.Command, err)
	}
	return &commandProcess{cmd: cmd, stdin: stdin, stdout: stdout, stderr: stderr}, nil
}

type commandProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr *bytes.Buffer
}

func (p *commandProcess) Stdin() io.WriteCloser {
	return p.stdin
}

func (p *commandProcess) Stdout() io.ReadCloser {
	return p.stdout
}

func (p *commandProcess) Stderr() *bytes.Buffer {
	return p.stderr
}

func (p *commandProcess) Close() error {
	if p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	if p.stdin != nil {
		_ = p.stdin.Close()
	}
	if p.stdout != nil {
		_ = p.stdout.Close()
	}
	if err := p.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("kill mcp process: %w", err)
	}
	if err := p.cmd.Wait(); err != nil && !isProcessExitAfterClose(err) {
		return fmt.Errorf("wait for mcp process: %w", err)
	}
	return nil
}

func processEnv(extra map[string]string) []string {
	env := os.Environ()
	for _, key := range sortedStringKeys(extra) {
		env = append(env, key+"="+extra[key])
	}
	return env
}

func sortedStringKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func copyEnvMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func isProcessExitAfterClose(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr)
}
