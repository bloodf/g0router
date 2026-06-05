package search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/store"
)

// --- RegisterBuiltInTools: store error via closed store ---

func TestRegisterBuiltInToolsStoreError(t *testing.T) {
	s := openSearchStoreForCoverageTest(t)
	s.Close() // close before use to force error
	tools := mcp.NewToolManager()
	err := RegisterBuiltInTools(context.Background(), s, tools, Config{})
	if err == nil {
		t.Fatal("RegisterBuiltInTools closed store should fail")
	}
}

// --- activeAPIKey: store error ---

func TestActiveAPIKeyStoreError(t *testing.T) {
	s := openSearchStoreForCoverageTest(t)
	s.Close()
	_, _, err := activeAPIKey(s, "kagi")
	if err == nil {
		t.Fatal("activeAPIKey closed store should fail")
	}
}

// --- RegisterBuiltInTools: nil args ---

func TestRegisterBuiltInToolsNilStore(t *testing.T) {
	tools := mcp.NewToolManager()
	if err := RegisterBuiltInTools(context.Background(), nil, tools, Config{}); err != nil {
		t.Fatalf("RegisterBuiltInTools nil store: %v", err)
	}
}

func TestRegisterBuiltInToolsNilTools(t *testing.T) {
	s := openSearchStoreForCoverageTest(t)
	defer s.Close()
	if err := RegisterBuiltInTools(context.Background(), s, nil, Config{}); err != nil {
		t.Fatalf("RegisterBuiltInTools nil tools: %v", err)
	}
}

// --- ListTools: nil/empty key ---

func TestListToolsMissingAPIKey(t *testing.T) {
	c := &Client{provider: ProviderKagi, apiKey: ""}
	_, err := c.ListTools(context.Background())
	if err != ErrMissingAPIKey {
		t.Fatalf("ListTools empty key = %v, want ErrMissingAPIKey", err)
	}
}

func TestListToolsNilClient(t *testing.T) {
	var c *Client
	_, err := c.ListTools(context.Background())
	if err != ErrMissingAPIKey {
		t.Fatalf("ListTools nil client = %v, want ErrMissingAPIKey", err)
	}
}

// --- CallTool: edge cases ---

func TestCallToolMissingAPIKey(t *testing.T) {
	c := &Client{provider: ProviderKagi, apiKey: ""}
	_, err := c.CallTool(context.Background(), mcp.CallRequest{Name: "search"})
	if err != ErrMissingAPIKey {
		t.Fatalf("CallTool empty key = %v, want ErrMissingAPIKey", err)
	}
}

func TestCallToolUnsupportedToolName(t *testing.T) {
	c := NewKagiClient(Config{}, "key")
	_, err := c.CallTool(context.Background(), mcp.CallRequest{Name: "other"})
	if err != ErrUnsupportedTool {
		t.Fatalf("CallTool other name = %v, want ErrUnsupportedTool", err)
	}
}

