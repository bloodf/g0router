package mcp

import (
	"bytes"
	"context"
)

type ProcessSpec struct {
	Command string
	Args    []string
	Env     map[string]string
	CWD     string
}

type Process interface {
	Stderr() *bytes.Buffer
	Close() error
}

type ProcessRunner interface {
	Start(ctx context.Context, spec ProcessSpec) (Process, error)
}
