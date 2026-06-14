package admin

import (
	"github.com/bloodf/g0router/internal/mcp"
	"github.com/valyala/fasthttp"
)

// skillDTO is the flat skill-catalog shape the frozen /skills page reads
// (§1.2): {name, category, description, url}. All-lowercase keys.
type skillDTO struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// skillsCatalog returns the static skills catalog (ESC-SKILLS-SRC: read-only
// static catalog, no store table). The catalog mirrors the w6-l seed shape — a
// set of "Endpoint Skills" entries derived from the well-known MCP servers plus
// the default plugin definitions — grouped by category on the page.
func skillsCatalog() []skillDTO {
	out := []skillDTO{
		{
			Name:        "filesystem",
			Category:    "Endpoint Skills",
			Description: "Read and write files",
			URL:         "https://github.com/modelcontextprotocol/servers/tree/main/src/filesystem",
		},
		{
			Name:        "github",
			Category:    "Endpoint Skills",
			Description: "GitHub API operations",
			URL:         "https://github.com/modelcontextprotocol/servers/tree/main/src/github",
		},
	}
	for _, p := range mcp.DefaultPlugins() {
		out = append(out, skillDTO{
			Name:        p.Name,
			Category:    "Default Plugins",
			Description: "Default MCP plugin (" + p.Transport + ")",
			URL:         p.URL,
		})
	}
	return out
}

// ListSkills handles GET /api/skills. It is a normal RequireSession route (NOT
// under /api/mcp/, so not local-only — §1.1 guard note). The data is a bare
// array of catalog entries grouped by category on the page.
func (h *Handlers) ListSkills(ctx *fasthttp.RequestCtx) {
	writeData(ctx, fasthttp.StatusOK, skillsCatalog())
}
