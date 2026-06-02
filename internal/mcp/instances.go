package mcp

import (
	"errors"
	"strings"
)

const RedactedValue = "********"

type LaunchType string

const (
	LaunchCommand LaunchType = "command"
	LaunchNPX     LaunchType = "npx"
	LaunchDocker  LaunchType = "docker"
	LaunchHTTP    LaunchType = "http"
)

var ErrInvalidInstanceConfig = errors.New("mcp: invalid instance config")

type InstanceConfig struct {
	ID           string
	Name         string
	ServerKey    string
	LaunchType   LaunchType
	Transport    Transport
	Command      string
	Args         []string
	URL          string
	Headers      map[string]string
	Env          map[string]string
	CWD          string
	AccountLabel string
}

func (c InstanceConfig) Validate() error {
	if c.Name == "" || c.ServerKey == "" {
		return ErrInvalidInstanceConfig
	}
	if !validLaunchType(c.LaunchType) || !validTransport(c.Transport) {
		return ErrInvalidInstanceConfig
	}
	switch c.LaunchType {
	case LaunchCommand, LaunchNPX, LaunchDocker:
		if c.Transport != TransportStdio {
			return ErrInvalidInstanceConfig
		}
	case LaunchHTTP:
		if c.Transport != TransportStreamableHTTP && c.Transport != TransportSSE {
			return ErrInvalidInstanceConfig
		}
		if c.URL == "" {
			return ErrInvalidInstanceConfig
		}
	}
	return nil
}

func (c InstanceConfig) Redacted() InstanceConfig {
	c.Env = redactSecretMap(c.Env)
	c.Headers = redactSecretMap(c.Headers)
	return c
}

func validLaunchType(value LaunchType) bool {
	switch value {
	case LaunchCommand, LaunchNPX, LaunchDocker, LaunchHTTP:
		return true
	default:
		return false
	}
}

func validTransport(value Transport) bool {
	switch value {
	case TransportStdio, TransportStreamableHTTP, TransportSSE:
		return true
	default:
		return false
	}
}

func redactSecretMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	redacted := make(map[string]string, len(values))
	for key, value := range values {
		if isSecretKey(key) {
			redacted[key] = RedactedValue
			continue
		}
		redacted[key] = value
	}
	return redacted
}

func isSecretKey(key string) bool {
	normalized := strings.ToLower(key)
	for _, marker := range []string{"token", "secret", "key", "authorization", "password"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
