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

	"github.com/bloodf/g0router/internal/mcp"
)

func newDefaultMCPRuntime() (*mcp.ClientManager, *mcp.ToolManager) {
	launcher := mcp.NewLauncher(commandProcessRunner{}, http.DefaultClient)
	return mcp.NewClientManager(mcpLauncherConnector{launcher: launcher}), mcp.NewToolManager()
}

type mcpLauncherConnector struct {
	launcher *mcp.Launcher
}

func (c mcpLauncherConnector) Connect(ctx context.Context, cfg mcp.ClientConfig) (mcp.Client, error) {
	instanceCfg, err := clientInstanceConfig(cfg)
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
	return &launchedMCPClient{process: result.Process}, nil
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
