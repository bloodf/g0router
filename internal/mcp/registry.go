package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RegistryServer is one entry from the Anthropic MCP registry, mirroring the
// 9router DTO (cowork-mcp-registry/route.js:40-51). mcp-3 maps this onto its
// marketplace/UI DTO (ESC-REG-DTO).
type RegistryServer struct {
	Name        string
	Slug        string
	Title       string
	Description string
	URL         string
	Transport   string // "sse" | "http"
	OAuth       bool
	ToolNames   []string
	ToolCount   int
	IconURL     string
}

// registryCacheEntry caches a fetched server list with its fetch time.
type registryCacheEntry struct {
	servers   []RegistryServer
	fetchedAt time.Time
}

// Registry fetches + caches the Anthropic MCP registry over an injectable
// *http.Client (nil → default) and an injectable clock (now, default time.Now) so
// the 1h cache TTL is unit-testable with no real sleep.
type Registry struct {
	client *http.Client
	now    func() time.Time
	mu     sync.Mutex
	cache  *registryCacheEntry
}

// NewRegistry builds a Registry. A nil client falls back to the package default;
// now defaults to time.Now.
func NewRegistry(client *http.Client) *Registry {
	if client == nil {
		client = defaultHTTPClient()
	}
	return &Registry{client: client, now: time.Now}
}

// List returns the direct-connect registry servers, paginating up to
// registryMaxPages, filtering, and deduping by URL. Within registryCacheTTL of the
// last fetch it returns the cached list (unless force is set — PAR-MCP-014/015/057).
func (r *Registry) List(ctx context.Context, force bool) ([]RegistryServer, error) {
	r.mu.Lock()
	if !force && r.cache != nil && r.now().Sub(r.cache.fetchedAt) < registryCacheTTL {
		servers := r.cache.servers
		r.mu.Unlock()
		return servers, nil
	}
	r.mu.Unlock()

	servers, err := r.fetchAll(ctx)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.cache = &registryCacheEntry{servers: servers, fetchedAt: r.now()}
	r.mu.Unlock()
	return servers, nil
}

// fetchAll runs the pagination loop, applies the filters, and dedupes by URL.
func (r *Registry) fetchAll(ctx context.Context) ([]RegistryServer, error) {
	var out []RegistryServer
	seen := make(map[string]struct{})
	cursor := ""
	for page := 0; page < registryMaxPages; page++ {
		body, err := r.fetchPage(ctx, cursor)
		if err != nil {
			return nil, err
		}
		var parsed registryResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, fmt.Errorf("decode registry page: %w", err)
		}
		for _, item := range parsed.Servers {
			rs, ok := mapRegistryItem(item)
			if !ok {
				continue
			}
			if _, dup := seen[rs.URL]; dup {
				continue
			}
			seen[rs.URL] = struct{}{}
			out = append(out, rs)
		}
		cursor = parsed.Metadata.NextCursor
		if cursor == "" {
			break
		}
	}
	return out, nil
}

// fetchPage requests one registry page and returns its raw body.
func (r *Registry) fetchPage(ctx context.Context, cursor string) ([]byte, error) {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(registryPageLimit))
	q.Set("visibility", registryVisibility)
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, registryURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("build registry request: %w", err)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch registry: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("read registry page: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("registry returned %d", resp.StatusCode)
	}
	return body, nil
}

// registryResponse is the wire shape of one registry page.
type registryResponse struct {
	Servers  []registryItem `json:"servers"`
	Metadata struct {
		NextCursor string `json:"nextCursor"`
	} `json:"metadata"`
}

type registryItem struct {
	Server struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Remotes     []struct {
			Type string `json:"type"`
			URL  string `json:"url"`
		} `json:"remotes"`
	} `json:"server"`
	Meta map[string]json.RawMessage `json:"_meta"`
}

type registryMeta struct {
	IsAuthless     bool     `json:"isAuthless"`
	RequiredFields []string `json:"requiredFields"`
	Title          string   `json:"title"`
	Slug           string   `json:"slug"`
	ToolNames      []string `json:"toolNames"`
	IconURL        string   `json:"iconUrl"`
}

// mapRegistryItem maps one registry item to a RegistryServer, returning ok=false
// when it must be skipped (no direct-connect remote or tenant-required fields).
func mapRegistryItem(item registryItem) (RegistryServer, bool) {
	if len(item.Server.Remotes) == 0 {
		return RegistryServer{}, false
	}
	remote := item.Server.Remotes[0]
	if !isDirectConnect(remote.URL) {
		return RegistryServer{}, false
	}

	var meta registryMeta
	if raw, ok := item.Meta["com.anthropic.api/mcp-registry"]; ok {
		_ = json.Unmarshal(raw, &meta)
	}
	if len(meta.RequiredFields) > 0 {
		return RegistryServer{}, false // tenant-required (PAR-MCP-017)
	}

	transport := "http"
	if remote.Type == "sse" {
		transport = "sse"
	}
	return RegistryServer{
		Name:        item.Server.Name,
		Slug:        meta.Slug,
		Title:       meta.Title,
		Description: item.Server.Description,
		URL:         remote.URL,
		Transport:   transport,
		OAuth:       !meta.IsAuthless,
		ToolNames:   meta.ToolNames,
		ToolCount:   len(meta.ToolNames),
		IconURL:     meta.IconURL,
	}, true
}

// isDirectConnect reports whether url is a directly-connectable MCP endpoint.
// PURE — mirrors cowork-mcp-registry/route.js:16-22: reject mcp.claude.com,
// api.anthropic.com/mcp, URLs containing '{' or '<', and non-https URLs
// (PAR-MCP-016).
func isDirectConnect(rawURL string) bool {
	if rawURL == "" {
		return false
	}
	if strings.Contains(rawURL, "{") || strings.Contains(rawURL, "<") {
		return false
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if u.Scheme != "https" {
		return false
	}
	if u.Host == "mcp.claude.com" {
		return false
	}
	if u.Host == "api.anthropic.com" && strings.HasPrefix(u.Path, "/mcp") {
		return false
	}
	return true
}
