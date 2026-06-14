package mcp

// ProcessRunner abstracts spawning + driving one MCP stdio child process. The
// REAL impl (osProcessRunner, process.go) shells out via os/exec
// (StdinPipe/StdoutPipe/StderrPipe + cmd.Wait); the TEST impl is a deterministic
// in-memory fake (canned stdout frames, a controllable exit) — so the launcher,
// the bridge broadcast, the filter, and the lifecycle are unit-tested WITHOUT
// spawning any process. Mirrors tunnel.Runner (tunnel/runner.go:15).
type ProcessRunner interface {
	// Start spawns the child for spec (command+args+env). It returns a Process
	// whose stdout frames are delivered to spec.OnFrame and whose exit invokes
	// spec.OnExit.
	Start(spec ProcessSpec) (Process, error)
}

// Process is one running (or fake) child process; the Bridge drives it.
type Process interface {
	// Write sends a JSON-RPC frame to the child's stdin (newline-delimited).
	Write(frame []byte) error
	// IsRunning reports liveness: not killed && exit code not yet observed
	// (PAR-MCP-051; mirrors !proc.killed && proc.exitCode === null).
	IsRunning() bool
	// Stop kills the child. Idempotent.
	Stop() error
}

// ProcessSpec describes a child process to spawn plus the callbacks that wire its
// stdout/stderr/exit back into the Bridge.
type ProcessSpec struct {
	// Command is the allowlist-validated base name (npx/node/uvx/...).
	Command string
	Args    []string
	Env     map[string]string
	// OnFrame receives each newline-delimited stdout JSON-RPC frame (PAR-MCP-007).
	OnFrame func(frame []byte)
	// OnStderr receives child stderr lines for logging (PAR-MCP-052).
	OnStderr func(line string)
	// OnExit fires once when the child exits, with its exit code (PAR-MCP-053).
	OnExit func(code int)
}
