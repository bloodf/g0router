package mcp

import "testing"

// TestIsAllowedCommandAccepts asserts every permitted launcher (bare base name
// and an absolute clean path to a permitted binary) is accepted.
func TestIsAllowedCommandAccepts(t *testing.T) {
	accepts := []string{
		"npx", "node", "uvx", "python", "python3", "bunx", "bun",
		"/usr/local/bin/npx",
	}
	for _, cmd := range accepts {
		if !isAllowedCommand(cmd) {
			t.Errorf("isAllowedCommand(%q) = false, want true", cmd)
		}
	}
}

// TestIsAllowedCommandRejects asserts the security boundary: non-allowlisted
// binaries, relative paths, absolute arbitrary binaries, shell metacharacters,
// and the empty command are all rejected BEFORE any spawn.
func TestIsAllowedCommandRejects(t *testing.T) {
	rejects := []string{
		"rm",            // not in the allowlist
		"bash",          // not in the allowlist
		"bash -c",       // not in the allowlist + space
		"sh",            // not in the allowlist
		"./npx",         // relative path
		"../npx",        // relative path
		"/bin/rm",       // absolute path to an arbitrary binary
		"npx; rm -rf /", // shell metacharacter
		"npx && rm",     // shell metacharacter
		"npx | cat",     // shell metacharacter
		"$(touch x)",    // command substitution
		"`id`",          // backtick command substitution
		"npx\nrm",       // embedded newline
		"npx<in",        // redirection metacharacter
		"npx>out",       // redirection metacharacter
		"npx (x)",       // subshell metacharacters
		"",              // empty command
		"   ",           // whitespace only
		"NPX",           // case-sensitive: not allowlisted
		"/usr/bin/../bin/rm", // non-clean absolute path
	}
	for _, cmd := range rejects {
		if isAllowedCommand(cmd) {
			t.Errorf("isAllowedCommand(%q) = true, want false (security boundary)", cmd)
		}
	}
}
