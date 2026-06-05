package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/store"
)

const (
	ProviderKagi   = "kagi"
	ProviderTavily = "tavily"

	defaultKagiBaseURL   = "https://kagi.com"
	defaultTavilyBaseURL = "https://api.tavily.com"
)

var (
	ErrUnsupportedTool = errors.New("search: unsupported tool")
	ErrMissingAPIKey   = errors.New("search: missing api key")
)

type Config struct {
	HTTPClient    *http.Client
	KagiBaseURL   string
	TavilyBaseURL string
}

type SearchResult struct {
	Provider       string             `json:"provider"`
	RequestID      string             `json:"request_id,omitempty"`
	Answer         string             `json:"answer,omitempty"`
	Results        []SearchResultItem `json:"results"`
	RelatedQueries []string           `json:"related_queries,omitempty"`
}

type SearchResultItem struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Snippet     string `json:"snippet,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
}

type Client struct {
	provider   string
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewKagiClient(config Config, apiKey string) *Client {
	return &Client{
		provider:   ProviderKagi,
		apiKey:     strings.TrimSpace(apiKey),
		baseURL:    normalizedBaseURL(config.KagiBaseURL, defaultKagiBaseURL),
		httpClient: normalizedHTTPClient(config.HTTPClient),
	}
}

func NewTavilyClient(config Config, apiKey string) *Client {
	return &Client{
		provider:   ProviderTavily,
		apiKey:     strings.TrimSpace(apiKey),
		baseURL:    normalizedBaseURL(config.TavilyBaseURL, defaultTavilyBaseURL),
		httpClient: normalizedHTTPClient(config.HTTPClient),
	}
}

func RegisterBuiltInTools(ctx context.Context, s *store.Store, tools *mcp.ToolManager, config Config) error {
	if s == nil || tools == nil {
		return nil
	}
	for _, provider := range []string{ProviderKagi, ProviderTavily} {
		apiKey, ok, err := activeAPIKey(s, provider)
		if err != nil {
			return err
		}
		if !ok {
			tools.UnregisterClient(provider)
			continue
		}
		var client *Client
		if provider == ProviderKagi {
			client = NewKagiClient(config, apiKey)
		} else {
			client = NewTavilyClient(config, apiKey)
		}
		if _, err := client.ListTools(ctx); err != nil {
			return err
		}
		if err := tools.RefreshManifest(mcp.Manifest{ClientID: provider, Tools: searchTools(provider)}); err != nil {
			return fmt.Errorf("register %s search tools: %w", provider, err)
		}
		tools.RegisterClient(provider, client)
	}
	return nil
}

func (c *Client) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	if c == nil || c.apiKey == "" {
		return nil, ErrMissingAPIKey
	}
	return searchTools(c.provider), nil
}

func (c *Client) CallTool(ctx context.Context, req mcp.CallRequest) (mcp.CallResult, error) {
	if c == nil || c.apiKey == "" {
		return mcp.CallResult{}, ErrMissingAPIKey
	}
	if req.Name != "search" {
		return mcp.CallResult{}, ErrUnsupportedTool
	}
	args, err := decodeArguments(req.Arguments)
	if err != nil {
		return mcp.CallResult{}, err
	}
	var result SearchResult
	switch c.provider {
	case ProviderKagi:
		result, err = c.callKagi(ctx, args)
	case ProviderTavily:
		result, err = c.callTavily(ctx, args)
	default:
		err = fmt.Errorf("search: unsupported provider %q", c.provider)
	}
	if err != nil {
		return mcp.CallResult{}, err
	}
	return mcp.CallResult{Content: result}, nil
}

func (c *Client) Close() error {
	return nil
}

type searchArguments struct {
	Query   string `json:"query"`
	Limit   int    `json:"limit"`
	Recency string `json:"recency"`
}

func (c *Client) callKagi(ctx context.Context, args searchArguments) (SearchResult, error) {
	endpoint, err := url.JoinPath(c.baseURL, "/api/v1/search")
	if err != nil {
		return SearchResult{}, fmt.Errorf("build kagi search url: %w", err)
	}
	payload := map[string]any{
		"query":    args.Query,
		"workflow": "search",
		"limit":    effectiveLimit(args.Limit),
	}
	if args.Recency != "" {
		payload["filters"] = map[string]string{"after": recencyAfterDate(args.Recency)}
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return SearchResult{}, fmt.Errorf("encode kagi search request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return SearchResult{}, fmt.Errorf("create kagi search request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SearchResult{}, fmt.Errorf("call kagi search: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		drainBody(resp.Body)
		return SearchResult{}, fmt.Errorf("kagi search returned status %d", resp.StatusCode)
	}

	var body kagiResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return SearchResult{}, fmt.Errorf("decode kagi search response: %w", err)
	}
	if len(body.Errors) > 0 {
		return SearchResult{}, fmt.Errorf("kagi search returned error")
	}
	return body.result(), nil
}

func (c *Client) callTavily(ctx context.Context, args searchArguments) (SearchResult, error) {
	endpoint, err := url.JoinPath(c.baseURL, "/search")
	if err != nil {
		return SearchResult{}, fmt.Errorf("build tavily search url: %w", err)
	}
	payload := map[string]any{
		"query":               args.Query,
		"search_depth":        "basic",
		"max_results":         effectiveLimit(args.Limit),
		"include_answer":      "advanced",
		"include_raw_content": false,
	}
	if args.Recency != "" {
		payload["time_range"] = args.Recency
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return SearchResult{}, fmt.Errorf("encode tavily search request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return SearchResult{}, fmt.Errorf("create tavily search request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SearchResult{}, fmt.Errorf("call tavily search: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		drainBody(resp.Body)
		return SearchResult{}, fmt.Errorf("tavily search returned status %d", resp.StatusCode)
	}

	var body tavilyResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return SearchResult{}, fmt.Errorf("decode tavily search response: %w", err)
	}
	return body.result(), nil
}

type kagiResponse struct {
	Meta struct {
		ID    string `json:"id"`
		Trace string `json:"trace"`
	} `json:"meta"`
	Data struct {
		Search []struct {
			URL     string `json:"url"`
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
			Date    string `json:"published"`
		} `json:"search"`
		Video []struct {
			URL     string `json:"url"`
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
			Date    string `json:"published"`
		} `json:"video"`
		News []struct {
			URL     string `json:"url"`
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
			Date    string `json:"published"`
		} `json:"news"`
		Infobox []struct {
			URL     string `json:"url"`
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
		} `json:"infobox"`
		DirectAnswer []struct {
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
		} `json:"direct_answer"`
		RelatedSearch []struct {
			Query string `json:"query"`
			Title string `json:"title"`
		} `json:"related_search"`
		AdjacentQuestion []struct {
			Query string `json:"query"`
			Title string `json:"title"`
		} `json:"adjacent_question"`
	} `json:"data"`
	Errors []struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	} `json:"error"`
}

func (r kagiResponse) result() SearchResult {
	requestID := r.Meta.Trace
	if requestID == "" {
		requestID = r.Meta.ID
	}
	result := SearchResult{Provider: ProviderKagi, RequestID: requestID}
	appendSource := func(title, urlValue, snippet, published string) {
		if urlValue == "" {
			return
		}
		result.Results = append(result.Results, SearchResultItem{Title: title, URL: urlValue, Snippet: snippet, PublishedAt: published})
	}
	for _, item := range r.Data.Search {
		appendSource(item.Title, item.URL, item.Snippet, item.Date)
	}
	for _, item := range r.Data.Video {
		appendSource(item.Title, item.URL, item.Snippet, item.Date)
	}
	for _, item := range r.Data.News {
		appendSource(item.Title, item.URL, item.Snippet, item.Date)
	}
	for _, item := range r.Data.Infobox {
		appendSource(item.Title, item.URL, item.Snippet, "")
	}
	if len(r.Data.DirectAnswer) > 0 {
		answer := strings.TrimSpace(r.Data.DirectAnswer[0].Title)
		snippet := strings.TrimSpace(r.Data.DirectAnswer[0].Snippet)
		if answer != "" && snippet != "" {
			result.Answer = answer + ": " + snippet
		} else {
			result.Answer = answer + snippet
		}
	}
	appendRelated := func(query, title string) {
		value := strings.TrimSpace(query)
		if value == "" {
			value = strings.TrimSpace(title)
		}
		if value != "" {
			result.RelatedQueries = append(result.RelatedQueries, value)
		}
	}
	for _, item := range r.Data.RelatedSearch {
		appendRelated(item.Query, item.Title)
	}
	for _, item := range r.Data.AdjacentQuestion {
		appendRelated(item.Query, item.Title)
	}
	return result
}

type tavilyResponse struct {
	Answer    string `json:"answer"`
	RequestID string `json:"request_id"`
	Results   []struct {
		Title         string `json:"title"`
		URL           string `json:"url"`
		Content       string `json:"content"`
		PublishedDate string `json:"published_date"`
	} `json:"results"`
}

func (r tavilyResponse) result() SearchResult {
	result := SearchResult{Provider: ProviderTavily, RequestID: r.RequestID, Answer: r.Answer}
	for _, item := range r.Results {
		if item.URL == "" {
			continue
		}
		result.Results = append(result.Results, SearchResultItem{
			Title:       item.Title,
			URL:         item.URL,
			Snippet:     item.Content,
			PublishedAt: item.PublishedDate,
		})
	}
	return result
}

func decodeArguments(raw json.RawMessage) (searchArguments, error) {
	var args searchArguments
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &args); err != nil {
			return args, fmt.Errorf("decode search arguments: %w", err)
		}
	}
	args.Query = strings.TrimSpace(args.Query)
	args.Recency = strings.TrimSpace(args.Recency)
	if args.Query == "" {
		return args, fmt.Errorf("search: query is required")
	}
	if args.Limit < 0 {
		return args, fmt.Errorf("search: limit must be positive")
	}
	if args.Limit > 20 {
		args.Limit = 20
	}
	return args, nil
}

func activeAPIKey(s *store.Store, provider string) (string, bool, error) {
	connections, err := s.GetActiveConnections(provider)
	if err != nil {
		return "", false, fmt.Errorf("load %s connections: %w", provider, err)
	}
	for _, conn := range connections {
		if conn.AuthType != store.AuthTypeAPIKey || conn.APIKey == nil {
			continue
		}
		key := strings.TrimSpace(*conn.APIKey)
		if key != "" {
			return key, true, nil
		}
	}
	return "", false, nil
}

func searchTools(provider string) []mcp.Tool {
	return []mcp.Tool{{
		ClientID:    provider,
		Name:        "search",
		Description: provider + " web search",
		InputSchema: json.RawMessage(`{"type":"object","required":["query"],"properties":{"query":{"type":"string"},"limit":{"type":"integer","minimum":1,"maximum":20},"recency":{"type":"string","enum":["day","week","month","year"]}},"additionalProperties":false}`),
	}}
}

func effectiveLimit(limit int) int {
	if limit <= 0 {
		return 5
	}
	return limit
}

func recencyAfterDate(recency string) string {
	now := time.Now().UTC()
	switch recency {
	case "day":
		return now.AddDate(0, 0, -1).Format("2006-01-02")
	case "week":
		return now.AddDate(0, 0, -7).Format("2006-01-02")
	case "month":
		return now.AddDate(0, -1, 0).Format("2006-01-02")
	case "year":
		return now.AddDate(-1, 0, 0).Format("2006-01-02")
	default:
		return now.AddDate(0, 0, -7).Format("2006-01-02")
	}
}

func normalizedBaseURL(value, fallback string) string {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" {
		return fallback
	}
	return value
}

func normalizedHTTPClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func drainBody(r io.Reader) {
	_, _ = io.Copy(io.Discard, io.LimitReader(r, 4096))
}
