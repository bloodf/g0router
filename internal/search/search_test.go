package search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/store"
)

func TestKagiSearchTool(t *testing.T) {
	var gotAuth string
	var gotRequest struct {
		Query    string `json:"query"`
		Workflow string `json:"workflow"`
		Limit    int    `json:"limit"`
		Filters  struct {
			After string `json:"after"`
		} `json:"filters"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"meta": map[string]any{"trace": "kagi-request"},
			"data": map[string]any{
				"search": []map[string]any{{"title": "Result", "url": "https://example.test", "snippet": "Snippet"}},
				"related_search": []map[string]any{
					{"query": "related one"},
					{"title": "related two"},
				},
				"direct_answer": []map[string]any{{"title": "Answer", "snippet": "Direct answer"}},
			},
		})
	}))
	defer server.Close()

	client := NewKagiClient(Config{KagiBaseURL: server.URL, HTTPClient: server.Client()}, "kagi-key")
	result, err := client.CallTool(context.Background(), mcp.CallRequest{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"router docs","limit":2,"recency":"week"}`),
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	if gotAuth != "Bearer kagi-key" {
		t.Fatalf("Authorization = %q, want Bearer key", gotAuth)
	}
	if gotRequest.Query != "router docs" || gotRequest.Workflow != "search" || gotRequest.Limit != 2 {
		t.Fatalf("request = %+v, want query/workflow/limit", gotRequest)
	}
	if gotRequest.Filters.After == "" {
		t.Fatalf("filters.after = empty, want recency date")
	}
	content, ok := result.Content.(SearchResult)
	if !ok {
		t.Fatalf("content = %T, want SearchResult", result.Content)
	}
	if content.Provider != ProviderKagi || content.RequestID != "kagi-request" {
		t.Fatalf("content = %+v, want kagi request", content)
	}
	if len(content.Results) != 1 || content.Results[0].URL != "https://example.test" || content.Results[0].Snippet != "Snippet" {
		t.Fatalf("results = %+v, want normalized Kagi result", content.Results)
	}
	if content.Answer != "Answer: Direct answer" {
		t.Fatalf("answer = %q, want direct answer", content.Answer)
	}
	if len(content.RelatedQueries) != 2 || content.RelatedQueries[0] != "related one" {
		t.Fatalf("related = %+v, want normalized related queries", content.RelatedQueries)
	}
}

func TestTavilySearchTool(t *testing.T) {
	var gotAuth string
	var gotRequest struct {
		Query             string `json:"query"`
		SearchDepth       string `json:"search_depth"`
		MaxResults        int    `json:"max_results"`
		TimeRange         string `json:"time_range"`
		IncludeAnswer     any    `json:"include_answer"`
		IncludeRawContent bool   `json:"include_raw_content"`
		Topic             string `json:"topic"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"answer":     "The answer",
			"request_id": "tavily-request",
			"results": []map[string]any{{
				"title":          "Result",
				"url":            "https://example.test/tavily",
				"content":        "Content",
				"published_date": "2026-06-04",
			}},
		})
	}))
	defer server.Close()

	client := NewTavilyClient(Config{TavilyBaseURL: server.URL, HTTPClient: server.Client()}, "tavily-key")
	result, err := client.CallTool(context.Background(), mcp.CallRequest{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"latest ai news","limit":2,"recency":"week"}`),
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	if gotAuth != "Bearer tavily-key" {
		t.Fatalf("Authorization = %q, want Bearer key", gotAuth)
	}
	if gotRequest.Query != "latest ai news" || gotRequest.SearchDepth != "basic" || gotRequest.MaxResults != 2 || gotRequest.TimeRange != "week" {
		t.Fatalf("request = %+v, want query/max_results/time_range", gotRequest)
	}
	if gotRequest.IncludeAnswer != "advanced" || gotRequest.IncludeRawContent || gotRequest.Topic != "" {
		t.Fatalf("request = %+v, want Tavily body without topic", gotRequest)
	}
	content, ok := result.Content.(SearchResult)
	if !ok {
		t.Fatalf("content = %T, want SearchResult", result.Content)
	}
	if content.Provider != ProviderTavily || content.Answer != "The answer" || content.RequestID != "tavily-request" {
		t.Fatalf("content = %+v, want Tavily answer/request", content)
	}
	if len(content.Results) != 1 || content.Results[0].PublishedAt != "2026-06-04" {
		t.Fatalf("results = %+v, want normalized Tavily result", content.Results)
	}
}

func TestSearchToolErrorsAreSanitized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad key kagi-secret-value", http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewKagiClient(Config{KagiBaseURL: server.URL, HTTPClient: server.Client()}, "kagi-secret-value")
	_, err := client.CallTool(context.Background(), mcp.CallRequest{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"router"}`),
	})
	if err == nil {
		t.Fatal("CallTool error = nil, want sanitized upstream error")
	}
	if strings.Contains(err.Error(), "kagi-secret-value") || strings.Contains(err.Error(), "bad key") {
		t.Fatalf("error leaks upstream body or secret: %v", err)
	}
}

func TestSearchToolRequiresActiveAPIKey(t *testing.T) {
	s := openSearchStoreForTest(t)
	defer s.Close()
	tools := mcp.NewToolManager()
	key := "inactive-key"
	if err := s.CreateConnection(&store.Connection{
		Provider: "kagi",
		Name:     "inactive",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &key,
		IsActive: false,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if err := RegisterBuiltInTools(context.Background(), s, tools, Config{}); err != nil {
		t.Fatalf("RegisterBuiltInTools: %v", err)
	}
	if got := tools.CompactTools(); len(got) != 0 {
		t.Fatalf("tools = %+v, want none without active API key connection", got)
	}
}

func TestBuiltInSearchTools(t *testing.T) {
	s := openSearchStoreForTest(t)
	defer s.Close()
	tools := mcp.NewToolManager()
	kagiKey := "kagi-key"
	tavilyKey := "tavily-key"
	for _, conn := range []*store.Connection{
		{Provider: "kagi", Name: "kagi", AuthType: store.AuthTypeAPIKey, APIKey: &kagiKey, IsActive: true},
		{Provider: "tavily", Name: "tavily", AuthType: store.AuthTypeAPIKey, APIKey: &tavilyKey, IsActive: true},
	} {
		if err := s.CreateConnection(conn); err != nil {
			t.Fatalf("CreateConnection %s: %v", conn.Provider, err)
		}
	}

	if err := RegisterBuiltInTools(context.Background(), s, tools, Config{}); err != nil {
		t.Fatalf("RegisterBuiltInTools: %v", err)
	}

	got := tools.CompactTools()
	if len(got) != 2 || got[0].Function.Name != "kagi__search" || got[1].Function.Name != "tavily__search" {
		t.Fatalf("tools = %+v, want kagi/tavily search tools", got)
	}
}

func openSearchStoreForTest(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.NewStore(t.TempDir() + "/g0router.db")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}