func TestCallToolUnsupportedProvider(t *testing.T) {
	c := &Client{provider: "unknown", apiKey: "key", httpClient: http.DefaultClient}
	_, err := c.CallTool(context.Background(), mcp.CallRequest{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"test"}`),
	})
	if err == nil {
		t.Fatal("CallTool unknown provider should fail")
	}
	if !strings.Contains(err.Error(), "unsupported provider") {
		t.Fatalf("error = %v", err)
	}
}

func TestCallToolDecodeArgumentsError(t *testing.T) {
	c := NewKagiClient(Config{}, "key")
	_, err := c.CallTool(context.Background(), mcp.CallRequest{
		Name:      "search",
		Arguments: json.RawMessage(`{bad json`),
	})
	if err == nil {
		t.Fatal("CallTool bad args should fail")
	}
}

// --- Close ---

func TestClientClose(t *testing.T) {
	c := NewKagiClient(Config{}, "key")
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// --- decodeArguments edge cases ---

func TestDecodeArgumentsEmptyQuery(t *testing.T) {
	_, err := decodeArguments(json.RawMessage(`{"query":"  "}`))
	if err == nil {
		t.Fatal("decodeArguments empty query should fail")
	}
	if !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeArgumentsNegativeLimit(t *testing.T) {
	_, err := decodeArguments(json.RawMessage(`{"query":"q","limit":-1}`))
	if err == nil {
		t.Fatal("decodeArguments negative limit should fail")
	}
	if !strings.Contains(err.Error(), "limit must be positive") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeArgumentsLimitCappedAt20(t *testing.T) {
	got, err := decodeArguments(json.RawMessage(`{"query":"q","limit":99}`))
	if err != nil {
		t.Fatalf("decodeArguments over limit: %v", err)
	}
	if got.Limit != 20 {
		t.Fatalf("limit = %d, want 20", got.Limit)
	}
}

func TestDecodeArgumentsBadJSON(t *testing.T) {
	_, err := decodeArguments(json.RawMessage(`{broken`))
	if err == nil {
		t.Fatal("decodeArguments bad JSON should fail")
	}
}

func TestDecodeArgumentsNilRaw(t *testing.T) {
	_, err := decodeArguments(nil)
	if err == nil || !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("decodeArguments nil = %v, want query required", err)
	}
}

// --- recencyAfterDate ---

func TestRecencyAfterDateDay(t *testing.T) {
	got := recencyAfterDate("day")
	expected := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	if got != expected {
		t.Fatalf("recencyAfterDate day = %q, want %q", got, expected)
	}
}

func TestRecencyAfterDateMonth(t *testing.T) {
	got := recencyAfterDate("month")
	expected := time.Now().UTC().AddDate(0, -1, 0).Format("2006-01-02")
	if got != expected {
		t.Fatalf("recencyAfterDate month = %q, want %q", got, expected)
	}
}

func TestRecencyAfterDateYear(t *testing.T) {
	got := recencyAfterDate("year")
	expected := time.Now().UTC().AddDate(-1, 0, 0).Format("2006-01-02")
	if got != expected {
		t.Fatalf("recencyAfterDate year = %q, want %q", got, expected)
	}
}

func TestRecencyAfterDateDefault(t *testing.T) {
	got := recencyAfterDate("unknown")
	expected := time.Now().UTC().AddDate(0, 0, -7).Format("2006-01-02")
	if got != expected {
		t.Fatalf("recencyAfterDate default = %q, want %q", got, expected)
	}
}

// --- kagiResponse.result() edge cases ---

func TestKagiResponseResultFallsBackToMetaID(t *testing.T) {
	r := kagiResponse{}
	r.Meta.ID = "meta-id-fallback"
	// Trace is empty, should use ID
	result := r.result()
	if result.RequestID != "meta-id-fallback" {
		t.Fatalf("requestID = %q, want meta-id-fallback", result.RequestID)
	}
}

func TestKagiResponseResultIncludesVideoAndNews(t *testing.T) {
	r := kagiResponse{}
	r.Data.Video = []struct {
		URL     string `json:"url"`
		Title   string `json:"title"`
		Snippet string `json:"snippet"`
		Date    string `json:"published"`
	}{
		{URL: "https://v.example.com", Title: "Video", Snippet: "V"},
	}
	r.Data.News = []struct {
		URL     string `json:"url"`
		Title   string `json:"title"`
		Snippet string `json:"snippet"`
		Date    string `json:"published"`
	}{
		{URL: "https://n.example.com", Title: "News", Snippet: "N"},
	}
	result := r.result()
	if len(result.Results) != 2 {
		t.Fatalf("results = %d, want 2 (video+news)", len(result.Results))
	}
}

func TestKagiResponseResultIncludesInfobox(t *testing.T) {
	r := kagiResponse{}
	r.Data.Infobox = []struct {
		URL     string `json:"url"`
		Title   string `json:"title"`
		Snippet string `json:"snippet"`
	}{
		{URL: "https://info.example.com", Title: "Infobox", Snippet: "Info"},
	}
	result := r.result()
	if len(result.Results) != 1 || result.Results[0].URL != "https://info.example.com" {
		t.Fatalf("results = %+v, want infobox", result.Results)
	}
}

func TestKagiResponseResultAnswerSnippetOnly(t *testing.T) {
	r := kagiResponse{}
	r.Data.DirectAnswer = []struct {
		Title   string `json:"title"`
		Snippet string `json:"snippet"`
	}{
		{Title: "", Snippet: "Just snippet"},
	}
	result := r.result()
	if result.Answer != "Just snippet" {
		t.Fatalf("answer = %q, want Just snippet", result.Answer)
	}
}

func TestKagiResponseResultAdjacentQuestions(t *testing.T) {
	r := kagiResponse{}
	r.Data.AdjacentQuestion = []struct {
		Query string `json:"query"`
		Title string `json:"title"`
	}{
		{Query: "", Title: "Adjacent title"},
		{Query: "Adjacent query", Title: ""},
	}
	result := r.result()
	if len(result.RelatedQueries) != 2 {
		t.Fatalf("related = %d, want 2", len(result.RelatedQueries))
	}
	if result.RelatedQueries[0] != "Adjacent title" {
		t.Fatalf("related[0] = %q, want title fallback", result.RelatedQueries[0])
	}
}

func TestKagiResponseResultSkipsEmptyURL(t *testing.T) {
	r := kagiResponse{}
	r.Data.Search = []struct {
		URL     string `json:"url"`
		Title   string `json:"title"`
		Snippet string `json:"snippet"`
		Date    string `json:"published"`
	}{
		{URL: "", Title: "No URL"},
	}
	result := r.result()
	if len(result.Results) != 0 {
		t.Fatalf("results = %d, want 0 for empty URL", len(result.Results))
	}
}

// --- tavilyResponse.result() edge cases ---

func TestTavilyResponseResultSkipsEmptyURL(t *testing.T) {
	r := tavilyResponse{
		Answer: "some answer",
		Results: []struct {
			Title         string `json:"title"`
			URL           string `json:"url"`
			Content       string `json:"content"`
			PublishedDate string `json:"published_date"`
		}{
			{URL: "", Title: "No URL"},
			{URL: "https://good.example.com", Title: "Good"},
		},
	}
	result := r.result()
	if len(result.Results) != 1 || result.Results[0].URL != "https://good.example.com" {
		t.Fatalf("results = %+v, want only good URL", result.Results)
	}
}

// --- Kagi: error status and JSON decode error ---

func TestKagiSearchReturnsErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	c := NewKagiClient(Config{KagiBaseURL: server.URL, HTTPClient: server.Client()}, "key")
	_, err := c.CallTool(context.Background(), mcp.CallRequest{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"test"}`),
	})
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("error = %v, want status 500", err)
	}
}

func TestKagiSearchResponseDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{not valid json`))
	}))
	defer server.Close()

	c := NewKagiClient(Config{KagiBaseURL: server.URL, HTTPClient: server.Client()}, "key")
	_, err := c.CallTool(context.Background(), mcp.CallRequest{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"test"}`),
	})
	if err == nil {
		t.Fatal("kagi bad JSON response should fail")
	}
}

func TestKagiSearchResponseWithErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"error": []map[string]any{{"code": "INVALID", "msg": "bad key"}},
		})
	}))
	defer server.Close()

	c := NewKagiClient(Config{KagiBaseURL: server.URL, HTTPClient: server.Client()}, "key")
	_, err := c.CallTool(context.Background(), mcp.CallRequest{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"test"}`),
	})
	if err == nil || !strings.Contains(err.Error(), "error") {
		t.Fatalf("error = %v, want kagi search returned error", err)
	}
}

// --- Tavily: error status and decode error ---

func TestTavilySearchReturnsErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer server.Close()

	c := NewTavilyClient(Config{TavilyBaseURL: server.URL, HTTPClient: server.Client()}, "key")
	_, err := c.CallTool(context.Background(), mcp.CallRequest{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"test"}`),
	})
	if err == nil || !strings.Contains(err.Error(), "status 403") {
		t.Fatalf("error = %v, want status 403", err)
	}
}

func TestTavilySearchResponseDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{not valid`))
	}))
	defer server.Close()

	c := NewTavilyClient(Config{TavilyBaseURL: server.URL, HTTPClient: server.Client()}, "key")
	_, err := c.CallTool(context.Background(), mcp.CallRequest{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"test"}`),
	})
	if err == nil {
		t.Fatal("tavily bad JSON should fail")
	}
}

// --- activeAPIKey: connection with nil APIKey ---

