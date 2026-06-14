package mcp

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestIsDirectConnectAcceptReject(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://mcp.exa.ai/mcp", true},
		{"https://api.tavily.com/mcp", true},
		{"https://mcp.claude.com/foo", false},        // rejected host
		{"https://api.anthropic.com/mcp", false},      // rejected host+path
		{"https://api.anthropic.com/mcp/extra", false},
		{"https://srv.example/{tenant}/mcp", false},   // brace placeholder
		{"https://srv.example/<id>/mcp", false},       // angle placeholder
		{"http://insecure.example/mcp", false},        // non-https
		{"", false},                                    // empty
	}
	for _, c := range cases {
		if got := isDirectConnect(c.url); got != c.want {
			t.Errorf("isDirectConnect(%q) = %v, want %v", c.url, got, c.want)
		}
	}
}

// page builds a registry page JSON body with the given servers and optional cursor.
func registryPage(servers, nextCursor string) string {
	meta := `"metadata":{}`
	if nextCursor != "" {
		meta = `"metadata":{"nextCursor":"` + nextCursor + `"}`
	}
	return `{"servers":[` + servers + `],` + meta + `}`
}

// srv builds one registry item: server block + _meta registry block.
func srv(name, url, transport string, authless bool, required int) string {
	requiredFields := ""
	if required > 0 {
		requiredFields = `"requiredFields":["tenant"]`
	} else {
		requiredFields = `"requiredFields":[]`
	}
	return `{"server":{"name":"` + name + `","description":"d ` + name + `",` +
		`"remotes":[{"type":"` + transport + `","url":"` + url + `"}]},` +
		`"_meta":{"com.anthropic.api/mcp-registry":{"isAuthless":` + boolStr(authless) + `,` + requiredFields +
		`,"title":"T ` + name + `","slug":"` + name + `"}}}`
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func TestRegistryPaginationFollowsCursor(t *testing.T) {
	ft := &fakeTransport{responses: []fakeResp{
		jsonResp(registryPage(srv("a", "https://a.example/mcp", "http", true, 0), "cur1")),
		jsonResp(registryPage(srv("b", "https://b.example/sse", "sse", false, 0), "")),
	}}
	r := NewRegistry(fakeClient(ft))
	servers, err := r.List(context.Background(), false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("got %d servers, want 2: %#v", len(servers), servers)
	}
	if len(ft.captured) != 2 {
		t.Fatalf("captured %d requests, want 2", len(ft.captured))
	}
	// First request: no cursor; carries limit + visibility.
	q0 := ft.captured[0].URL.Query()
	if q0.Get("cursor") != "" {
		t.Fatalf("first request should have no cursor, got %q", q0.Get("cursor"))
	}
	if q0.Get("limit") != "500" {
		t.Fatalf("limit = %q, want 500", q0.Get("limit"))
	}
	if q0.Get("visibility") != registryVisibility {
		t.Fatalf("visibility = %q", q0.Get("visibility"))
	}
	// Second request: follows nextCursor.
	if c := ft.captured[1].URL.Query().Get("cursor"); c != "cur1" {
		t.Fatalf("second request cursor = %q, want cur1", c)
	}
	// Transport + oauth mapping.
	byName := map[string]RegistryServer{}
	for _, s := range servers {
		byName[s.Name] = s
	}
	if byName["a"].Transport != "http" || byName["a"].OAuth != false {
		t.Fatalf("a = %#v (authless → oauth false)", byName["a"])
	}
	if byName["b"].Transport != "sse" || byName["b"].OAuth != true {
		t.Fatalf("b = %#v (not authless → oauth true)", byName["b"])
	}
}

func TestRegistrySkipsDirectConnectAndRequiredFields(t *testing.T) {
	servers := strings.Join([]string{
		srv("good", "https://good.example/mcp", "http", true, 0),
		srv("direct", "https://mcp.claude.com/mcp", "http", true, 0), // isDirectConnect reject
		srv("tenant", "https://tenant.example/mcp", "http", true, 1), // requiredFields skip
	}, ",")
	ft := &fakeTransport{responses: []fakeResp{jsonResp(registryPage(servers, ""))}}
	r := NewRegistry(fakeClient(ft))
	got, err := r.List(context.Background(), false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].Name != "good" {
		t.Fatalf("got %#v, want only 'good'", got)
	}
}

func TestRegistryDedupesByURL(t *testing.T) {
	servers := strings.Join([]string{
		srv("first", "https://dup.example/mcp", "http", true, 0),
		srv("second", "https://dup.example/mcp", "http", true, 0), // same URL
		srv("third", "https://other.example/mcp", "http", true, 0),
	}, ",")
	ft := &fakeTransport{responses: []fakeResp{jsonResp(registryPage(servers, ""))}}
	r := NewRegistry(fakeClient(ft))
	got, err := r.List(context.Background(), false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d, want 2 (deduped): %#v", len(got), got)
	}
	if got[0].Name != "first" {
		t.Fatalf("dedupe should keep first occurrence, got %#v", got[0])
	}
}

func TestRegistryCacheHitMissForce(t *testing.T) {
	clk := newFakeClock(time.Unix(1_700_000_000, 0))
	ft := &fakeTransport{responses: []fakeResp{
		jsonResp(registryPage(srv("a", "https://a.example/mcp", "http", true, 0), "")),
		jsonResp(registryPage(srv("b", "https://b.example/mcp", "http", true, 0), "")),
		jsonResp(registryPage(srv("c", "https://c.example/mcp", "http", true, 0), "")),
	}}
	r := NewRegistry(fakeClient(ft))
	r.now = clk.Now

	// First call fetches.
	if _, err := r.List(context.Background(), false); err != nil {
		t.Fatalf("List 1: %v", err)
	}
	if len(ft.captured) != 1 {
		t.Fatalf("after first list: %d requests", len(ft.captured))
	}
	// Within TTL → cache hit, no second fetch.
	clk.Advance(30 * time.Minute)
	if _, err := r.List(context.Background(), false); err != nil {
		t.Fatalf("List 2: %v", err)
	}
	if len(ft.captured) != 1 {
		t.Fatalf("cache hit should not refetch, got %d requests", len(ft.captured))
	}
	// Past TTL → cache miss, refetch.
	clk.Advance(31 * time.Minute) // total 61m
	if _, err := r.List(context.Background(), false); err != nil {
		t.Fatalf("List 3: %v", err)
	}
	if len(ft.captured) != 2 {
		t.Fatalf("expired cache should refetch, got %d requests", len(ft.captured))
	}
	// force=true bypasses the cache.
	if _, err := r.List(context.Background(), true); err != nil {
		t.Fatalf("List 4: %v", err)
	}
	if len(ft.captured) != 3 {
		t.Fatalf("force should refetch, got %d requests", len(ft.captured))
	}
}

func TestRegistryStopsAtMaxPages(t *testing.T) {
	// Every page returns a cursor; the loop must cap at registryMaxPages.
	resps := make([]fakeResp, registryMaxPages+5)
	for i := range resps {
		resps[i] = jsonResp(registryPage(srv("s"+itoa(i), "https://s"+itoa(i)+".example/mcp", "http", true, 0), "next"))
	}
	ft := &fakeTransport{responses: resps}
	r := NewRegistry(fakeClient(ft))
	if _, err := r.List(context.Background(), false); err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(ft.captured) != registryMaxPages {
		t.Fatalf("captured %d pages, want %d (cap)", len(ft.captured), registryMaxPages)
	}
}

func TestNewRegistryNilClient(t *testing.T) {
	r := NewRegistry(nil)
	if r == nil || r.now == nil {
		t.Fatalf("NewRegistry(nil) must set a default client + clock")
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}
