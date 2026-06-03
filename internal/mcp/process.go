package mcp

import (
	"bytes"
	"context"
	"io"
)

type ProcessSpec struct {
	Command string
	Args    []string
	Env     map[string]string
	CWD     string
}

type Process interface {
	Stdin() io.WriteCloser
	Stdout() io.ReadCloser
	Stderr() *bytes.Buffer
	Close() error
}

type ProcessRunner interface {
	Start(ctx context.Context, spec ProcessSpec) (Process, error)
}
