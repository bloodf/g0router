package mcp

import (
	"path/filepath"
	"strings"
)

// allowedMCPCommands is the set of launcher binaries an MCP stdio plugin may
// spawn. It mirrors 9router's ALLOWED_MCP_COMMANDS (coworkPlugins.js:67):
// npx, node, uvx, python, python3, bunx, bun. Any other binary is rejected
// before any process is spawned.
var allowedMCPCommands = map[string]struct{}{
	"npx":     {},
	"node":    {},
	"uvx":     {},
	"python":  {},
	"python3": {},
	"bunx":    {},
	"bun":     {},
}

// shellMetaChars are characters that, if present in a command string, indicate a
// shell-injection attempt. A spawnable MCP command must contain none of them.
const shellMetaChars = ";|&$`><()\n\r\t "

// isAllowedCommand reports whether cmd resolves to an allowlisted launcher. It is
// PURE — no I/O. It HARDENS 9router's raw path.basename gate (coworkPlugins.js:67):
//
//   - the base name (filepath.Base) must be in allowedMCPCommands;
//   - the command must contain NO shell metacharacters (; | & $ ` > < ( ) and
//     whitespace), so "npx; rm -rf /" / "$(touch x)" / "bash -c" are rejected;
//   - the command must be EITHER a bare base name (no separators) OR an absolute,
//     already-clean path (filepath.IsAbs && filepath.Clean(cmd)==cmd), so a
//     relative path ("./npx", "../npx") or a non-clean path is rejected, while
//     "/usr/local/bin/npx" passes.
//
// This deliberately rejects "/bin/rm" (base "rm" not allowlisted), "./npx"
// (relative), and any metacharacter-bearing command — a parity improvement over
// 9router recorded in open-questions.md (ESC-ALLOWLIST).
func isAllowedCommand(cmd string) bool {
	if cmd == "" {
		return false
	}
	if strings.ContainsAny(cmd, shellMetaChars) {
		return false
	}
	if _, ok := allowedMCPCommands[filepath.Base(cmd)]; !ok {
		return false
	}
	// Bare base name (no path separators) is permitted.
	if !strings.ContainsRune(cmd, filepath.Separator) {
		return true
	}
	// Otherwise it must be an absolute, already-clean path.
	return filepath.IsAbs(cmd) && filepath.Clean(cmd) == cmd
}