func TestActiveAPIKeyNilAPIKey(t *testing.T) {
	s := openSearchStoreForCoverageTest(t)
	defer s.Close()
	if err := s.CreateConnection(&store.Connection{
		Provider: "kagi",
		Name:     "nil-key",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   nil,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	key, ok, err := activeAPIKey(s, "kagi")
	if err != nil {
		t.Fatalf("activeAPIKey: %v", err)
	}
	if ok || key != "" {
		t.Fatalf("activeAPIKey nil key = %q %v, want empty false", key, ok)
	}
}

func TestActiveAPIKeyEmptyKey(t *testing.T) {
	s := openSearchStoreForCoverageTest(t)
	defer s.Close()
	empty := "   "
	if err := s.CreateConnection(&store.Connection{
		Provider: "kagi",
		Name:     "empty-key",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &empty,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	key, ok, err := activeAPIKey(s, "kagi")
	if err != nil {
		t.Fatalf("activeAPIKey: %v", err)
	}
	if ok || key != "" {
		t.Fatalf("activeAPIKey blank key = %q %v, want empty false", key, ok)
	}
}

// --- Search with no recency (empty recency branch) ---

func TestKagiSearchNoRecency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["filters"]; ok {
			// Should not have filters when no recency
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"meta": map[string]any{"trace": "t1"},
			"data": map[string]any{
				"search": []map[string]any{{"url": "https://x.com", "title": "X"}},
			},
		})
	}))
	defer server.Close()

	c := NewKagiClient(Config{KagiBaseURL: server.URL, HTTPClient: server.Client()}, "key")
	result, err := c.CallTool(context.Background(), mcp.CallRequest{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"test"}`),
	})
	if err != nil {
		t.Fatalf("kagi no recency: %v", err)
	}
	sr, ok := result.Content.(SearchResult)
	if !ok || len(sr.Results) != 1 {
		t.Fatalf("result = %+v", result)
	}
}

func TestTavilySearchNoRecency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"answer":     "",
			"request_id": "t1",
			"results":    []map[string]any{{"url": "https://x.com", "title": "X", "content": "C"}},
		})
	}))
	defer server.Close()

	c := NewTavilyClient(Config{TavilyBaseURL: server.URL, HTTPClient: server.Client()}, "key")
	result, err := c.CallTool(context.Background(), mcp.CallRequest{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"test"}`),
	})
	if err != nil {
		t.Fatalf("tavily no recency: %v", err)
	}
	sr, ok := result.Content.(SearchResult)
	if !ok || len(sr.Results) != 1 {
		t.Fatalf("result = %+v", result)
	}
}

// --- effectiveLimit ---

func TestEffectiveLimitZero(t *testing.T) {
	if got := effectiveLimit(0); got != 5 {
		t.Fatalf("effectiveLimit 0 = %d, want 5", got)
	}
}

func TestEffectiveLimitNegative(t *testing.T) {
	if got := effectiveLimit(-1); got != 5 {
		t.Fatalf("effectiveLimit -1 = %d, want 5", got)
	}
}

func TestEffectiveLimitPositive(t *testing.T) {
	if got := effectiveLimit(10); got != 10 {
		t.Fatalf("effectiveLimit 10 = %d, want 10", got)
	}
}

// --- Kagi/Tavily: network error via cancelled context ---

func TestKagiSearchNetworkError(t *testing.T) {
	// Start a server and immediately close it so httpClient.Do returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close() // closed before request

	c := NewKagiClient(Config{KagiBaseURL: serverURL, HTTPClient: &http.Client{}}, "key")
	_, err := c.CallTool(context.Background(), mcp.CallRequest{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"test"}`),
	})
	if err == nil {
		t.Fatal("kagi network error should fail")
	}
	if !strings.Contains(err.Error(), "call kagi search") {
		t.Fatalf("error = %v, want 'call kagi search'", err)
	}
}

func TestTavilySearchNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close()

	c := NewTavilyClient(Config{TavilyBaseURL: serverURL, HTTPClient: &http.Client{}}, "key")
	_, err := c.CallTool(context.Background(), mcp.CallRequest{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"test"}`),
	})
	if err == nil {
		t.Fatal("tavily network error should fail")
	}
	if !strings.Contains(err.Error(), "call tavily search") {
		t.Fatalf("error = %v, want 'call tavily search'", err)
	}
}

func openSearchStoreForCoverageTest(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.NewStore(t.TempDir() + "/g0router.db")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}
