package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// MCP protocol constants, ported verbatim from the 9router probe/registry refs.
const (
	mcpProtocolVersion = "2025-06-18"                                          // cowork-mcp-tools/route.js:13 (PAR-MCP-010)
	probeTimeout       = 8 * time.Second                                       // cowork-mcp-tools/route.js:5  (PAR-MCP-058)
	registryURL        = "https://api.anthropic.com/mcp-registry/v0/servers"   // cowork-mcp-registry/route.js:5
	registryVisibility = "commercial,gsuite,gsuite-google"                     // cowork-mcp-registry/route.js:6
	registryPageLimit  = 500                                                   // cowork-mcp-registry/route.js:28
	registryMaxPages   = 20                                                    // cowork-mcp-registry/route.js:27 (PAR-MCP-014)
	registryCacheTTL   = 1 * time.Hour                                         // cowork-mcp-registry/route.js:7  (PAR-MCP-015)
)

// ProbeTool is one tool advertised by an MCP server.
type ProbeTool struct {
	Name        string
	Description string
}

// ProbeResult is the outcome of a probe handshake. Error is "" on success,
// "timeout" on a deadline, or "init <status>"/<msg> otherwise.
type ProbeResult struct {
	Tools        []ProbeTool
	RequiresAuth bool   // 401/403 detected (PAR-MCP-013)
	Error        string // "timeout" | "init <status>" | <msg>
}

// Probe performs the MCP handshake (initialize → notifications/initialized →
// tools/list) over an injectable *http.Client (mirrors internal/auth/oauth.go:128).
type Probe struct {
	client *http.Client
}

// NewProbe builds a Probe. A nil client falls back to the package default.
func NewProbe(client *http.Client) *Probe {
	if client == nil {
		client = defaultHTTPClient()
	}
	return &Probe{client: client}
}

// Run probes url with the three-step handshake, honoring the 8s timeout via ctx.
// Mirrors cowork-mcp-tools/route.js.
func (p *Probe) Run(ctx context.Context, url string) ProbeResult {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	// 1. initialize (id 1).
	initBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"` +
		mcpProtocolVersion + `","capabilities":{},"clientInfo":{"name":"g0router","version":"1"}}}`
	status, header, _, err := p.post(ctx, url, "", []byte(initBody))
	if err != nil {
		return errResult(err)
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return ProbeResult{RequiresAuth: true}
	}
	if status < 200 || status >= 300 {
		return ProbeResult{Error: fmt.Sprintf("init %d", status)}
	}
	sessionID := header.Get("mcp-session-id")

	// 2. notifications/initialized (best-effort; errors swallowed).
	initializedBody := `{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`
	_, _, _, _ = p.post(ctx, url, sessionID, []byte(initializedBody))

	// 3. tools/list (id 2).
	toolsBody := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`
	status, header, body, err := p.post(ctx, url, sessionID, []byte(toolsBody))
	if err != nil {
		return errResult(err)
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return ProbeResult{RequiresAuth: true}
	}
	if status < 200 || status >= 300 {
		return ProbeResult{Error: fmt.Sprintf("tools/list %d", status)}
	}

	tools, err := extractTools(header.Get("Content-Type"), body)
	if err != nil {
		return ProbeResult{Error: err.Error()}
	}
	return ProbeResult{Tools: tools}
}

// post issues one JSON-RPC POST with the MCP headers, replaying sessionID when set.
func (p *Probe) post(ctx context.Context, url, sessionID string, payload []byte) (int, http.Header, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", mcpProtocolVersion)
	if sessionID != "" {
		req.Header.Set("mcp-session-id", sessionID)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, nil, nil, err
	}
	return resp.StatusCode, resp.Header, body, nil
}

// errResult maps a transport error to a ProbeResult, recognizing the deadline.
func errResult(err error) ProbeResult {
	if errors.Is(err, context.DeadlineExceeded) {
		return ProbeResult{Error: "timeout"}
	}
	return ProbeResult{Error: err.Error()}
}

// toolsListResult is the JSON-RPC shape of a tools/list response.
type toolsListResult struct {
	ID     int `json:"id"`
	Result struct {
		Tools []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"tools"`
	} `json:"result"`
}

// extractTools parses a tools/list response body. For text/event-stream it scans
// the data frames for the id==2 result (PAR-MCP-012); otherwise it unmarshals JSON.
func extractTools(contentType string, body []byte) ([]ProbeTool, error) {
	if strings.Contains(contentType, "text/event-stream") {
		for _, frame := range parseSSEDataFrames(string(body)) {
			var r toolsListResult
			if err := json.Unmarshal([]byte(frame), &r); err != nil {
				continue
			}
			if r.ID == 2 {
				return mapTools(r), nil
			}
		}
		return nil, nil
	}
	var r toolsListResult
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("decode tools/list: %w", err)
	}
	return mapTools(r), nil
}

func mapTools(r toolsListResult) []ProbeTool {
	out := make([]ProbeTool, 0, len(r.Result.Tools))
	for _, t := range r.Result.Tools {
		out = append(out, ProbeTool{Name: t.Name, Description: t.Description})
	}
	return out
}
